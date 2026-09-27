package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// selfHelpLogs serves the self-help queries for authkey abcdefghij: 25
// crystal records (two pages) and 21 primogem records of which two are 680s.
type selfHelpLogs struct{}

func (selfHelpLogs) RoundTrip(request *http.Request) (*http.Response, error) {
	query := request.URL.Query()
	body := map[string]any{"retcode": 0, "message": "OK"}
	switch {
	case query.Get("authkey") != "abcdefghij" || query.Get("auth_appid") != "csc":
		body = map[string]any{"retcode": -101, "message": "authkey timeout"}
	case strings.HasSuffix(request.URL.Path, "/GetUserInfo"):
		body["data"] = map[string]any{"uid": "100000001"}
	default:
		crystal := strings.HasSuffix(request.URL.Path, "/GetCrystalLog")
		// Record n has ID 900-n, newest first; end_id is the previous page's
		// last ID.
		count, first := 21, 1
		if crystal {
			count = 25
		}
		if end, _ := strconv.Atoi(query.Get("end_id")); end != 0 {
			first = 900 - end + 1
		}
		list := []any{}
		for n := first; n <= count && len(list) < 20; n++ {
			add := "60"
			if !crystal {
				add = map[bool]string{true: "680", false: "40"}[n == 3 || n == 20]
			}
			list = append(list, map[string]any{"id": strconv.Itoa(900 - n), "datetime": "2026-0" + strconv.Itoa(1+n%2) + "-01 12:00:00", "add_num": add})
		}
		body["data"] = map[string]any{"list": list}
	}
	raw, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(http.StatusOK)
	_, _ = recorder.Write(raw)
	return recorder.Result(), nil
}

func TestPayLogReadsBothLogsWithTheLinkKey(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: selfHelpLogs{}}
	job := &payLogJob{key: "abcdefghij", uid: "100000001", api: "GetCrystalLog"}
	if err := a.readPayLog(context.Background(), AccountsClient{}, job); err != nil || job.pages != 4 {
		t.Fatal(job.pages, err)
	}
	crystals, hymns := 0, 0
	for _, record := range job.records {
		if record.add == 680 {
			hymns++
		} else {
			crystals++
		}
	}
	if crystals != 25 || hymns != 2 {
		t.Fatal(crystals, hymns)
	}
	if _, err := a.payLogRequest(context.Background(), "expired", "GetUserInfo", ""); err == nil || friendlyError(err) != "您的链接过期，请重新获取" {
		t.Fatal(err)
	}
}

// accountLogs serves the same logs through the accounts plugin's billing
// pages for the role, and records each request's input.
type accountLogs struct {
	uid    string
	inputs []map[string]any
}

func (c *accountLogs) CallService(_ context.Context, r rayleabot.ServiceCallRequest, out any) error {
	if r.Method != "execute" || r.Params["operation"] != "genshin.billing" {
		return gameError("operation_denied", "unsupported")
	}
	input := r.Params["input"].(map[string]any)
	c.inputs = append(c.inputs, input)
	*out.(*QueryResult) = QueryResult{Role: Role{UID: c.uid}, Data: billingPage(input)}
	return nil
}

// billingPage is the accounts plugin's billing page for input: the logs
// selfHelpLogs serves.
func billingPage(input map[string]any) map[string]any {
	api := map[any]string{"crystal": "GetCrystalLog", "primogem": "GetPrimogemLog"}[input["category"]]
	request := httptest.NewRequest(http.MethodGet, payLogURL+api+"?authkey=abcdefghij&auth_appid=csc&end_id="+asText(input["end_id"]), nil)
	response, _ := selfHelpLogs{}.RoundTrip(request)
	var body struct {
		Data struct {
			List []any `json:"list"`
		} `json:"data"`
	}
	_ = json.NewDecoder(response.Body).Decode(&body)
	data := map[string]any{"items": body.Data.List, "has_more": len(body.Data.List) == 20}
	if len(body.Data.List) == 20 {
		data["next_end_id"] = body.Data.List[19].(map[string]any)["id"]
	}
	return data
}

func TestPayLogReadsTheAccountWithoutALink(t *testing.T) {
	a := pluginApp(t)
	caller := &accountLogs{uid: "100000001"}
	job := &payLogJob{uid: "100000001", choice: Selection{AccountRef: "account", RoleRef: "role"}, api: "GetCrystalLog"}
	if err := a.readPayLog(context.Background(), AccountsClient{Caller: caller}, job); err != nil || len(job.records) != 27 {
		t.Fatal(len(job.records), err)
	}
	// Only produced crystals and primogems are asked for, from cursor 0.
	if first := caller.inputs[0]; first["category"] != "crystal" || first["direction"] != "produce" || first["end_id"] != "0" || caller.inputs[2]["category"] != "primogem" {
		t.Fatal(caller.inputs)
	}
	// Pages of another role are refused.
	caller.uid = "100000002"
	if err := a.readPayLog(context.Background(), AccountsClient{Caller: caller}, &payLogJob{uid: "100000001", api: "GetCrystalLog"}); err == nil {
		t.Fatal("read another role's logs")
	}
}

