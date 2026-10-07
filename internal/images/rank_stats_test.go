package images_test

import (
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestRankStatsDrawsThePanelBlockWithoutAPanel(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog}
	character, _ := application.Catalog.Get("10000046")
	damage := &app.RankCurve{Scores: []float64{99, 90, 80, 70, 60, 50, 40, 30, 20, 10}, Top: 100, Total: "60391", Percent: -100, Score: -100}
	image, ok := images.RankStats(context, app.RankStatsImage{Character: character, Damage: damage, DamageTitle: "重击伤害"})
	if !ok || image.Template != "rank-stats" || image.Data["elem"] != "pyro" || image.Data["name"] != "胡桃" || image.Data["title"] != "重击伤害" {
		t.Fatalf("image = %v", image.Data)
	}
	charts := image.Data["charts"].([]any)
	drawn, missing := charts[0].(map[string]any), charts[1].(map[string]any)
	// Without a panel nothing is written on the curve and the area keeps one
	// colour; a distribution ark did not send draws 暂无数据.
	if drawn["line"] == "" || drawn["mark"] != nil || len(drawn["stops"].([]any)) != 2 || len(missing) != 0 {
		t.Fatalf("charts = %v", charts)
	}
	if totals := image.Data["totals"].([]any); totals[0] != "统计样本 60391 个UID" || totals[1] != "暂无数据" {
		t.Fatalf("totals = %v", totals)
	}
}
