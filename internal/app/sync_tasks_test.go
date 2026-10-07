package app

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/gacha"
)

// gachaPages answers the accounts plugin's wish history pages for user u's
// role: characters character event records, IDs 5000-n newest first, and
// none in the other pools. Pages are read with delegation by a trigger when
// it is set, and as the user otherwise. asked is when each page was asked
// for, and before, when set, runs before page n is answered.
type gachaPages struct {
	t          *testing.T
	clock      *fakeClock
	characters int
	delegation string
	asked      []time.Time
	before     func(n int)
}

func (p *gachaPages) service(accounts *fakeAccounts) func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
	return func(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
		if request.Method != "execute" || request.Params["operation"] != "genshin.gacha" {
			return accounts.answer(request, scheduled)
		}
		// An event keeps its origin in the background: only a trigger reads
		// with a delegation.
		if scheduled != (p.delegation != "") || asText(request.Params["delegation_ref"]) != p.delegation {
			p.t.Errorf("read %+v", request.Params)
		}
		p.asked = append(p.asked, p.clock.Now())
		if p.before != nil {
			p.before(len(p.asked))
		}
		input := asObject(request.Params["input"])
		records := []any{}
		if input["gacha_type"] == "301" {
			first := 1
			if end, _ := strconv.Atoi(asText(input["end_id"])); end != 0 {
				first = 5000 - end + 1
			}
			for n := first; n <= p.characters && len(records) < 20; n++ {
				at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(1000-n) * time.Second).Format(time.DateTime)
				records = append(records, map[string]any{"uid": "100000001", "gacha_type": "301", "item_id": "10000046", "count": "1", "time": at, "name": "胡桃", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(5000 - n)})
			}
		}
		return map[string]any{"operation": "genshin.gacha", "role": testRole, "data": map[string]any{"list": records, "region": "cn_gf01"}}, ""
	}
}

// syncHost runs the plugin through the SDK with user u's account, whose 45
// character event records take three pages, on a test clock.
func syncHost(t *testing.T) (*App, *sdkHost, *gachaPages) {
	t.Helper()
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	pages := &gachaPages{t: t, clock: clock, characters: 45}
	return a, newSDKHost(t, a, nil, pages.service(&fakeAccounts{t: t})), pages
}

// A second at least between two pages, as 米游社's rate limits want.
func pagedASecondApart(t *testing.T, asked []time.Time) {
	t.Helper()
	for i := 1; i < len(asked); i++ {
		if asked[i].Sub(asked[i-1]) < time.Second {
			t.Fatalf("pages asked at %v", asked)
		}
	}
}

// 更新抽卡记录, run through the SDK as the host runs it: after the role is
// chosen the event moves to the background, says it is reading, reads every
// page of every pool a second apart as the user and answers with Yunzai's
// report; a full read is announced turned off.
func TestGachaSyncCommandReadsEveryPageInItsBackgroundEvent(t *testing.T) {
	a, host, pages := syncHost(t)
	host.message("#设置全量更新抽卡记录", "设置全量更新抽卡记录")
	end, actions := host.message("#更新抽卡记录", "更新抽卡记录")
	moved := slices.IndexFunc(actions, func(action hostAction) bool { return action.Name == "event.detach" })
	read := slices.IndexFunc(actions, func(action hostAction) bool { return asObject(action.Data["params"])["operation"] == "genshin.gacha" })
	if end["type"] != "result" || moved < 0 || read < moved || len(host.jobs) != 0 {
		t.Fatalf("the command ended with %v after %+v, jobs %v", end, actions, host.jobs)
	}
	if len(pages.asked) != 8 {
		t.Fatalf("read %d pages", len(pages.asked))
	}
	pagedASecondApart(t, pages.asked)
	if len(host.sent) != 3 || sentText(host.sent[0].Message) != "抽卡记录获取中请稍等..." || !strings.HasPrefix(sentText(host.sent[1].Message), "[角色]记录获取成功，更新45条") || sentText(host.sent[2].Message) != "已关闭全量更新抽卡记录" {
		t.Fatalf("answered %+v", host.sent)
	}
	if archive, err := a.Gacha.Read("100000001", "cn_gf01"); err != nil || len(archive.Records) != 45 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
	if items := a.BackgroundSyncs.list(); len(items) != 1 || items[0].State != "completed" || !items[0].Full || items[0].Progress.Result == nil || items[0].Progress.Result.Added != 45 {
		t.Fatalf("syncs %+v", items)
	}
}

