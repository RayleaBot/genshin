package app

import (
	"bytes"
	"io"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// testRole is the one role of user u's account.
var testRole = map[string]any{"ref": "role", "game": "genshin", "uid": "100000001", "region": "cn_gf01", "nickname": "旅行者", "level": 60}

// fakeAccounts answers the account plugin's service for user u's account
// with testRole. Each delegation it grants is "grant:" and the task ID it was
// granted for, execute answers with data by operation, and every execute is
// recorded; a trigger's must carry a delegation.
type fakeAccounts struct {
	t        *testing.T
	data     map[string]map[string]any
	executes []executeCall
}

type executeCall struct {
	operation, delegation string
	scheduled             bool
}

func (s *fakeAccounts) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	if request.TargetPluginID != "raylea.mihoyo-accounts" || request.Service != "accounts" || request.ServiceVersion != 1 {
		s.t.Errorf("call %+v", request)
		return nil, "plugin.service_not_found"
	}
	params := request.Params
	switch request.Method {
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "旅行者", "roles": []any{testRole}, "status": "valid", "cloud_configured": []any{"genshin"}}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "roles":
		return map[string]any{"roles": []any{testRole}}, ""
	case "delegation.create":
		return map[string]any{"delegation": map[string]any{"ref": "grant:" + asText(params["task_id"]), "expires_at_ms": time.Now().Add(30 * 24 * time.Hour).UnixMilli()}}, ""
	case "execute":
		call := executeCall{operation: asText(params["operation"]), delegation: asText(params["delegation_ref"]), scheduled: scheduled}
		if scheduled && call.delegation == "" {
			s.t.Errorf("a trigger read %s without a delegation", call.operation)
		}
		s.executes = append(s.executes, call)
		return map[string]any{"operation": call.operation, "role": testRole, "data": s.data[call.operation]}, ""
	}
	// Choosing the UID in use and revoking delegations need no answer.
	return map[string]any{}, ""
}

// read reports whether a trigger executed operation with the delegation
// granted for task.
func (s *fakeAccounts) read(operation, task string) bool {
	return slices.ContainsFunc(s.executes, func(call executeCall) bool {
		return call.scheduled && call.operation == operation && call.delegation == "grant:"+task
	})
}

// dueNow makes a saved task due at its next trigger.
func dueNow(t *testing.T, a *App, ref string) {
	t.Helper()
	err := a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		(*items)[i].NextCheckMS = 0
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Every account task, created as a user or the management page creates it,
// has a job whose payload carries its task ID. The host's trigger of the job,
// which carries the payload but not the task ID, reaches that task: it reads
// with the delegation granted for it.
func TestSchedulerTriggersReachTheirAccountTasks(t *testing.T) {
	for _, c := range []struct {
		name, prefix, kind, operation string
		create                        func(h *sdkHost)
		due                           bool
	}{
		{"体力提醒", "game.reminder.genshin.", "stamina_reminder", "genshin.note", func(h *sdkHost) {
			h.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 50, "days": 30})
		}, false},
		{"体力推送", notePushTask, notePushKind, "genshin.note", func(h *sdkHost) {
			h.groupMessage("#开启体力推送", "开启体力推送")
		}, false},
		{"每日签到", "game.sign.", "signin", "genshin.sign", func(h *sdkHost) {
			h.manage("signin.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 0, "confirm": true})
		}, false},
		{"每日月报收集", "game.monthly.", "monthly", "genshin.monthly", func(h *sdkHost) {
			h.manage("monthly.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 20, "confirm": true})
		}, true},
		{"米游社任务", "game.community.", "community", "genshin.community_run", func(h *sdkHost) {
			h.manage("community.task.create", map[string]any{"account_ref": "account", "days": 30, "once": true, "read": true, "confirm": true})
		}, false},
		{"云游戏签到", "game.cloudgame.", "cloudgame", "genshin.cloud_sign", func(h *sdkHost) {
			h.manage("cloudgame.task.create", map[string]any{"account_ref": "account", "days": 30, "once": true, "confirm": true})
		}, false},
		{"挑战提醒", "game.challenge.", "challenge_reminder", "genshin.abyss", func(h *sdkHost) {
			h.manage("challenge.reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "kind": "abyss", "metric": "star", "threshold": 36, "hour": 20, "days": 30, "confirm": true})
		}, true},
		{"抽卡后台同步", "game.sync.", "gacha_sync", "genshin.gacha", func(h *sdkHost) {
			h.message("#更新抽卡记录", "更新抽卡记录")
		}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pluginApp(t)
			accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.note": {"current_resin": 160, "max_resin": 200}, "genshin.gacha": {"list": []any{}, "region": "cn_gf01"}}}
			host := newSDKHost(t, a, map[string]any{"xiaoyao": map[string]any{"is_note_task": true}}, accounts.answer)
			c.create(host)
			ref := host.job(c.prefix)
			if payload := host.jobs[ref].Payload; payload["task_id"] != ref || payload["kind"] != c.kind {
				t.Fatalf("job payload %v", payload)
			}
			if c.due {
				dueNow(t, a, ref)
			}
			if end, _ := host.trigger(ref); end["type"] != "result" {
				t.Fatalf("the trigger ended with %v", end)
			}
			if !accounts.read(c.operation, ref) {
				t.Fatalf("the trigger did not read %s for its task: %+v", c.operation, accounts.executes)
			}
		})
	}
}

// 更新抽卡记录 is answered in the chat it was sent in once its job's trigger
// has read the records.
func TestGachaSyncTriggerAnswersInTheChat(t *testing.T) {
	a := pluginApp(t)
	accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.gacha": {"list": []any{}, "region": "cn_gf01"}}}
	host := newSDKHost(t, a, nil, accounts.answer)
	if end, _ := host.message("#更新抽卡记录", "更新抽卡记录"); terminalText(end) != "抽卡记录获取中请稍等..." {
		t.Fatalf("the command ended with %v", end)
	}
	host.trigger(host.job("game.sync."))
	if len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" || !strings.Contains(sentText(host.sent[0].Message), "抽卡记录更新完成") {
		t.Fatalf("answered %+v", host.sent)
	}
}

