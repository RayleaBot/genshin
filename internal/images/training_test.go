package images_test

import (
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

// A UID on the requester's account is drawn with the account's details,
// which count over the panels kept for it.
func TestTrainingFollowsMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list := decode(t, `{"list":[{"id":10000007,"level":90,"actived_constellation_num":6,"fetter":3},{"id":10000089,"level":90,"actived_constellation_num":2,"fetter":8}]}`)["list"].([]any)
	player := app.CharactersImage{Role: app.Role{UID: "100000001"}, Characters: list, Saved: app.SavedProfiles{Panels: map[string]app.SavedPanel{
		"10000089": {Panel: app.CharacterPanel{ID: "10000089", Level: 90, Skills: []app.PanelSkill{{SkillType: 1, ID: "10891", Level: 1}, {SkillType: 1, ID: "10892", Level: 1}, {SkillType: 1, ID: "10893", Level: 1}}}},
	}}}
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
	image, ok := images.Training(context, player)
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
	image, _ = images.Training(context, player)
	rows = image.Data["rows"].([]any)
	book, _ := rows[0].(map[string]any)["book"].(map[string]any)
	if image.Data["talent"] != true || len(rows) != 1 || book["label"] != "枫丹·正义" || book["week"] != "2/5" || scored != 1 {
		t.Errorf("talent rows = %v", rows)
	}
	context.Word = "周一天赋统计"
	image, _ = images.Training(context, player)
	for _, raw := range image.Data["rows"].([]any) {
		if raw.(map[string]any)["name"] == "芙宁娜" {
			t.Error("周一 kept Furina's Tuesday book")
		}
	}
	// miao's yzRule words keep the stars and elements they name.
	context.Word = "水角色统计"
	image, _ = images.Training(context, player)
	if rows = image.Data["rows"].([]any); image.Data["talent"] != false || len(rows) != 1 || rows[0].(map[string]any)["name"] != "芙宁娜" {
		t.Errorf("水角色统计 rows = %v", rows)
	}
	context.Word = "四星列表"
	image, _ = images.Training(context, player)
	if rows = image.Data["rows"].([]any); len(rows) != 0 {
		t.Errorf("四星列表 rows = %v", rows)
	}
}

// A UID read without its account is drawn from its player data as miao
// draws it: a kept panel gives talents and scored artifacts under the weapon
// the list names, a character only the list has shows no talents but its
// weapon, and the last 更新面板 is dated beside the official data.
func TestTrainingDrawsKeptPanels(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
		Score: func(panel app.CharacterPanel) (app.CharacterPanel, error) {
			panel.ScoreDetail = &app.ScoreDetail{Mark: "180.2", Grade: "SS"}
			return panel, nil
		}}
	kept := app.SavedProfiles{RefreshedAtMS: time.Date(2026, 9, 21, 1, 30, 0, 0, time.UTC).UnixMilli(), Panels: map[string]app.SavedPanel{
		"10000089": {Panel: app.CharacterPanel{ID: "10000089", Level: 90, Rank: 2, Weapon: &app.PanelEquipment{ID: "11513", Level: 90, Rarity: "5", Refinement: 1},
			Equipment: []app.PanelEquipment{{Slot: 1, SetName: "黄金剧团"}, {Slot: 2, SetName: "黄金剧团"}},
			Skills:    []app.PanelSkill{{SkillType: 1, ID: "10891", Level: 9}, {SkillType: 1, ID: "10892", Level: 13, ExtraLevel: 3}, {SkillType: 1, ID: "10893", Level: 10}}}},
	}}
	characters := decode(t, `{"list":[{"id":10000089,"level":90,"actived_constellation_num":2,"weapon":{"id":11401,"name":"西风剑","rarity":4,"level":90,"affix_level":3}},
		{"id":10000025,"level":80,"actived_constellation_num":6,"weapon":{"id":11501,"name":"风鹰剑","rarity":5,"level":90,"affix_level":1}}]}`)["list"].([]any)
	image, ok := images.Training(context, app.CharactersImage{Role: app.Role{UID: "100000001"}, Characters: characters, Saved: kept, Public: true})
	if !ok || image.Data["count"] != 2 || image.Data["profile_updated"] != "09-21 09:30" || image.Data["updated"] != "09-22 20:00" {
		t.Fatalf("image = %v", image.Data)
	}
	rows := image.Data["rows"].([]any)
	furina, xingqiu := rows[0].(map[string]any), rows[1].(map[string]any)
	talents := furina["talents"].([]any)
	if furina["name"] != "芙宁娜" || furina["weapon"].(map[string]any)["name"] != "西风剑" || furina["grade"] != "SS" || len(furina["artis"].([]any)) != 1 ||
		talents[0].(map[string]any)["level"] != 9 || talents[1].(map[string]any)["plus"] != true {
		t.Errorf("芙宁娜 = %v", furina)
	}
	if xingqiu["name"] != "行秋" || xingqiu["weapon"].(map[string]any)["name"] != "风鹰剑" || xingqiu["grade"] != nil ||
		xingqiu["talents"].([]any)[0].(map[string]any)["level"] != "-" {
		t.Errorf("行秋 = %v", xingqiu)
	}
}
