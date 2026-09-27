package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

func TestGachaLinkParsesYunzaiLinksOnly(t *testing.T) {
	link, ok, err := parseGachaLink("https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey_ver=1&region=cn_gf01&authkey=abcdefgh%2Bij#/log〈=zh-cn")
	if !ok || err != nil || link.key != "abcdefgh+ij" || link.region != "cn_gf01" {
		t.Fatal(link, ok, err)
	}
	if _, ok, _ := parseGachaLink("https://webstatic.mihoyo.com/hkrpg/event/e20211215gacha-v2/index.html?authkey=abcdefghij&game_biz=hkrpg_cn"); ok {
		t.Fatal("a Star Rail link was claimed")
	}
	if _, ok, _ := parseGachaLink("绝区零 index.html?authkey=abcdefghij&game_biz=nap_cn"); ok {
		t.Fatal("a Zenless link was claimed")
	}
	if _, ok, _ := parseGachaLink("https://webstatic.mihoyo.com/csc-service-center-fe/index.html?authkey=abcdefghij#/player-log"); ok {
		t.Fatal("a customer service link was claimed")
	}
	if _, ok, err := parseGachaLink("authkey=short"); !ok || err == nil {
		t.Fatal("a broken authkey was accepted")
	}
	if _, ok, _ := parseGachaLink("抽卡记录"); ok {
		t.Fatal("a message without a link was claimed")
	}
}

// wishHistory serves the official wish history: 21 character event and 1
// weapon event records for UID 100000001, found on the mainland server.
type wishHistory struct{ retcode int }

func (h wishHistory) RoundTrip(request *http.Request) (*http.Response, error) {
	query := request.URL.Query()
	body := map[string]any{"retcode": h.retcode, "message": "", "data": nil}
	if h.retcode == 0 {
		if query.Get("region") != "cn_gf01" || query.Get("authkey") != "abcdefghij" {
			body["retcode"] = -101
		} else {
			// Record n of a pool has ID base-n, newest first; end_id is the
			// last ID of the previous page.
			records := []any{}
			pool := query.Get("gacha_type")
			count, base := map[string]int{"301": 21, "302": 1}[pool], map[string]int{"301": 5000, "302": 4000}[pool]
			first := 1
			if end, _ := strconv.Atoi(query.Get("end_id")); end != 0 {
				first = base - end + 1
			}
			size, _ := strconv.Atoi(query.Get("size"))
			for n := first; n <= count && len(records) < size; n++ {
				records = append(records, map[string]any{"uid": "100000001", "gacha_type": pool, "item_id": "10000046", "count": "1", "time": fmt.Sprintf("2024-01-01 00:00:%02d", 59-n), "name": "胡桃", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(base - n)})
			}
			body["data"] = map[string]any{"list": records, "region": "cn_gf01"}
		}
	}
	raw, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(http.StatusOK)
	_, _ = recorder.Write(raw)
	return recorder.Result(), nil
}

func TestGachaLinkFetchesTheWholeHistoryWithoutAnAccount(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: wishHistory{}}
	link, _, _ := parseGachaLink("https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey=" + url.QueryEscape("abcdefghij") + "#/log")
	uid, err := a.checkGachaLink(context.Background(), &link, 80)
	if err != nil || uid != "100000001" || link.region != "cn_gf01" {
		t.Fatal(uid, link.region, err)
	}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{Link: true}, uid, link.region, false)
	if err != nil {
		t.Fatal(err)
	}
	job := &gachaLinkJob{link: link, uid: uid, sync: info.Ref, before: map[string]int{}}
	result, err := a.stepGachaLink(context.Background(), job, time.Now().Add(time.Minute))
	if err != nil || result == nil || result.Added != 22 {
		t.Fatal(result, err)
	}
	archive, err := a.Gacha.Read(uid, link.region)
	if err != nil {
		t.Fatal(err)
	}
	summary := gachaLinkSummary("#", job.before, gachaPoolCounts(archive))
	if !strings.HasPrefix(summary, "[角色]记录获取成功，更新21条\n[武器]记录获取成功，更新1条\n\n抽卡记录更新完成") {
		t.Fatal(summary)
	}
	// Management steps cannot drive a link's sync.
	if choice, err := a.Syncs.Choice(info.Ref); err == nil && !choice.Link {
		t.Fatal("link sync lost its kind")
	}
}

