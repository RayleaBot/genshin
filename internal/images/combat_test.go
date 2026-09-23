package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

func TestCombatFollowsYunzai(t *testing.T) {
	round := func(id string, medal bool, kind string) map[string]any {
		return map[string]any{"round_id": json.Number(id), "is_get_medal": medal, "is_tarot": false,
			"avatars": []any{map[string]any{"name": "芙宁娜", "element": "Hydro", "rarity": json.Number("5"), "level": json.Number("90"), "avatar_type": json.Number(kind)}},
			"buffs":   []any{map[string]any{"icon": "http://insecure.example/buff.png"}}}
	}
	data := map[string]any{"has_detail_data": true, "data": []any{map[string]any{
		"stat":   map[string]any{"difficulty_id": json.Number("4"), "max_round_id": json.Number("10"), "heraldry": json.Number("3"), "get_medal_round_list": []any{json.Number("1"), json.Number("0")}, "coin_num": json.Number("22"), "avatar_bonus_num": json.Number("6"), "rent_cnt": json.Number("3")},
		"detail": map[string]any{"rounds_data": []any{round("1", true, "3"), round("2", false, "1")}},
	}}}
	image, ok := Combat(app.ImageContext{}, app.QueryResult{Role: app.Role{UID: "100000001", Nickname: "旅行者", Level: 60}, Data: data})
	if !ok || image.Data["difficulty"] != "卓越" || image.Data["max_round"] != "10" || image.Data["heraldry"] != "3" {
		t.Fatalf("image = %v", image.Data)
	}
	if medals := image.Data["medals"].([]any); len(medals) != 2 || medals[0] != 1 {
		t.Errorf("medals = %v", medals)
	}
	rounds := image.Data["rounds"].([]any)
	first := rounds[0].(map[string]any)
	cast := first["cast"].([]any)[0].(map[string]any)
	if first["medal"] != true || cast["type_name"] != "助演" || rounds[1].(map[string]any)["cast"].([]any)[0].(map[string]any)["type_name"] != "" {
		t.Errorf("rounds = %v", rounds)
	}
	// Only https official images are cached; others are left out.
	if buffs := first["buffs"].([]any); len(buffs) != 0 {
		t.Errorf("buffs = %v", buffs)
	}
	// 上期 reads the second period.
	last := map[string]any{"stat": map[string]any{"difficulty_id": json.Number("2"), "max_round_id": json.Number("8")}, "detail": map[string]any{"rounds_data": []any{round("1", false, "1")}}}
	data["data"] = append(data["data"].([]any), last)
	if image, ok := Combat(app.ImageContext{Word: "上期剧诗"}, app.QueryResult{Data: data}); !ok || image.Data["max_round"] != "8" {
		t.Errorf("last period = %v %v", ok, image.Data["max_round"])
	}
	data["has_detail_data"] = "false"
	if _, ok := Combat(app.ImageContext{}, app.QueryResult{Data: data}); ok {
		t.Error("a season without detail should answer in text like upstream")
	}
}
