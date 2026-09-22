package images_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestLedgerCountFollowsYunzai(t *testing.T) {
	month := func(key string, primogems, mora int, groups ...any) app.SavedMonth {
		return app.SavedMonth{Month: key, Data: map[string]any{"month_data": map[string]any{
			"current_primogems": json.Number(itoa(primogems)), "current_mora": json.Number(itoa(mora)), "group_by": groups}}}
	}
	group := func(action string, id, num int) any {
		return map[string]any{"action": action, "action_id": json.Number(itoa(id)), "num": json.Number(itoa(num))}
	}
	stats := app.MonthlyStats{Role: app.Role{UID: "100000001"}, Months: []app.SavedMonth{
		month("2025-08", 9999, 1, group("活动奖励", 6, 100)),
		month("2025-10", 5000, 2000000, group("每日奖励", 3, 300), group("活动奖励", 6, 700)),
		month("2026-08", 1250, 3000000, group("活动奖励", 6, 200), group("深境螺旋", 4, 1000)),
		month("2026-09", 7000, 1000000),
	}}
	context := app.ImageContext{Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	image, ok := images.LedgerCount(context, stats)
	// The last twelve months exclude August 2025.
	if !ok || image.Data["total"] != "1.33w" || image.Data["gacha"] != "83" || image.Data["mora"] != "600w" ||
		image.Data["best_month"] != "9" || image.Data["best_mora_month"] != "8" || image.Data["year_text"] != "" || image.Data["more"] != false {
		t.Fatalf("recent = %v", image.Data)
	}
	// Sources are summed and ordered by amount.
	top := image.Data["top"].([]any)
	if top[0].(map[string]any)["action"] != "深境螺旋" || top[1].(map[string]any)["num"] != 900 || image.Data["sum"] != "2200" {
		t.Errorf("top = %v, sum = %v", top, image.Data["sum"])
	}
	// The columns run from the oldest month.
	bars := image.Data["columns"].(map[string]any)["bars"].([]any)
	if len(bars) != 3 || bars[0].(map[string]any)["label"] != "10月" || bars[2].(map[string]any)["label"] != "9月" {
		t.Errorf("bars = %v", bars)
	}
	stats.Word = "去年原石统计"
	image, _ = images.LedgerCount(context, stats)
	if image.Data["year_text"] != "2025年-" || image.Data["best_month"] != "8" || len(image.Data["columns"].(map[string]any)["bars"].([]any)) != 2 {
		t.Errorf("last year = %v", image.Data)
	}
	stats.Word = "2024年原石统计"
	if _, ok := images.LedgerCount(context, stats); ok {
		t.Error("a year without months drew a page")
	}
}

func itoa(value int) string { raw, _ := json.Marshal(value); return string(raw) }
