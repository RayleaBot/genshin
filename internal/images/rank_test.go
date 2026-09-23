package images_test

import (
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestCloudRankFollowsArk(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	game := application.Game
	game.Prefix = "#"
	character, _ := application.Catalog.Get("10000046")
	damage := 64248.529
	entries := []app.CustomRankEntry{
		{UID: "10****543", Level: 90, Cons: 0, Talents: map[string]app.PanelTalent{"a": {Level: 10, Original: 10}}, Weapon: "护摩之杖", WeaponLevel: 90, WeaponAffix: 4,
			Sets: []string{"炽烈的炎之魔女"}, SetName: "炽烈的炎之魔女4", Mark: 281.6, Damage: &damage},
		{UID: "10****107", Mark: 221.7},
		{UID: "10****494", Mark: 34.9},
	}
	drawn, ok := images.CloudRank(app.ImageContext{Game: game, Catalog: application.Catalog}, app.CloudRankImage{Character: character, Mode: "mark", DamageTitle: "重击蒸发(半血开E)",
		Limit: "排序: 圣遗物评分 降序 / 筛选: 无", Number: 20, Entries: entries})
	if !ok || drawn.Template != "rank" || drawn.Data["title"] != "#胡桃圣遗物评分排行" || drawn.Data["time"] != "全服数据" || drawn.Data["width"] != 850 || drawn.Data["number"] != 20 {
		t.Fatalf("drawn = %+v", drawn.Data)
	}
	rows := drawn.Data["rows"].([]any)
	first := rows[0].(map[string]any)
	weapon := first["weapon"].(map[string]any)
	if first["uid"] != "10****543" || first["mark"] != "281.6" || weapon["name"] != "护摩之杖" || weapon["affix"] != 4 || weapon["star"] != 5 || first["set_name"] != "炽烈的炎之魔女4" {
		t.Fatalf("first row = %v", first)
	}
	if dmg := first["damage"].(map[string]any); dmg["title"] != "重击蒸发(半血开E)" || dmg["value"] != "64,248.5" {
		t.Fatalf("damage = %v", dmg)
	}
	// ark grades the average piece score, MAX from 56 on.
	for index, grade := range []string{"MAX", "SSS", "D"} {
		if got := rows[index].(map[string]any)["grade"]; got != grade {
			t.Errorf("row %d grade %v, want %s", index, got, grade)
		}
	}
	if _, has := rows[1].(map[string]any)["damage"]; has {
		t.Error("damage drawn without ark's dmg_avg")
	}
}
