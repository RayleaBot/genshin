package images_test

import (
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
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
	asked := []string{}
	context := app.ImageContext{Game: application.Game, Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC), Query: func(operation string, input map[string]any) (app.QueryResult, error) {
		asked = append(asked, operation)
		if operation == "genshin.profile" {
			return app.QueryResult{Data: decode(t, `{"role":{"nickname":"旅行者","level":60},"stats":{"achievement_number":900,"way_point_number":400,"avatar_number":4,"active_day_number":800,"common_chest_number":3000,"magic_chest_number":500},
				"world_explorations":[{"name":"蒙德","exploration_percentage":1000},{"name":"层岩巨渊","exploration_percentage":10},{"name":"层岩巨渊·地下矿区","exploration_percentage":987}]}`)}, nil
		}
		// Hu Tao wears a five-star weapon at refinement 2; Furina's detail is
		// missing and falls back to the list.
		return app.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000046,"level":90,"actived_constellation_num":1},"weapon":{"id":13501,"level":90,"affix_level":2,"rarity":5},
			"skills":[{"skill_id":10461,"skill_type":1,"level":10},{"skill_id":10462,"skill_type":1,"level":10},{"skill_id":10463,"skill_type":1,"level":10}]}]}`)}, nil
	}}
	image, ok := images.Characters(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	if !ok {
		t.Fatal("characters")
	}
	if strings.Join(asked, ",") != "genshin.character,genshin.profile" {
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
}
