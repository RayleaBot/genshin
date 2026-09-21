package images

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// Abyss draws the Spiral Abyss the way Yunzai's html/abyss/abyss does: the
// period, deepest floor and stars of floors 9 to 12, the most used
// characters, and the five battle records with their characters.
func Abyss(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	data := result.Data
	// Upstream answers in text while the period has no battles or ranks yet.
	if damage, _ := data["damage_rank"].([]any); gamekit.Int(data["total_battle_times"]) <= 0 || len(damage) == 0 {
		return gamekit.Image{}, false
	}
	names := characterNames(context)
	resources := []rayleabot.RenderImageResource{}
	add := func(id, source, name string) bool {
		resource, ok := context.ArtworkResource(id, source, name)
		if ok {
			resources = append(resources, resource)
		}
		return ok
	}
	add("tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf")
	for _, image := range []string{"abyss/bg", "other/bg5", "other/bg4", "other/bg105", "other/fill"} {
		add("img-"+strings.ReplaceAll(image, "/", "-"), "yunzai-genshin", "resources/img/"+image+".png")
	}
	portraits := map[string]string{}
	portrait := func(id, kind string) string {
		resource := kind + "-" + gamekit.Text(id)
		if known, seen := portraits[resource]; seen {
			return known
		}
		name := names[gamekit.Text(id)]
		if name == "" || !add(resource, "miao-plugin", "resources/meta-gs/character/"+name+"/imgs/"+kind+".webp") {
			resource = ""
		}
		portraits[kind+"-"+gamekit.Text(id)] = resource
		return resource
	}

	start := time.Unix(int64(gamekit.Int(data["start_time"])), 0).In(chinaTime)
	stars := []string{}
	total := 0
	floors, _ := data["floors"].([]any)
	for _, raw := range floors {
		floor, _ := raw.(map[string]any)
		if gamekit.Int(floor["index"]) < 9 {
			continue
		}
		total += gamekit.Int(floor["star"])
		stars = append(stars, gamekit.Text(floor["star"]))
	}

	ranks := map[string]any{}
	for _, key := range []string{"damage", "take_damage", "defeat", "normal_skill", "energy_skill"} {
		list, _ := data[key+"_rank"].([]any)
		id, value := "10000007", 0
		if len(list) > 0 {
			first, _ := list[0].(map[string]any)
			id, value = gamekit.Text(first["avatar_id"]), gamekit.Int(first["value"])
		}
		num := strconv.Itoa(value)
		if value > 1000 {
			num = fmt.Sprintf("%.1f w", float64(value)/10000)
		}
		ranks[key] = map[string]any{"num": num, "icon": portrait(id, "side")}
	}
	used := []any{}
	reveals, _ := data["reveal_rank"].([]any)
	for _, raw := range reveals {
		reveal, _ := raw.(map[string]any)
		id := gamekit.Text(reveal["avatar_id"])
		// Upstream shows the badge only when the entry carries "life".
		used = append(used, map[string]any{"life": gamekit.Int(reveal["life"]), "rarity": gamekit.Int(reveal["rarity"]), "icon": portrait(id, "face"), "value": gamekit.Int(reveal["value"])})
	}
	return gamekit.Image{Template: "abyss", Data: map[string]any{
		"uid": result.Role.UID, "time": strconv.Itoa(int(start.Month())) + "月",
		"max_floor": gamekit.Text(data["max_floor"]), "total_star": strconv.Itoa(total) + "（" + strings.Join(stars, "-") + "）",
		"list": used, "total_battle_times": gamekit.Int(data["total_battle_times"]), "ranks": ranks,
	}, Resources: resources}, true
}

// characterNames maps character IDs to the names miao keeps its images under.
func characterNames(context gamekit.ImageContext) map[string]string {
	names := map[string]string{}
	if context.Game.Calc == nil {
		return names
	}
	for _, record := range context.Game.Calc.Metadata().Characters {
		names[record.ID] = record.Name
	}
	return names
}