func TestPayLogCountsPacksByMonthAsYunzai(t *testing.T) {
	// Oldest first by ID: two months, a doubled first top-up, a Welkin Moon
	// and a Gnostic Hymn, which does not add crystals; unknown amounts only
	// open their month.
	log := payLogData("100000001", []payRecord{
		{"1004", "2026-02-03 10:00:00", 60},
		{"1001", "2026-01-01 10:00:00", 12960},
		{"1002", "2026-01-02 10:00:00", 300},
		{"1003", "2026-01-05 10:00:00", 680},
		{"1005", "2026-03-01 10:00:00", 90},
	})
	want := PayLog{UID: "100000001", Crystal: 12960 + 300 + 60, Months: []PayMonth{
		{Month: "1月", Counts: [8]int{1, 1, 1, 0, 0, 0, 0, 0}},
		{Month: "2月", Counts: [8]int{0, 0, 0, 0, 0, 0, 0, 1}},
		{Month: "3月"},
	}}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("%+v", log)
	}
	if match := payLinkKey.FindStringSubmatch("https://webstatic.mihoyo.com/csc-service-center-fe/index.html?page_id=1&authkey=ab%2Bcd&game_biz=hk4e_cn#/player-log"); match == nil || match[1] != "ab%2Bcd" {
		t.Fatal(match)
	}
}

// customerServiceLink is a customer service link with authkey abcdefghij.
const customerServiceLink = "https://webstatic.mihoyo.com/csc-service-center-fe/index.html?page_id=1&authkey=abcdefghij&game_biz=hk4e_cn#/player-log"

// A customer service link sent in private, run through the SDK as the host
// runs it: the event moves to the background first, then answers as Yunzai
// does, the wait and then the result, keeping it and the link's authkey for
// the sender.
func TestPayLogByLinkReadsInItsBackgroundEvent(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: selfHelpLogs{}}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	end, actions := host.message(customerServiceLink, "")
	if len(actions) == 0 || actions[0].Name != "event.detach" || end["type"] != "result" {
		t.Fatalf("the link ended with %v after %+v", end, actions)
	}
	if len(host.sent) != 2 || sentText(host.sent[0].Message) != "正在获取消费数据,可能需要30s~~" || !strings.Contains(sentText(host.sent[1].Message), "充值统计") {
		t.Fatalf("answered %+v", host.sent)
	}
	owner := Subject{"onebot11", "a", "bot", "u"}
	if logs, err := a.PayLogs.Get(owner); err != nil || len(logs) != 1 || logs[0].UID != "100000001" || logs[0].Crystal != 25*60 {
		t.Fatalf("kept %+v, %v", logs, err)
	}
	if kept, ok := a.PayKeys.get(owner); !ok || kept.key != "abcdefghij" {
		t.Fatal("the link's authkey was not kept")
	}
}

// billingService answers the account plugin's service as accounts does, and
// its billing pages with the logs selfHelpLogs serves, each taking fifteen
// seconds on clock. fail, when set, answers a page with a failure code.
func billingService(t *testing.T, clock *fakeClock, reads *[]map[string]any, fail string) func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
	accounts := &fakeAccounts{t: t}
	return func(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
		if request.Method != "execute" || request.Params["operation"] != "genshin.billing" {
			return accounts.answer(request, scheduled)
		}
		// The event keeps its origin in the background: no delegation.
		if scheduled || request.Params["delegation_ref"] != nil {
			t.Errorf("read %+v", request.Params)
		}
		input := asObject(request.Params["input"])
		*reads = append(*reads, input)
		clock.set(clock.Now().Add(15 * time.Second))
		if fail != "" {
			return nil, fail
		}
		return map[string]any{"operation": "genshin.billing", "role": testRole, "data": billingPage(input)}, ""
	}
}

// 更新充值记录 without a link, run through the SDK as the host runs it: the
// event moves to the background and reads all four pages of the account's
// logs as the sender, longer than an event may take, then answers in the
// chat and keeps the result for the sender.
func TestPayLogByAccountReadsInItsBackgroundEvent(t *testing.T) {
	a := pluginApp(t)
	start := time.Unix(1_800_000_000, 0)
	clock := &fakeClock{at: start}
	a.clock = clock
	reads := []map[string]any{}
	host := newSDKHost(t, a, nil, billingService(t, clock, &reads, ""))
	end, actions := host.message("#更新充值记录", "更新充值记录")
	if _, ok := detached(actions); !ok || end["type"] != "result" || len(host.jobs) != 0 {
		t.Fatalf("the command ended with %v after %+v, jobs %v", end, actions, host.jobs)
	}
	if len(reads) != 4 || clock.Now().Sub(start) < time.Minute {
		t.Fatalf("read %+v by %v", reads, clock.Now().Sub(start))
	}
	if len(host.sent) != 2 || sentText(host.sent[0].Message) != "正在获取数据,可能需要30s" || host.sent[1].TargetID != "u" || !strings.Contains(sentText(host.sent[1].Message), "充值统计") {
		t.Fatalf("answered %+v", host.sent)
	}
	if logs, err := a.PayLogs.Get(Subject{"onebot11", "a", "bot", "u"}); err != nil || len(logs) != 1 || logs[0].UID != "100000001" {
		t.Fatalf("kept %+v, %v", logs, err)
	}
}

// A read that fails answers the failure in the chat and keeps nothing.
func TestPayLogFailedReadAnswersTheFailure(t *testing.T) {
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	reads := []map[string]any{}
	host := newSDKHost(t, a, nil, billingService(t, clock, &reads, "plugin.upstream_auth_invalid"))
	host.message("#更新充值记录", "更新充值记录")
	if len(host.sent) != 2 || sentText(host.sent[1].Message) != "plugin.upstream_auth_invalid" || len(reads) != 1 {
		t.Fatalf("answered %+v after %d reads", host.sent, len(reads))
	}
	if logs, _ := a.PayLogs.Get(Subject{"onebot11", "a", "bot", "u"}); len(logs) != 0 {
		t.Fatalf("kept %+v", logs)
	}
}