// The management page's background sync: the page is answered with the
// running sync as the action moves to the background, and the same event
// reads the role's records; the page then lists the completed sync.
func TestGachaSyncFromTheManagementPage(t *testing.T) {
	a, host, pages := syncHost(t)
	end, actions := host.manage("gacha.task.start", map[string]any{"account_ref": "account", "role_ref": "role", "full": false, "confirm": true})
	moved, ok := detached(actions)
	if task := asObject(asObject(moved.Data["result"])["task"]); !ok || task["state"] != "running" || asObject(task["role"])["uid"] != "100000001" {
		t.Fatalf("the page was answered with %+v", moved)
	}
	if task := asObject(asObject(end["data"])["task"]); end["type"] != "result" || task["state"] != "completed" || len(pages.asked) != 8 {
		t.Fatalf("the action ended with %v after %d pages", end, len(pages.asked))
	}
	pagedASecondApart(t, pages.asked)
	listed, _ := host.manage("gacha.task.list", nil)
	items := asList(asObject(listed["data"])["items"])
	if len(items) != 1 || asObject(items[0])["state"] != "completed" || asObject(asObject(asObject(items[0])["progress"])["result"])["added"] != float64(45) {
		t.Fatalf("listed %+v", listed)
	}
	if archive, err := a.Gacha.Read("100000001", "cn_gf01"); err != nil || len(archive.Records) != 45 || len(host.sent) != 0 {
		t.Fatalf("kept %d records, %v; sent %+v", len(archive.Records), err, host.sent)
	}
}

// A background sync the host will not move to the background, while the
// plugin holds its limit of background events, answers the page clearly and
// reads nothing.
func TestGachaSyncRefusedTheBackgroundAnswersThePage(t *testing.T) {
	a, host, pages := syncHost(t)
	host.busy = true
	end, _ := host.manage("gacha.task.start", map[string]any{"account_ref": "account", "role_ref": "role", "confirm": true})
	if end["type"] != "error" || end["code"] != "plugin.game_background_busy" || end["message"] != "正在后台处理的任务较多，请稍后再试。" {
		t.Fatalf("the action ended with %v", end)
	}
	if len(pages.asked) != 0 || len(a.BackgroundSyncs.list()) != 0 {
		t.Fatalf("read %d pages, syncs %+v", len(pages.asked), a.BackgroundSyncs.list())
	}
}

// Removing an archive stops its background sync and pauses its daily sync:
// the pages read are not merged, the removed archive is not revived, and the
// chat is told.
func TestRemovingAnArchiveStopsItsBackgroundSync(t *testing.T) {
	a, host, pages := syncHost(t)
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 8, "confirm": true})
	if _, _, err := a.Gacha.Import(gacha.Archive{UID: "100000001", Region: "cn_gf01", Timezone: 8, Language: "zh-cn", Records: []gacha.Record{{ID: "1", ItemID: "10000046", GachaType: "301", UIGFType: "301", Time: "2023-01-01 00:00:00", Rank: "5"}}}); err != nil {
		t.Fatal(err)
	}
	pages.before = func(n int) {
		if n == 3 {
			if _, err := a.manageGacha("gacha.remove", map[string]any{"uid": "100000001", "region": "cn_gf01"}); err != nil {
				t.Error(err)
			}
		}
	}
	host.message("#更新抽卡记录", "更新抽卡记录")
	if len(host.sent) != 2 || sentText(host.sent[1].Message) != "抽卡记录读取已取消。" || len(pages.asked) != 3 {
		t.Fatalf("answered %+v after %d pages", host.sent, len(pages.asked))
	}
	if items := a.BackgroundSyncs.list(); len(items) != 1 || items[0].State != "canceled" || items[0].LastCode != "archive_removed" {
		t.Fatalf("syncs %+v", items)
	}
	if _, err := a.Gacha.Read("100000001", "cn_gf01"); err == nil {
		t.Fatal("the removed archive was revived")
	}
	if tasks, _ := a.SyncTasks.List(); len(tasks) != 1 || tasks[0].State != "paused" || tasks[0].LastCode != "archive_removed" {
		t.Fatalf("daily syncs %+v", tasks)
	}
}

