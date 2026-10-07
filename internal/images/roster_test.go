package images_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/artwork"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestCharactersFollowMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list := decode(t, `{"list":[
		{"id":10000025,"level":80,"rarity":4,"actived_constellation_num":6,"fetter":10},
		{"id":10000046,"level":90,"rarity":5,"actived_constellation_num":1,"fetter":10},
		{"id":10000007,"level":90,"rarity":5,"actived_constellation_num":6,"fetter":10},
		{"id":10000089,"level":90,"rarity":5,"actived_constellation_num":2,"fetter":8}]}`)
	index := decode(t, `{"role":{"nickname":"旅行者","level":60},"stats":{"achievement_number":900,"way_point_number":400,"avatar_number":4,"active_day_number":800,"common_chest_number":3000,"magic_chest_number":500},
		"world_explorations":[{"name":"蒙德","exploration_percentage":1000},{"name":"层岩巨渊","exploration_percentage":10},{"name":"层岩巨渊·地下矿区","exploration_percentage":987}]}`)
	asked := []string{}
	context := app.ImageContext{Game: application.Game, Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC), Query: func(operation string, input map[string]any) (app.QueryResult, error) {
		asked = append(asked, operation)
		// Hu Tao wears a five-star weapon at refinement 2; Furina's detail is
		// missing and falls back to the list.
		return app.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000046,"level":90,"actived_constellation_num":1},"weapon":{"id":13501,"level":90,"affix_level":2,"rarity":5},
			"skills":[{"skill_id":10461,"skill_type":1,"level":10},{"skill_id":10462,"skill_type":1,"level":10},{"skill_id":10463,"skill_type":1,"level":10}]}]}`)}, nil
	}}
	characters := list["list"].([]any)
	image, ok := images.Characters(context, app.CharactersImage{Role: app.Role{UID: "100000001"}, Characters: characters, Index: index})
	if !ok {
		t.Fatal("characters")
	}
	if strings.Join(asked, ",") != "genshin.character" {
		t.Errorf("asked = %v", asked)
	}
	// Level, rarity, then talents: Hu Tao's talents put her before the
	// traveler and Furina, and the four-star at level 80 comes last.
	avatars := image.Data["avatars"].([]any)
	order := []string{}
	for _, avatar := range avatars {
		order = append(order, avatar.(map[string]any)["name"].(string))
	}
	if order[0] != "胡桃" || order[3] != "行秋" {
		t.Errorf("order = %v", order)
	}
	stats := map[string]any{}
	for _, stat := range image.Data["stats"].([]any) {
		stats[stat.(map[string]any)["title"].(string)] = stat.(map[string]any)["value"]
	}
	// Gold cards: Hu Tao 1+1 and her weapon 2, Furina 2+1; the traveler does
	// not count.
	if stats["五星角色"] != 3 || stats["金卡总数"] != 7 || stats["成就"] != 900 {
		t.Errorf("stats = %v", stats)
	}
	player := image.Data["player"].(map[string]any)
	if player["name"] != "旅行者" || player["active"] != "2年2个月9天" || player["show_level"] != true {
		t.Errorf("player = %v", player)
	}
	exploration := image.Data["exploration"].([]any)
	if exploration[0].(map[string]any)["value"] != "100%" || exploration[3].(map[string]any)["value"] != "98.7%" || exploration[2].(map[string]any)["value"] != "0%" {
		t.Errorf("exploration = %v", exploration)
	}
	if chests := image.Data["chests"].([]any); chests[4].(map[string]any)["max"] != 500 || chests[0].(map[string]any)["max"] != 3838 {
		t.Errorf("chests = %v", chests)
	}
	// 四星角色 lists only the four-stars, under the strongest of them, and
	// keeps the whole roster's counts.
	context.Word = "四星角色"
	image, _ = images.Characters(context, app.CharactersImage{Role: app.Role{UID: "100000001"}, Characters: characters, Index: index})
	if avatars := image.Data["avatars"].([]any); len(avatars) != 1 || avatars[0].(map[string]any)["name"] != "行秋" {
		t.Errorf("四星角色 avatars = %v", avatars)
	}
	if stats := image.Data["stats"].([]any); len(stats) != 5 {
		t.Errorf("四星角色 stats = %v", stats)
	}
}

// As miao draws a UID's player data, a kept panel stands in where the
// account gives no detail, the list's level and the weapon it names count
// over it unless miao does not know the weapon, the kept profile picture
// heads the page and the last 更新面板 is dated; a UID read without its
// account carries miao's notice.
func TestCharactersDrawKeptPanels(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	banner := filepath.Join(root, "miao-plugin", filepath.FromSlash("resources/meta-gs/character/芙宁娜/imgs/banner.webp"))
	if err := os.MkdirAll(filepath.Dir(banner), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(banner, []byte("webp"), 0o600); err != nil {
		t.Fatal(err)
	}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC), Artwork: &artwork.Store{Root: root}}
	kept := app.SavedProfiles{Face: "10000089", Nickname: "展柜", RefreshedAtMS: time.Date(2026, 9, 20, 1, 30, 0, 0, time.UTC).UnixMilli(), Panels: map[string]app.SavedPanel{
		"10000046": {Panel: app.CharacterPanel{ID: "10000046", Level: 80, Rank: 1, Weapon: &app.PanelEquipment{ID: "13501", Level: 90, Rarity: "5", Refinement: 1}}},
		"10000089": {Panel: app.CharacterPanel{ID: "10000089", Level: 90, Weapon: &app.PanelEquipment{ID: "11513", Level: 90, Rarity: "5", Refinement: 1}}},
	}}
	characters := decode(t, `{"list":[{"id":10000046,"level":90,"actived_constellation_num":0,"weapon":{"id":13415,"name":"「渔获」","rarity":4,"level":90,"affix_level":5}},
		{"id":10000025,"level":90,"actived_constellation_num":6,"weapon":{"id":11401,"name":"西风剑","rarity":4,"level":80,"affix_level":2}},
		{"id":10000089,"level":90,"weapon":{"id":19999,"name":"未知","rarity":4,"level":1,"affix_level":1}}]}`)["list"].([]any)
	image, ok := images.Characters(context, app.CharactersImage{Role: app.Role{UID: "100000001"}, Characters: characters, Saved: kept, Public: true})
	if !ok || image.Data["notice"] != true {
		t.Fatal("characters")
	}
	avatars := map[string]map[string]any{}
	for _, raw := range image.Data["avatars"].([]any) {
		avatar := raw.(map[string]any)
		avatars[avatar["name"].(string)] = avatar
	}
	weapon := func(name string) map[string]any { return avatars[name]["weapon"].(map[string]any) }
	if hutao := avatars["胡桃"]; hutao["level"] != 90 || hutao["cons"] != 1 || weapon("胡桃")["star"] != 4 || weapon("胡桃")["badge"] != 6 {
		t.Errorf("胡桃 = %v", hutao)
	}
	if weapon("行秋")["affix"] != 2 || weapon("芙宁娜")["star"] != 5 {
		t.Errorf("行秋 = %v, 芙宁娜 = %v", avatars["行秋"], avatars["芙宁娜"])
	}
	player := image.Data["player"].(map[string]any)
	if player["name"] != "展柜" || image.Data["profile_updated"] != "09-20 09:30" || image.Data["updated"] != "09-21 20:00" {
		t.Errorf("player = %v, dates %v %v", player, image.Data["profile_updated"], image.Data["updated"])
	}
	if player["banner"] == "" {
		t.Error("the kept profile picture's banner is missing")
	}
}