func TestGachaLinkRepliesAsYunzaiToOfficialErrors(t *testing.T) {
	a := pluginApp(t)
	for retcode, want := range map[int]string{-101: "该链接已失效，请重新进入游戏，重新复制链接", -109: "2.3版本后，反馈的链接已无法查询！请用安卓方式获取链接", -100: "链接不完整"} {
		a.LinkHTTP = &http.Client{Transport: wishHistory{retcode: retcode}}
		link := gachaLink{key: "abcdefghij", region: "cn_gf01"}
		if _, err := a.checkGachaLink(context.Background(), &link, 80); err == nil || !strings.HasPrefix(friendlyError(err), want) {
			t.Fatal(retcode, err)
		}
	}
	a.LinkHTTP = &http.Client{Transport: wishHistory{retcode: -100}}
	link := gachaLink{key: "abcdefghij", region: "cn_gf01"}
	if _, err := a.checkGachaLink(context.Background(), &link, 1000); err == nil || !strings.HasPrefix(friendlyError(err), "输入法限制") {
		t.Fatal(err)
	}
}

// roundTripFunc answers a client's requests with a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// wishLink is a wish history link of the mainland server.
const wishLink = "https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey_ver=1&region=cn_gf01&authkey=abcdefghij#/log"

// A link whose history takes longer than its event, run through the SDK as
// the host runs it: the triggers of its job fetch the rest and answer in the
// private chat the link came from, each pool's count and then the record.
func TestGachaLinkFinishesOnTheHostsTriggers(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	// Each request takes ten seconds, so the event reads three pages after
	// finding the UID.
	a.LinkHTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		clock.set(clock.Now().Add(10 * time.Second))
		return wishHistory{}.RoundTrip(r)
	})}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	end, _ := host.message(wishLink, "")
	if terminalText(end) != "记录较多，将在后台继续获取，完成后在此回复。" || len(host.sent) != 1 {
		t.Fatalf("the link event ended with %v after %d messages", end, len(host.sent))
	}
	ref := host.job(gachaLinkTask)
	triggerUntilDone(t, clock, host, ref, time.Unix(1_800_000_000, 0), 5)
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.sent) != 3 {
		t.Fatalf("deleted %v, answered %v", host.deleted, host.sent)
	}
	if sent := host.sent[1]; sent.TargetType != "private" || sent.TargetID != "u" || !strings.HasPrefix(sentText(sent.Message), "[角色]记录获取成功，更新21条\n[武器]记录获取成功，更新1条") {
		t.Fatalf("answered %+v", sent)
	}
	if archive, err := a.Gacha.Read("100000001", "cn_gf01"); err != nil || len(archive.Records) != 22 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}

// A link read within its event in a group answers every message of the
// reply, then asks the sender to recall the link.
func TestGachaLinkInAGroupAnswersEachMessage(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: wishHistory{}}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	end, _ := host.groupMessage(wishLink, "")
	if terminalText(end) != "已收到链接，请撤回" || len(host.jobs) != 0 {
		t.Fatalf("the link event ended with %v, jobs %v", end, host.jobs)
	}
	if len(host.sent) != 3 || sentText(host.sent[0].Message) != "链接发送成功，数据获取中……" || !strings.HasPrefix(sentText(host.sent[1].Message), "[角色]记录获取成功") || host.sent[2].TargetType != "group" {
		t.Fatalf("answered %+v", host.sent)
	}
}
