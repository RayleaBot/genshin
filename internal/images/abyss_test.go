package images

import (
	"encoding/json"
	"strconv"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func TestAbyssFollowsYunzai(t *testing.T) {
	floor := func(index, star int) map[string]any {
		return map[string]any{"index": json.Number(strconv.Itoa(index)), "star": json.Number(strconv.Itoa(star))}
	}
	rank := func(id string, value int) []any {
		return []any{map[string]any{"avatar_id": json.Number(id), "value": json.Number(strconv.Itoa(value))}}
	}
	data := map[string]any{
		"start_time": "1788220800", "max_floor": "12-3", "total_battle_times": json.Number("12"),
		"floors":      []any{floor(8, 9), floor(9, 9), floor(10, 9), floor(11, 8), floor(12, 7)},
		"damage_rank": rank("10000089", 1234567), "take_damage_rank": rank("10000089", 800), "defeat_rank": rank("10000046", 25),
		"normal_skill_rank": rank("10000046", 30), "energy_skill_rank": rank("10000046", 12),
		"reveal_rank": []any{map[string]any{"avatar_id": json.Number("10000089"), "rarity": json.Number("5"), "value": json.Number("8")}},
	}
	image, ok := Abyss(gamekit.ImageContext{}, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: data})
	if !ok {
		t.Fatal("abyss")
	}
	ranks := image.Data["ranks"].(map[string]any)
	if image.Data["total_star"] != "33（9-9-8-7）" || image.Data["time"] != "9月" {
		t.Errorf("summary = %v %v", image.Data["total_star"], image.Data["time"])
	}
	if ranks["damage"].(map[string]any)["num"] != "123.5 w" || ranks["take_damage"].(map[string]any)["num"] != "800" || ranks["defeat"].(map[string]any)["num"] != "25" {
		t.Errorf("ranks = %v", ranks)
	}
	if used := image.Data["list"].([]any)[0].(map[string]any); used["value"] != 8 || used["life"] != 0 {
		t.Errorf("used = %v", used)
	}
	delete(data, "damage_rank")
	if _, ok := Abyss(gamekit.ImageContext{}, gamekit.QueryResult{Data: data}); ok {
		t.Error("a period without ranks should answer in text like upstream")
	}
}
