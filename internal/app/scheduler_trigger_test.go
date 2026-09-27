package app

import (
	"bytes"
	"io"
	"net/http"
	"slices"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// farFuture is when the delegations fakeAccounts grants expire, past any
// test clock.
const farFuture = int64(4_102_444_800_000)

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
		return map[string]any{"delegation": map[string]any{"ref": "grant:" + asText(params["task_id"]), "expires_at_ms": farFuture}}, ""
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
// has a job without a payload. The host's trigger of the job, which names its
// task ID, reaches that task: it reads with the delegation granted for it.
func TestSchedulerTriggersReachTheirAccountTasks(t *testing.T) {
	for _, c := range []struct {
		name, prefix, operation string
		create                  func(h *sdkHost)
		due                     bool
	}{
		{"体力提醒", "game.reminder.genshin.", "genshin.note", func(h *sdkHost) {
			h.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 50, "days": 30})
		}, false},
		{"体力推送", notePushTask, "genshin.note", func(h *sdkHost) {
			h.groupMessage("#开启体力推送", "开启体力推送")
		}, false},
		{"每日签到", "game.sign.", "genshin.sign", func(h *sdkHost) {
			h.manage("signin.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 0, "confirm": true})
		}, false},
		{"每日月报收集", "game.monthly.", "genshin.monthly", func(h *sdkHost) {
			h.manage("monthly.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 20, "confirm": true})
		}, true},
		{"米游社任务", "game.community.", "genshin.community_run", func(h *sdkHost) {
			h.manage("community.task.create", map[string]any{"account_ref": "account", "days": 30, "once": true, "read": true, "confirm": true})
		}, false},
		{"云游戏签到", "game.cloudgame.", "genshin.cloud_sign", func(h *sdkHost) {
			h.manage("cloudgame.task.create", map[string]any{"account_ref": "account", "days": 30, "once": true, "confirm": true})
		}, false},
		{"挑战提醒", "game.challenge.", "genshin.abyss", func(h *sdkHost) {
			h.manage("challenge.reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "kind": "abyss", "metric": "star", "threshold": 36, "hour": 20, "days": 30, "confirm": true})
		}, true},
		{"每日抽卡同步", "game.sync.", "genshin.gacha", func(h *sdkHost) {
			h.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 0, "confirm": true})
		}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pluginApp(t)
			// Waits between pages pass on the test clock.
			a.clock = &fakeClock{at: time.Now()}
			accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.note": {"current_resin": 160, "max_resin": 200}, "genshin.gacha": {"list": []any{}, "region": "cn_gf01"}}}
			host := newSDKHost(t, a, map[string]any{"xiaoyao": map[string]any{"is_note_task": true}}, accounts.answer)
			c.create(host)
			ref := host.job(c.prefix)
			if payload := host.jobs[ref].Payload; payload != nil {
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
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the trigger ended with %v", end)
	}
	items, _ := a.Subscriptions.List()
	if len(items) != 1 || items[0].LastCode != "checked" || len(host.deleted) != 0 {
		t.Fatalf("subscriptions %+v, deleted %v", items, host.deleted)
	}
}

// A trigger of a task this plugin does not keep, as the jobs earlier
// versions left behind, deletes its own job and does nothing else.
func TestATriggerOfAnUnknownTaskDeletesItsJob(t *testing.T) {
	for _, id := range []string{"game.link.OLD", "game.pay.OLD", "game.sync.genshin.OLD", "game.reminder.genshin.OLD", "game.content.genshin.OLD", notePushTask + "OLD", "game.other"} {
		t.Run(id, func(t *testing.T) {
			a := pluginApp(t)
			host := newSDKHost(t, a, map[string]any{"xiaoyao": map[string]any{"is_note_task": true}}, (&fakeAccounts{t: t}).answer)
			host.jobs[id] = rayleabot.SchedulerCreateRequest{TaskID: id, Cron: "* * * * *", Payload: map[string]any{"kind": "old"}}
			end, actions := host.trigger(id)
			if end["type"] != "result" || len(host.deleted) != 1 || host.deleted[0] != id || len(actions) != 1 {
				t.Fatalf("the trigger ended with %v after %v, deleted %v", end, actions, host.deleted)
			}
		})
	}
}

// Removing a task, as its user in chat or on the management page, deletes its
// job in the same event.
func TestRemovingATaskDeletesItsJob(t *testing.T) {
	ref := func(h *sdkHost, prefix string) map[string]any { return map[string]any{"ref": h.job(prefix)} }
	for _, c := range []struct {
		name, prefix   string
		create, remove func(h *sdkHost)
	}{
		{"体力提醒", "game.reminder.genshin.", func(h *sdkHost) {
			h.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 50, "days": 30})
		}, func(h *sdkHost) { h.manage("reminder.remove", ref(h, "game.reminder.genshin.")) }},
		{"体力推送", notePushTask, func(h *sdkHost) {
			h.groupMessage("#开启体力推送", "开启体力推送")
		}, func(h *sdkHost) { h.groupMessage("#关闭体力推送", "关闭体力推送") }},
		{"每日签到", "game.sign.", func(h *sdkHost) {
			h.message("#自动签到 开启", "自动签到", "开启")
		}, func(h *sdkHost) { h.message("#自动签到 关闭", "自动签到", "关闭") }},
		{"每日月报收集", "game.monthly.", func(h *sdkHost) {
			h.manage("monthly.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 20, "confirm": true})
		}, func(h *sdkHost) { h.manage("monthly.task.remove", ref(h, "game.monthly.")) }},
		{"米游社任务", "game.community.", func(h *sdkHost) {
			h.manage("community.task.create", map[string]any{"account_ref": "account", "days": 30, "once": true, "read": true, "confirm": true})
		}, func(h *sdkHost) { h.manage("community.task.remove", ref(h, "game.community.")) }},
		{"云游戏签到", "game.cloudgame.", func(h *sdkHost) {
			h.manage("cloudgame.task.create", map[string]any{"account_ref": "account", "days": 30, "once": true, "confirm": true})
		}, func(h *sdkHost) { h.manage("cloudgame.task.remove", ref(h, "game.cloudgame.")) }},
		{"挑战提醒", "game.challenge.", func(h *sdkHost) {
			h.manage("challenge.reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "kind": "abyss", "metric": "star", "threshold": 36, "hour": 20, "days": 30, "confirm": true})
		}, func(h *sdkHost) { h.manage("challenge.reminder.remove", ref(h, "game.challenge.")) }},
		{"每日抽卡同步", "game.sync.", func(h *sdkHost) {
			h.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 8, "confirm": true})
		}, func(h *sdkHost) { h.manage("gacha.task.remove", ref(h, "game.sync.")) }},
		{"群推送", "game.content.", func(h *sdkHost) {
			h.groupMessage("#开启公告推送", "开启公告推送")
		}, func(h *sdkHost) {
			input := ref(h, "game.content.")
			input["confirm"] = true
			h.manage("content.subscription.remove", input)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pluginApp(t)
			host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
			c.create(host)
			id := host.job(c.prefix)
			c.remove(host)
			if len(host.deleted) != 1 || host.deleted[0] != id || len(host.jobs) != 0 {
				t.Fatalf("deleted %v, jobs left %v", host.deleted, host.jobs)
			}
		})
	}
}