// A daily sync, run through the SDK as the host runs it: once its hour has
// come, its trigger moves to the background and reads the round with the
// task's delegation, a second a page, then tells the account's user and
// waits for the next day; later triggers that day read nothing.
func TestDailyGachaSyncReadsItsRoundInItsTriggersBackgroundEvent(t *testing.T) {
	a := pluginApp(t)
	start := time.Date(2026, 9, 20, 9, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	clock := &fakeClock{at: start}
	a.clock = clock
	pages := &gachaPages{t: t, clock: clock, characters: 45}
	host := newSDKHost(t, a, nil, pages.service(&fakeAccounts{t: t}))
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 8, "notify": true, "confirm": true})
	ref := host.job("game.sync.")
	pages.delegation = "grant:" + ref
	end, actions := host.trigger(ref)
	if _, ok := detached(actions); !ok || end["type"] != "result" || len(pages.asked) != 8 {
		t.Fatalf("the trigger ended with %v after %d pages", end, len(pages.asked))
	}
	pagedASecondApart(t, pages.asked)
	if archive, err := a.Gacha.Read("100000001", "cn_gf01"); err != nil || len(archive.Records) != 45 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
	next := time.Date(2026, 9, 21, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*3600)).UnixMilli()
	tasks, _ := a.SyncTasks.List()
	if len(tasks) != 1 || tasks[0].State != "waiting" || tasks[0].LastCode != "sync_completed" || tasks[0].NextCheckMS != next || tasks[0].Result == nil || tasks[0].Result.Added != 45 || tasks[0].LastNotificationMS == 0 {
		t.Fatalf("tasks %+v", tasks)
	}
	if len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" || !strings.HasPrefix(sentText(host.sent[0].Message), "抽卡后台同步完成") {
		t.Fatalf("told %+v", host.sent)
	}
	clock.set(start.Add(time.Hour))
	if _, actions := host.trigger(ref); len(pages.asked) != 8 || len(actions) != 0 {
		t.Fatalf("a later trigger asked for %+v", actions)
	}
}

// A daily round the host would not move to the background, while the plugin
// holds its limit of background events, is read by the next trigger.
func TestDailyGachaSyncRefusedTheBackgroundReadsOnTheNextTrigger(t *testing.T) {
	a := pluginApp(t)
	start := time.Date(2026, 9, 20, 9, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	clock := &fakeClock{at: start}
	a.clock = clock
	pages := &gachaPages{t: t, clock: clock, characters: 45}
	host := newSDKHost(t, a, nil, pages.service(&fakeAccounts{t: t}))
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 8, "confirm": true})
	ref := host.job("game.sync.")
	pages.delegation = "grant:" + ref
	host.busy = true
	if end, _ := host.trigger(ref); end["type"] != "result" || len(pages.asked) != 0 {
		t.Fatalf("the refused trigger ended with %v after %d pages", end, len(pages.asked))
	}
	if tasks, _ := a.SyncTasks.List(); len(tasks) != 1 || tasks[0].State != "waiting" || tasks[0].RunDay != "" {
		t.Fatalf("tasks %+v", tasks)
	}
	host.busy = false
	clock.set(start.Add(time.Minute))
	host.trigger(ref)
	if tasks, _ := a.SyncTasks.List(); len(pages.asked) != 8 || tasks[0].LastCode != "sync_completed" {
		t.Fatalf("read %d pages, tasks %+v", len(pages.asked), tasks)
	}
}

// A second 更新抽卡记录 sent as soon as the first answers finds the role free:
// the round is over before its report goes out.
func TestGachaSyncCommandFreesTheRoleBeforeItAnswers(t *testing.T) {
	_, host, pages := syncHost(t)
	again := false
	host.sending = func(request rayleabot.MessageSendRequest) {
		if !again && strings.HasPrefix(sentText(request.Message), "[角色]记录获取成功") {
			again = true
			if end, _ := host.message("#更新抽卡记录", "更新抽卡记录"); terminalText(end) != "" {
				t.Errorf("the second command answered %q", terminalText(end))
			}
		}
	}
	host.message("#更新抽卡记录", "更新抽卡记录")
	reports := 0
	for _, sent := range host.sent {
		if strings.Contains(sentText(sent.Message), "抽卡记录更新完成") {
			reports++
		}
	}
	// The second round reads one page of each pool: every record is known.
	if reports != 2 || len(pages.asked) != 14 {
		t.Fatalf("%d reports after %d pages", reports, len(pages.asked))
	}
}

// A daily trigger right after a round tells its user finds the task free:
// the task is released before the notification goes out.
func TestDailyGachaSyncFreesItsTaskBeforeItTellsItsUser(t *testing.T) {
	a := pluginApp(t)
	start := time.Date(2026, 9, 20, 9, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	clock := &fakeClock{at: start}
	a.clock = clock
	pages := &gachaPages{t: t, clock: clock, characters: 45}
	host := newSDKHost(t, a, nil, pages.service(&fakeAccounts{t: t}))
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "days": 30, "hour": 8, "notify": true, "confirm": true})
	ref := host.job("game.sync.")
	pages.delegation = "grant:" + ref
	told := 0
	host.sending = func(request rayleabot.MessageSendRequest) {
		if told++; told == 1 {
			// Run again at once, as 重新运行此同步 asks.
			if err := a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
				(*items)[i].State, (*items)[i].NextCheckMS = "running", 0
				return nil
			}); err != nil {
				t.Error(err)
			}
			host.trigger(ref)
		}
	}
	host.trigger(ref)
	// The second round reads one page of each pool: every record is known.
	if told != 2 || len(pages.asked) != 14 {
		t.Fatalf("told %d times after %d pages", told, len(pages.asked))
	}
}
