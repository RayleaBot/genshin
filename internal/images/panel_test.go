package images_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/artwork"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestPanelFollowsMiaoRules(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Key == "gs_10000046" {
			image.Record = record
		}
	}
	image.Panel = app.CharacterPanel{ID: "10000046", Level: 90, Rank: 2, Element: "pyro",
		Weapon: &app.PanelEquipment{ID: "13501", Name: "护摩之杖", Level: 90, Refinement: 2, Main: []app.PanelStat{{Value: "608"}}, Sub: []app.PanelStat{{Key: "cdmg", Value: "66.2%"}}},
		Stats:  []app.PanelStat{{ID: "2000", Value: "35012", Base: "15552", Added: "19460"}},
		Skills: []app.PanelSkill{{ID: "10461", SkillType: 1, Level: 10}, {ID: "10462", SkillType: 1, Level: 12, ExtraLevel: 3}, {ID: "10463", SkillType: 1, Level: 9}},
	}
	game := application.Game
	game.Prefix = "#"
	drawn, ok := images.Panel(app.ImageContext{Game: game, Now: time.Now()}, image)
	if !ok || drawn.Template != "panel" {
		t.Fatalf("drawn = %+v", drawn)
	}
	talents := drawn.Data["talents"].([]any)
	// The skill with a constellation bonus is marked; only original levels of
	// ten or more earn the crown.
	for index, want := range []map[string]any{{"level": 10, "plus": false, "crown": true}, {"level": 12, "plus": true, "crown": false}, {"level": 9, "plus": false, "crown": false}} {
		got := talents[index].(map[string]any)
		for key, value := range want {
			if got[key] != value {
				t.Errorf("talent %d %s = %v, want %v", index, key, got[key], value)
			}
		}
	}
	if cons := drawn.Data["cons_icons"].([]any); cons[1].(map[string]any)["off"] != false || cons[2].(map[string]any)["off"] != true {
		t.Errorf("constellations = %v", cons)
	}
	if hp := drawn.Data["attrs"].([]any)[0].(map[string]any); hp["value"] != "35,012" || hp["plus"] != "19,460" {
		t.Errorf("hp row = %v", hp)
	}
	weapon := drawn.Data["weapon"].(map[string]any)
	desc := weapon["desc"].([]any)
	if first := desc[0].(map[string]any); first["text"] != "生命值提升" || first["value"] != "25%" {
		t.Errorf("refinement text = %v", desc)
	}
	if drawn.Data["artifact_hint"] != "#胡桃圣遗物" || len(drawn.Resources) != 0 {
		t.Errorf("hint %v, resources without artwork %v", drawn.Data["artifact_hint"], drawn.Resources)
	}
	if _, ok := images.Panel(app.ImageContext{Game: game}, app.PanelImage{}); ok {
		t.Error("a panel without a calculation record should keep the summary card")
	}
}

