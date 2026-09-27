package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// cutFirstRead answers the account plugin's service as accounts does, except
// that the first read of a trigger is answered only after the trigger's
// event has ended.
func cutFirstRead(accounts *fakeAccounts, clock *fakeClock) func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
	cut := false
	return func(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
		answer, failure := accounts.answer(request, scheduled)
		if request.Method == "execute" && scheduled && !cut {
			cut = true
			clock.set(clock.Now().Add(time.Minute))
			return nil, "plugin.event_timeout"
		}
		return answer, failure
	}
}

// An account task whose read its trigger's event ended records no failure
// and is not left half-done: the next trigger reads again.
func TestAccountTaskCutByItsEventIsReadAgain(t *testing.T) {
	for _, c := range []struct {
		name, prefix, operation string
		create                  func(h *sdkHost)
		due                     bool
	}{
		{"体力提醒", "game.reminder.genshin.", "genshin.note", func(h *sdkHost) {
			h.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 50, "days": 30})
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
	} {
		t.Run(c.name, func(t *testing.T) {
			a := pluginApp(t)
			start := time.Unix(1_800_000_000, 0)
			clock := &fakeClock{at: start}
			a.clock = clock
			accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.note": {"current_resin": 160, "max_resin": 200}, "genshin.sign": {"signed": true, "outcome": "signed"}, "genshin.community_run": {"can_get_points": "0"}}}
			host := newSDKHost(t, a, nil, cutFirstRead(accounts, clock))
			c.create(host)
			ref := host.job(c.prefix)
			if c.due {
				dueNow(t, a, ref)
			}
			clock.set(start.Add(time.Minute))
			if end, _ := host.trigger(ref); end["type"] != "result" {
				t.Fatalf("the cut trigger ended with %v", end)
			}
			tasks, _ := a.Reminders.List()
			if len(tasks) != 1 || !tasks[0].Enabled || tasks[0].LastCode != "" && tasks[0].LastCode != "plugin.game_task_unfinished" || tasks[0].Community.State == "failed" {
				t.Fatalf("after the cut trigger: %+v", tasks)
			}
			clock.set(start.Add(3 * time.Minute))
			if end, _ := host.trigger(ref); end["type"] != "result" {
				t.Fatalf("the next trigger ended with %v", end)
			}
			reads := 0
			for _, call := range accounts.executes {
				if call.operation == c.operation && call.delegation == "grant:"+ref {
					reads++
				}
			}
			if reads != 2 {
				t.Fatalf("read %d times: %+v", reads, accounts.executes)
			}
		})
	}
}

// A 体力推送 whose read its trigger's event ended, or ended past the
// trigger's budget, pushes nothing and marks nothing: the next trigger reads
// again and pushes.
func TestNotePushLeftByItsEventIsPushedByTheNextTrigger(t *testing.T) {
	for name, delay := range map[string]time.Duration{"cut": time.Minute, "past the budget": 45 * time.Second} {
		t.Run(name, func(t *testing.T) {
			a := pluginApp(t)
			start := time.Unix(1_800_000_000, 0)
			clock := &fakeClock{at: start}
			a.clock = clock
			accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.note": {"current_resin": 160, "max_resin": 200}}}
			slow := false
			host := newSDKHost(t, a, map[string]any{"xiaoyao": map[string]any{"is_note_task": true}}, func(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
				answer, failure := accounts.answer(request, scheduled)
				if request.Method == "execute" && scheduled && !slow {
					slow = true
					clock.set(clock.Now().Add(delay))
					if delay >= chatTaskLimit {
						return nil, "plugin.event_timeout"
					}
				}
				return answer, failure
			})
			host.groupMessage("#开启体力推送", "开启体力推送")
			ref := host.job(notePushTask)
			sent := len(host.sent)
			clock.set(start.Add(time.Minute))
			host.trigger(ref)
			if tasks, _ := a.Reminders.List(); len(host.sent) != sent || len(tasks) != 1 || tasks[0].LastAttemptMS != 0 || tasks[0].LastCheckedMS != 0 {
				t.Fatalf("after the first trigger: %d messages, %+v", len(host.sent)-sent, tasks)
			}
			clock.set(start.Add(10 * time.Minute))
			host.trigger(ref)
			if len(host.sent) != sent+1 || host.sent[sent].TargetType != "group" || !strings.Contains(sentText(host.sent[sent].Message), "体力快满了") {
				t.Fatalf("pushed %+v", host.sent[sent:])
			}
		})
	}
}

// recentNews serves a group push one recent announcement, 1001, whose
// details the first time are answered only after the trigger's event ended.
type recentNews struct {
	clock *fakeClock
	cut   bool
}

func (n *recentNews) Do(request *http.Request) (*http.Response, error) {
	data := map[string]any{"list": []any{}}
	switch {
	case strings.Contains(request.URL.Path, "getNewsList"):
		data["list"] = []any{map[string]any{"post": map[string]any{"post_id": "1001", "subject": "版本更新说明", "created_at": time.Now().Unix() - 60}}}
	case strings.Contains(request.URL.Path, "getPostFull"):
		if !n.cut {
			n.cut = true
			n.clock.set(n.clock.Now().Add(time.Minute))
			return nil, errors.New("timeout")
		}
		data = map[string]any{"post": map[string]any{"post": map[string]any{"post_id": "1001", "subject": "版本更新说明"}}}
	}
	raw, _ := json.Marshal(map[string]any{"retcode": 0, "data": data})
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(raw))}, nil
}

// A group push its trigger's event ended before the post was sent keeps the
// post unsent, and the next trigger pushes it.
func TestContentPushLeftByItsEventIsPushedByTheNextTrigger(t *testing.T) {
	a := pluginApp(t)
	start := time.Unix(1_800_000_000, 0)
	clock := &fakeClock{at: start}
	a.clock = clock
	a.Content = PublicContentClient{HTTP: &recentNews{clock: clock}}
	host := newSDKHost(t, a, map[string]any{"image_replies": false}, (&fakeAccounts{t: t}).answer)
	host.groupMessage("#开启公告推送", "开启公告推送")
	ref := host.job("game.content.")
	sent := len(host.sent)
	clock.set(start.Add(time.Minute))
	host.trigger(ref)
	if items, _ := a.Subscriptions.List(); len(host.sent) != sent || len(items) != 1 || len(items[0].Sent) != 0 || items[0].LastCode != "plugin.game_task_unfinished" {
		t.Fatalf("after the cut trigger: %d messages, %+v", len(host.sent)-sent, items)
	}
	clock.set(start.Add(5 * time.Minute))
	host.trigger(ref)
	if len(host.sent) != sent+1 || host.sent[sent].TargetID != "g" || !strings.Contains(sentText(host.sent[sent].Message), "版本更新说明") {
		t.Fatalf("pushed %+v", host.sent[sent:])
	}
	if items, _ := a.Subscriptions.List(); len(items[0].Sent) != 1 || items[0].LastCode != "checked" {
		t.Fatalf("after the push: %+v", items)
	}
}
