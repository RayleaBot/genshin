package images

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
)

func TestNoteFollowsUpstreamWording(t *testing.T) {
	now := time.Date(2026, 9, 21, 22, 30, 0, 0, time.FixedZone("UTC+8", 8*3600))
	data := map[string]any{
		"current_resin": json.Number("150"), "max_resin": json.Number("200"), "resin_recovery_time": "24000",
		"current_home_coin": json.Number("2300"), "max_home_coin": json.Number("2400"), "home_coin_recovery_time": "90000",
		"finished_task_num": json.Number("4"), "total_task_num": json.Number("4"), "is_extra_task_reward_received": true,
		"current_expedition_num": json.Number("2"), "max_expedition_num": json.Number("5"),
		"expeditions":               []any{map[string]any{"remained_time": "0"}, map[string]any{"remained_time": "3600"}},
		"remain_resin_discount_num": json.Number("1"), "resin_discount_num_limit": json.Number("3"),
		"transformer": map[string]any{"obtained": true, "recovery_time": map[string]any{"Day": json.Number("2"), "Hour": json.Number("3"), "Minute": json.Number("0"), "reached": false}},
	}
	image, ok := Note(app.ImageContext{Now: now}, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok || image.Template != "note" || image.Data["day"] != "09-21 22:30 星期一" {
		t.Fatalf("image = %+v", image)
	}
	want := [][3]string{
		{"原粹树脂", "将于明天 05:10 全部恢复", "150/200"},
		{"洞天宝钱", "预计1天1小时0分钟后达到上限", "2300/2400"},
		{"每日委托任务", "今日委托奖励已领取", "4/4"},
		// The earliest expedition has finished, so upstream reports all done.
		{"探索派遣", "派遣已完成", "2/5"},
		{"值得铭记的强敌", "周本树脂减半次数已用", "2/3"},
		{"参量质变仪", "2天3小时后可使用", "冷却中"},
	}
	rows := image.Data["rows"].([]any)
	for index, row := range rows {
		item := row.(map[string]any)
		if got := [3]string{item["name"].(string), item["detail"].(string), item["value"].(string)}; got != want[index] {
			t.Errorf("row %d = %v, want %v", index, got, want[index])
		}
	}
	if rows[1].(map[string]any)["alert"] != true {
		t.Error("home coin above 90% should be highlighted")
	}
	if len(image.Resources) != 0 {
		t.Errorf("resources without downloaded artwork: %v", image.Resources)
	}
	if _, ok := Note(app.ImageContext{Now: now}, app.QueryResult{Data: map[string]any{}}); ok {
		t.Error("a result without resin should keep the summary card")
	}
}
