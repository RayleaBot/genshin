package images_test

import (
	"testing"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestTrainingFollowsMiao(t *testing.T) {
	app, err := gamekit.New(assets.Kit(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list := decode(t, `{"list":[{"id":10000007,"level":90,"actived_constellation_num":6,"fetter":3},{"id":10000089,"level":90,"actived_constellation_num":2,"fetter":8}]}`)
	scored := 0
	context := gamekit.ImageContext{Game: app.Game, Catalog: app.Catalog, Now: time.Now(),
		Query: func(operation string, input map[string]any) (gamekit.QueryResult, error) {
			if operation == "genshin.profile" {
				return gamekit.QueryResult{}, nil
			}
			return gamekit.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000089,"level":90,"actived_constellation_num":2},
				"weapon":{"id":11513,"name":"静水流涌之辉","level":90,"affix_level":1,"rarity":5},
				"relics":[{"id":1,"pos":1,"set":{"name":"黄金剧团"}},{"id":2,"pos":2,"set":{"name":"黄金剧团"}}],
				"skills":[{"skill_id":10891,"skill_type":1,"level":6},{"skill_id":10892,"skill_type":1,"level":13,"extra_level":3},{"skill_id":10893,"skill_type":1,"level":10}]}]}`)}, nil
		},
		Score: func(panel gamekit.CharacterPanel) (gamekit.CharacterPanel, error) {
			scored++
			panel.ScoreDetail = &gamekit.ScoreDetail{Mark: "210.5", Grade: "ACE"}
			return panel, nil
		}}
	image, ok := images.Training(context, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: list})
	if !ok || image.Data["count"] != 2 {
		t.Fatalf("image = %v", image.Data)
	}
	rows := image.Data["rows"].([]any)
	// Furina's talents rank her before the traveler; only she has artifacts
	// to score.
	furina, traveler := rows[0].(map[string]any), rows[1].(map[string]any)
	if furina["name"] != "芙宁娜" || furina["no"] != 1 || furina["grade"] != "ACE" || furina["mark"] != "210.5" || scored != 1 {
		t.Errorf("furina = %v", furina)
	}
	weapon := furina["weapon"].(map[string]any)
	if weapon["name"] != "静水之辉" || weapon["badge"] != 2 {
		t.Errorf("weapon = %v", weapon)
	}
	talents := furina["talents"].([]any)
	if e := talents[1].(map[string]any); e["class"] != 5 || e["plus"] != true || talents[0].(map[string]any)["class"] != 3 {
		t.Errorf("talents = %v", talents)
	}
	// The traveler shows full friendship; without details miao prints "-".
	if traveler["fetter"] != 10 || traveler["talents"].([]any)[0].(map[string]any)["level"] != "-" || traveler["grade"] != nil {
		t.Errorf("traveler = %v", traveler)
	}
}
