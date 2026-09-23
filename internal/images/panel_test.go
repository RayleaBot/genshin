package images_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/artwork"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
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
