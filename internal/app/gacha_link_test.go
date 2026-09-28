package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
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
// weapon event records for UID 100000001, found on the mainland server, or
// characters character event records when set.
type wishHistory struct{ retcode, characters int }

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
			if pool == "301" && h.characters > 0 {
				count = h.characters
			}
			first := 1
			if end, _ := strconv.Atoi(query.Get("end_id")); end != 0 {
				first = base - end + 1
			}
			size, _ := strconv.Atoi(query.Get("size"))
			for n := first; n <= count && len(records) < size; n++ {
				at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(1000-n) * time.Second).Format(time.DateTime)
				records = append(records, map[string]any{"uid": "100000001", "gacha_type": pool, "item_id": "10000046", "count": "1", "time": at, "name": "胡桃", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(base - n)})
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

func TestGachaLinkFindsTheUIDWithoutAnAccount(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: wishHistory{}}
	link, _, _ := parseGachaLink("https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey=" + url.QueryEscape("abcdefghij") + "#/log")
	uid, characters, err := a.checkGachaLink(context.Background(), &link, 80)
	if err != nil || uid != "100000001" || !characters || link.region != "cn_gf01" {
		t.Fatal(uid, characters, link.region, err)
	}
}

func TestGachaLinkRepliesAsYunzaiToOfficialErrors(t *testing.T) {
	a := pluginApp(t)
	for retcode, want := range map[int]string{-101: "该链接已失效，请重新进入游戏，重新复制链接", -109: "2.3版本后，反馈的链接已无法查询！请用安卓方式获取链接", -100: "链接不完整"} {
		a.LinkHTTP = &http.Client{Transport: wishHistory{retcode: retcode}}
		link := gachaLink{key: "abcdefghij", region: "cn_gf01"}
		if _, _, err := a.checkGachaLink(context.Background(), &link, 80); err == nil || !strings.HasPrefix(friendlyError(err), want) {
			t.Fatal(retcode, err)
		}
	}
	a.LinkHTTP = &http.Client{Transport: wishHistory{retcode: -100}}
	link := gachaLink{key: "abcdefghij", region: "cn_gf01"}
	if _, _, err := a.checkGachaLink(context.Background(), &link, 1000); err == nil || !strings.HasPrefix(friendlyError(err), "输入法限制") {
		t.Fatal(err)
	}
}

// roundTripFunc answers a client's requests with a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// wishLink is a wish history link of the mainland server.
const wishLink = "https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey_ver=1&region=cn_gf01&authkey=abcdefghij#/log"

// A link whose history takes minutes to read, run through the SDK as the
// host runs it: the event moves to the background at once and reads every
// page in turn, then answers in the private chat the link came from as
// Yunzai does, the first read announced, each pool's count and then the
// record, without any scheduled job.
func TestGachaLinkReadsEveryPageInItsBackgroundEvent(t *testing.T) {
	a := pluginApp(t)
	start := time.Unix(1_800_000_000, 0)
	clock := &fakeClock{at: start}
	a.clock = clock
	requests := 0
	// 205 character event records take eleven pages, 17 requests with the
	// link's check and the other pools; each takes ten seconds.
	a.LinkHTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		clock.set(clock.Now().Add(10 * time.Second))
		return wishHistory{characters: 205}.RoundTrip(r)
	})}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	end, actions := host.message(wishLink, "")
	if action, ok := detached(actions); !ok || actions[0].Name != "event.detach" || len(action.Data) != 0 {
		t.Fatalf("the link did not move to the background first: %+v", actions)
	}
	if end["type"] != "result" || len(host.jobs) != 0 || requests != 17 || clock.Now().Sub(start) < 2*time.Minute {
		t.Fatalf("the link ended with %v after %v, jobs %v", end, clock.Now().Sub(start), host.jobs)
	}
	texts := []string{}
	for _, sent := range host.sent {
		if sent.TargetType != "private" || sent.TargetID != "u" {
			t.Fatalf("answered in %+v", sent)
		}
		texts = append(texts, sentText(sent.Message))
	}
	if len(texts) != 4 || texts[0] != "链接发送成功，数据获取中……" || texts[1] != "开始获取角色记录，首次获取数据较多，请耐心等待..." || !strings.HasPrefix(texts[2], "[角色]记录获取成功，更新205条\n[武器]记录获取成功，更新1条\n\n抽卡记录更新完成") || !strings.Contains(texts[3], "胡桃") {
		t.Fatalf("answered %q", texts)
	}
	if archive, err := a.Gacha.Read("100000001", "cn_gf01"); err != nil || len(archive.Records) != 206 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}

// A link in a group answers every message of the reply, then asks the
// sender to recall the link, mentioning them. A second link reads without
// announcing a first read.
func TestGachaLinkInAGroupAnswersEachMessage(t *testing.T) {
	a := pluginApp(t)
	a.clock = &fakeClock{at: time.Now()}
	a.LinkHTTP = &http.Client{Transport: wishHistory{}}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	end, _ := host.groupMessage(wishLink, "")
	if end["type"] != "result" || len(host.jobs) != 0 {
		t.Fatalf("the link event ended with %v, jobs %v", end, host.jobs)
	}
	recall := host.sent[len(host.sent)-1]
	if len(host.sent) != 5 || sentText(host.sent[0].Message) != "链接发送成功，数据获取中……" || !strings.HasPrefix(sentText(host.sent[2].Message), "[角色]记录获取成功") || recall.TargetType != "group" || recall.TargetID != "g" {
		t.Fatalf("answered %+v", host.sent)
	}
	if mention := recall.Message.Segments[0]; mention.Type != "at" || asText(mention.Data["user_id"]) != "u" || sentText(recall.Message) != "\n已收到链接，请撤回" {
		t.Fatalf("asked to recall with %+v", recall.Message)
	}
	host.sent = nil
	host.groupMessage(wishLink, "")
	if len(host.sent) != 4 || !strings.HasPrefix(sentText(host.sent[1].Message), "[角色]记录获取成功，更新0条") {
		t.Fatalf("answered the second link with %+v", host.sent)
	}
}

// A link the host will not move to the background, while the plugin holds its
// limit of background events, is answered at once and not read.
func TestGachaLinkRefusedTheBackgroundRepliesClearly(t *testing.T) {
	a := pluginApp(t)
	requests := 0
	a.LinkHTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return wishHistory{}.RoundTrip(r)
	})}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	host.busy = true
	end, _ := host.message(wishLink, "")
	if terminalText(end) != "正在后台处理的任务较多，请稍后再试。" || requests != 0 || len(host.sent) != 0 {
		t.Fatalf("the refused link ended with %v after %d requests, %+v", end, requests, host.sent)
	}
}
