package images_test

import (
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestRoleExploreFollowsYunzai(t *testing.T) {
	data := decode(t, `{"role":{"nickname":"旅行者","level":60,"region":"cn_gf01"},
		"stats":{"active_day_number":900,"avatar_number":100,"full_fetter_avatar_num":3,"common_chest_number":3000,"exquisite_chest_number":2500,
			"precious_chest_number":900,"luxurious_chest_number":350,"magic_chest_number":300,
			"role_combat":{"is_unlock":true,"has_detail_data":true,"max_round_id":10,"tarot_finished_cnt":2},"hard_challenge":{"is_unlock":true,"has_data":true,"difficulty":4}},
		"world_explorations":[{"id":6,"name":"层岩巨渊","exploration_percentage":1000,"offerings":[{"name":"流明石触媒","level":10}]},
			{"id":7,"name":"层岩巨渊·地下矿区","exploration_percentage":987},{"id":1,"name":"蒙德","exploration_percentage":1000,"level":8},
			{"id":99,"name":"很长很长的新地区名字","exploration_percentage":250}]}`)
	context := app.ImageContext{Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC), Word: "探索"}
	image, ok := images.Profile(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok || image.Template != "role-explore" {
		t.Fatalf("image = %v", image)
	}
	lines := image.Data["lines"].([]any)
	first := lines[0].([]any)
	// Days since release, the theater's act and tarot count, the onslaught's roman difficulty.
	if first[0].(map[string]any)["extra"] != "2198" || first[2].(map[string]any)["num"] != "第10幕 圣牌2" || first[3].(map[string]any)["num"] != "IV" {
		t.Errorf("first line = %v", first)
	}
	// 7050 of Yunzai's 9109 chests is 77.4%: grade B.
	if rate := lines[2].([]any)[1].(map[string]any); rate["num"] != "B[77.4%]" || rate["color"] != "#806d9e" {
		t.Errorf("chest rate = %v", rate)
	}
	// The underground mine joins the Chasm with its offering; unknown areas are cut to six characters.
	areas := image.Data["explorations"].([]any)
	chasm := areas[1].(map[string]any)["lines"].([]any)
	if len(areas) != 3 || len(chasm) != 3 || chasm[1].(map[string]any)["text"] != "98.7%" || chasm[2].(map[string]any)["name"] != "流明石" {
		t.Errorf("areas = %v", areas)
	}
	if name := areas[0].(map[string]any)["lines"].([]any)[0].(map[string]any)["name"]; name != "很长很..." {
		t.Errorf("long name = %v", name)
	}
	if image, _ := images.Profile(app.ImageContext{Word: "角色卡片"}, app.QueryResult{Data: data}); image.Template != "role-card" {
		t.Errorf("角色卡片 drew %s", image.Template)
	}
}
