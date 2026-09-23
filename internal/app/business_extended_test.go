package app

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func decoded(t *testing.T, source string) map[string]any {
	t.Helper()
	var result map[string]any
	decoder := json.NewDecoder(strings.NewReader(source))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMonthlyAndChallengeViewsUseBusinessFields(t *testing.T) {
	for _, tc := range []struct {
		game, op, data string
		want           []string
	}{
		{"genshin", "monthly", `{"data_month":9,"month_data":{"current_primogems":1200,"current_mora":30000,"group_by":[{"action":"活动奖励","num":800,"percent":60}]} }`, []string{"本月原石：1200", "本月摩拉：30000", "活动奖励：800"}},
	} {
		t.Run(tc.game+tc.op, func(t *testing.T) {
			view := BusinessView(Game{ID: tc.game}, Operation{Name: tc.game + "." + tc.op, Label: "测试"}, QueryResult{Data: decoded(t, tc.data)}, Catalog{})
			for _, text := range tc.want {
				if !strings.Contains(view.Text(), text) {
					t.Fatalf("missing business output %q", text)
				}
			}
			if strings.Contains(view.Text(), "avatar_list") || strings.Contains(view.Text(), "grid_fight_brief") {
				t.Fatal("raw protocol key shown to user")
			}
		})
	}
}
func TestExtendedCommandInputs(t *testing.T) {
	app := &App{}
	// A month outside the last three is refused, as upstream.
	now := int(time.Now().In(time.FixedZone("UTC+8", 28800)).Month())
	for _, tc := range []struct{ mode, valid, invalid, key string }{{"month", strconv.Itoa(now), strconv.Itoa((now+5)%12 + 1), "month"}, {"period", "上期", "3", "schedule_type"}} {
		input, uid, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.valid, "100000001"}, nil)
		if err != nil || uid != "100000001" || input[tc.key] == nil {
			t.Fatal("valid selector failed")
		}
		if _, _, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.invalid}, nil); err == nil {
			t.Fatal("invalid selector accepted")
		}
		// A UID alone, as in 原石100000001 or 深渊12层100000001, is the UID.
		if input, uid, err := app.commandInput(Operation{Input: tc.mode}, []string{"100000001"}, nil); err != nil || uid != "100000001" || input[tc.key] != nil {
			t.Errorf("%s with a UID alone = %v %q %v", tc.mode, input, uid, err)
		}
	}
	// 往期 is the last period, and 本期 as in 深渊本期 the current one.
	for word, period := range map[string]int{"往期": 2, "本期": 1} {
		if input, _, err := app.commandInput(Operation{Input: "period"}, []string{word}, nil); err != nil || input["schedule_type"] != period {
			t.Errorf("%s = %v, %v", word, input["schedule_type"], err)
		}
	}
}