// A trigger whose task is no longer saved deletes its own job.
func TestATriggerWhoseTaskIsGoneDeletesItsJob(t *testing.T) {
	for _, c := range []struct {
		name, prefix string
		create       func(h *sdkHost)
		forget       func(a *App, ref string) error
	}{
		{"体力提醒", "game.reminder.genshin.", func(h *sdkHost) {
			h.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 50, "days": 30})
		}, forgetReminder},
		{"体力推送", notePushTask, func(h *sdkHost) {
			h.groupMessage("#开启体力推送", "开启体力推送")
		}, forgetReminder},
		{"每日抽卡同步", "game.sync.", func(h *sdkHost) {
			h.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 8, "confirm": true})
		}, func(a *App, ref string) error {
			return a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
				*items = slices.Delete(*items, i, i+1)
				return nil
			})
		}},
		{"群推送", "game.content.", func(h *sdkHost) {
			h.groupMessage("#开启公告推送", "开启公告推送")
		}, func(a *App, ref string) error {
			return a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
				*items = slices.Delete(*items, i, i+1)
				return nil
			})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pluginApp(t)
			host := newSDKHost(t, a, map[string]any{"xiaoyao": map[string]any{"is_note_task": true}}, (&fakeAccounts{t: t}).answer)
			c.create(host)
			ref := host.job(c.prefix)
			if err := c.forget(a, ref); err != nil {
				t.Fatal(err)
			}
			if end, _ := host.trigger(ref); end["type"] != "result" {
				t.Fatalf("the trigger ended with %v", end)
			}
			if len(host.deleted) != 1 || host.deleted[0] != ref {
				t.Fatalf("deleted %v", host.deleted)
			}
		})
	}
}

// forgetReminder removes a saved reminder or account task, leaving its job.
func forgetReminder(a *App, ref string) error {
	return a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
}