func TestPanelTakesTheTravelerPicturesFromMiaoFolders(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Name == "旅行者" && record.Element == "anemo" {
			image.Record = record
		}
	}
	image.Panel = app.CharacterPanel{ID: "10000005", Level: 90, Rank: 6, Element: "anemo"}
	root := t.TempDir()
	splash := "resources/meta-gs/character/空/imgs/splash.webp"
	cons := "resources/meta-gs/character/旅行者/anemo/icons/cons-1.webp"
	for _, name := range []string{splash, cons} {
		file := filepath.Join(root, "miao-plugin", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("webp"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	drawn, ok := images.Panel(app.ImageContext{Game: application.Game, Now: time.Now(), Artwork: &artwork.Store{Root: root}}, image)
	if !ok {
		t.Fatal("the Traveler panel was not drawn")
	}
	paths := map[string]string{}
	for _, resource := range drawn.Resources {
		paths[resource.ID] = resource.Path
	}
	// miao draws 空 or 荧 by the character, and the constellation icons of
	// the element the Traveler resonates with.
	if paths["splash"] != "assets/miao-plugin/"+splash || paths["cons-1"] != "assets/miao-plugin/"+cons {
		t.Errorf("resources = %v", paths)
	}
}

func TestPanelDrawsArkRanks(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Key == "gs_10000046" {
			image.Record = record
		}
	}
	expected := 45892.0
	image.Panel = app.CharacterPanel{ID: "10000046", Level: 90, Element: "pyro"}
	image.Damage = &app.BuildResult{Baseline: app.BuildScenario{Results: []app.BuildSkillResult{{Title: "重击蒸发", Expected: &expected}}}}
	game := application.Game
	game.Prefix = "#"
	context := app.ImageContext{Game: game, Catalog: application.Catalog, Now: time.Now()}
	// ark's ranks follow the damage rows, numbered on.
	image.Rank = &app.PanelRank{Rows: []app.Row{{Label: "总伤害排名", Value: "1148 / 60392 (1.90%)"}}}
	drawn, _ := images.Panel(context, image)
	rows := drawn.Data["damage"].(map[string]any)["rows"].([]any)
	if last := rows[len(rows)-1].(map[string]any); len(rows) != 2 || last["index"] != 2 || last["title"] != "总伤害排名" || last["full"] != "1148 / 60392 (1.90%)" {
		t.Fatalf("rows = %v", rows)
	}
	scores := []float64{99, 90, 80, 70, 60, 50, 40, 30, 20, 10}
	image.Rank = &app.PanelRank{Chart: &app.PanelRankChart{Damage: &app.RankCurve{Scores: scores, Top: 100, Percent: 1.9, Score: 74.172}, Places: [2]string{"1148 / 60392 (1.90%)", ""}}}
	drawn, _ = images.Panel(context, image)
	chart := drawn.Data["rank_chart"].(map[string]any)
	damage, artis := chart["charts"].([]any)[0].(map[string]any), chart["charts"].([]any)[1].(map[string]any)
	if chart["hint"] != "#胡桃排名统计" || len(artis) != 0 || damage["line"] == "" {
		t.Fatalf("chart = %v", chart)
	}
	// x reads 100 - x, and the panel is marked at 100 - its percent with its
	// score to two places.
	ticks := damage["x_ticks"].([]any)
	if ticks[0].(map[string]any)["label"] != "100" || ticks[5].(map[string]any)["label"] != "0" || damage["mark"].(map[string]any)["text"] != "74.17" {
		t.Fatalf("damage = %v", damage)
	}
	if stops := damage["stops"].([]any); stops[2].(map[string]any)["color"] != "rgb(255, 0, 0)" || math.Abs(stops[1].(map[string]any)["offset"].(float64)-0.981) > 1e-9 {
		t.Fatalf("stops = %v", stops)
	}
}

func TestPanelComparesChangedDamageAsArk(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Key == "gs_10000046" {
			image.Record = record
		}
	}
	f := func(v float64) *float64 { return &v }
	long := "一二三四五六七八九十(1)二三四五六七八九十二三"
	image.Panel = app.CharacterPanel{ID: "10000046", Level: 90, Element: "pyro"}
	image.Damage = &app.BuildResult{Baseline: app.BuildScenario{Results: []app.BuildSkillResult{
		{Title: "重击", Critical: f(110), Expected: f(55)}, {Title: "治疗", Text: "12,345"}, {Title: long, Expected: f(1)},
	}}}
	image.Original = &app.BuildResult{Baseline: app.BuildScenario{Results: []app.BuildSkillResult{{Title: "重击", Critical: f(100), Expected: f(55)}, {Title: "治疗", Text: "12,000"}}}}
	image.LongTitles = 1
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Now()}
	drawn, _ := images.Panel(context, image)
	rows := drawn.Data["damage"].(map[string]any)["rows"].([]any)
	first, second, third := rows[0].(map[string]any), rows[1].(map[string]any), rows[2].(map[string]any)
	diff := func(row map[string]any, key string) string { return row[key].(map[string]any)["text"].(string) }
	// An unchanged value reads 0.0% (upstream: " ↓0%"); a text has no critical
	// damage to compare.
	if diff(first, "dmg_diff") != " ↑10%" || diff(first, "avg_diff") != "0.0%" || diff(second, "avg_diff") != " ↑2.9%" || diff(second, "dmg_diff") != "--" || third["avg_diff"] != nil {
		t.Fatalf("rows = %v", rows)
	}
	// 1, ( and ) count half: the title is cut before its 21st width.
	if third["title"] != "一二三四五六七八九十(1)二三四五六七八九..." {
		t.Fatalf("title = %v", third["title"])
	}
	image.LongTitles = 2
	drawn, _ = images.Panel(context, image)
	if damage := drawn.Data["damage"].(map[string]any); damage["wrap"] != true || damage["rows"].([]any)[2].(map[string]any)["title"] != long {
		t.Fatalf("damage = %v", damage)
	}
}
