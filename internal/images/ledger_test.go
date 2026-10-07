package images_test

import (
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestLedgerFollowsYunzai(t *testing.T) {
	data := decode(t, `{"data_month":9,"month_data":{"current_primogems":12345,"current_mora":2345678,"last_primogems":8000,"last_mora":900,
		"group_by":[{"action_id":1,"action":"冒险奖励","num":9000,"percent":75},{"action_id":3,"action":"每日奖励","num":2800,"percent":23},{"action_id":6,"action":"活动奖励","num":200,"percent":2}]}}`)
	context := app.ImageContext{Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	image, ok := images.Ledger(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok {
		t.Fatal("ledger")
	}
	// Upstream writes ten thousands with a w, counts pulls at 160 each and
	// dates the current month to the day.
	for key, want := range map[string]any{"day": "9月21号", "current_primogems": "1.23 w", "gacha": "77", "current_mora": "234.6 w", "last_primogems": "8000", "last_gacha": "50", "total": "12000"} {
		if image.Data[key] != want {
			t.Errorf("%s = %v, want %v", key, image.Data[key], want)
		}
	}
	slices := image.Data["slices"].([]any)
	first, last := slices[0].(map[string]any), slices[2].(map[string]any)
	// The ring starts at the top; slices of 2% or less carry no label.
	if first["color"] != "#d56565" || !strings.HasPrefix(first["path"].(string), "M 120.00 10.00") || first["label"] != "75%" || last["label"] != nil {
		t.Errorf("slices = %v", slices)
	}
}