// emptyNews answers the news lists with no posts.
type emptyNews struct{}

func (emptyNews) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte(`{"retcode":0,"data":{"list":[]}}`)))}, nil
}

// A group's push job checks that group when the host triggers it.
func TestContentPushTriggerChecksItsGroup(t *testing.T) {
	a := pluginApp(t)
	a.Content = PublicContentClient{HTTP: emptyNews{}}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	host.groupMessage("#开启公告推送", "开启公告推送")
	ref := host.job("game.content.")
	if payload := host.jobs[ref].Payload; payload["task_id"] != ref {
		t.Fatalf("job payload %v", payload)
	}
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the trigger ended with %v", end)
	}
	items, _ := a.Subscriptions.List()
	if len(items) != 1 || items[0].LastCode != "checked" || len(host.deleted) != 0 {
		t.Fatalf("subscriptions %+v, deleted %v", items, host.deleted)
	}
}

// Jobs created before their payloads carried their task IDs: the first
// trigger without one registers the job of every saved task again, with the
// same schedule and label and a payload carrying its ID, and the tasks reach
// their triggers again. A job without a saved task is left as it is, and
// later triggers without an ID register nothing more.
func TestJobsFromBeforeTaskIDsAreRegisteredAgain(t *testing.T) {
	a := pluginApp(t)
	a.Content = PublicContentClient{HTTP: emptyNews{}}
	accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.note": {"current_resin": 160, "max_resin": 200}}}
	host := newSDKHost(t, a, map[string]any{"xiaoyao": map[string]any{"is_note_task": true}}, accounts.answer)
	host.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 50, "days": 30})
	host.manage("signin.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 0, "confirm": true})
	host.manage("challenge.reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "kind": "abyss", "metric": "star", "threshold": 36, "hour": 20, "days": 30, "confirm": true})
	host.groupMessage("#开启体力推送", "开启体力推送")
	host.groupMessage("#开启公告推送", "开启公告推送")
	host.message("#更新抽卡记录", "更新抽卡记录")
	created := maps.Clone(host.jobs)
	if len(created) != 6 {
		t.Fatalf("jobs %v", created)
	}
	// Before, a job's payload held only its kind.
	for id, job := range host.jobs {
		job.Payload = map[string]any{"kind": job.Payload["kind"]}
		host.jobs[id] = job
	}
	host.jobs["game.link.OLD"] = rayleabot.SchedulerCreateRequest{TaskID: "game.link.OLD", Cron: "* * * * *", Payload: map[string]any{"kind": "gacha_link"}}
	end, actions := host.trigger("game.link.OLD")
	if end["type"] != "result" || len(actions) != len(created) || len(host.deleted) != 0 || len(host.sent) != 0 {
		t.Fatalf("the trigger ended with %v after %v", end, actions)
	}
	for id, job := range created {
		if !reflect.DeepEqual(host.jobs[id], job) {
			t.Fatalf("job %s registered as %+v, created as %+v", id, host.jobs[id], job)
		}
	}
	if job := host.jobs["game.link.OLD"]; job.Payload["task_id"] != nil {
		t.Fatalf("the link job became %+v", job)
	}
	ref := host.job("game.reminder.genshin.")
	host.trigger(ref)
	if !accounts.read("genshin.note", ref) {
		t.Fatalf("the reminder did not read with its delegation: %+v", accounts.executes)
	}
	if _, actions := host.trigger("game.link.OLD"); len(actions) != 0 {
		t.Fatalf("a later trigger asked for %v", actions)
	}
}
