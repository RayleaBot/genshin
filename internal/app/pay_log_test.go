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
	done, err := a.stepPayLog(context.Background(), job, time.Now().Add(time.Minute))
	if err != nil || !done || job.pages != 4 {
		t.Fatal(done, job.pages, err)
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
