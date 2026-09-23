package images_test

import (
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestTrainingFollowsMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list := decode(t, `{"list":[{"id":10000007,"level":90,"actived_constellation_num":6,"fetter":3},{"id":10000089,"level":90,"actived_constellation_num":2,"fetter":8}]}`)
	scored := 0
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Now(),
		Query: func(operation string, input map[string]any) (app.QueryResult, error) {
			if operation == "genshin.profile" {
				return app.QueryResult{}, nil
			}
			return app.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000089,"level":90,"actived_constellation_num":2},
				"weapon":{"id":11513,"name":"静水流涌之辉","level":90,"affix_level":1,"rarity":5},
				"relics":[{"id":1,"pos":1,"set":{"name":"黄金剧团"}},{"id":2,"pos":2,"set":{"name":"黄金剧团"}}],
				"skills":[{"skill_id":10891,"skill_type":1,"level":6},{"skill_id":10892,"skill_type":1,"level":13,"extra_level":3},{"skill_id":10893,"skill_type":1,"level":10}]}]}`)}, nil
		},
		Score: func(panel app.CharacterPanel) (app.CharacterPanel, error) {
			scored++
			panel.ScoreDetail = &app.ScoreDetail{Mark: "210.5", Grade: "ACE"}
			return panel, nil
		}}
	image, ok := images.Training(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
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
	// 天赋统计 keeps the characters whose book the weekday names and skips
	// the scoring; Furina's 正义 books drop on Tuesday and Friday.
	context.Word = "周二五星天赋统计"
	image, _ = images.Training(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	rows = image.Data["rows"].([]any)
	book, _ := rows[0].(map[string]any)["book"].(map[string]any)
	if image.Data["talent"] != true || len(rows) != 1 || book["label"] != "枫丹·正义" || book["week"] != "2/5" || scored != 1 {
		t.Errorf("talent rows = %v", rows)
	}
	context.Word = "周一天赋统计"
	image, _ = images.Training(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	for _, raw := range image.Data["rows"].([]any) {
		if raw.(map[string]any)["name"] == "芙宁娜" {
			t.Error("周一 kept Furina's Tuesday book")
		}
	}
	// miao's yzRule words keep the stars and elements they name.
	context.Word = "水角色统计"
	image, _ = images.Training(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	if rows = image.Data["rows"].([]any); image.Data["talent"] != false || len(rows) != 1 || rows[0].(map[string]any)["name"] != "芙宁娜" {
		t.Errorf("水角色统计 rows = %v", rows)
	}
	context.Word = "四星列表"
	image, _ = images.Training(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	if rows = image.Data["rows"].([]any); len(rows) != 0 {
		t.Errorf("四星列表 rows = %v", rows)
	}
}
