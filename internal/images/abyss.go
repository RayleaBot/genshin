package images

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// Abyss draws the Spiral Abyss the way Yunzai's html/abyss/abyss does: the
// period, deepest floor and stars of floors 9 to 12, the most used
// characters, and the five battle records with their characters.
func Abyss(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	// Upstream answers in text while the period has no battles or ranks yet.
	if damage, _ := data["damage_rank"].([]any); app.Int(data["total_battle_times"]) <= 0 || len(damage) == 0 {
		return app.Image{}, false
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
		resource := kind + "-" + app.Text(id)
		if known, seen := portraits[resource]; seen {
			return known
		}
		name := names[app.Text(id)]
		if name == "" || !add(resource, "miao-plugin", "resources/meta-gs/character/"+name+"/imgs/"+kind+".webp") {
			resource = ""
		}
		portraits[kind+"-"+app.Text(id)] = resource
		return resource
	}

	start := time.Unix(int64(app.Int(data["start_time"])), 0).In(chinaTime)
	stars := []string{}
	total := 0
	floors, _ := data["floors"].([]any)
	for _, raw := range floors {
		floor, _ := raw.(map[string]any)
		if app.Int(floor["index"]) < 9 {
			continue
		}
		total += app.Int(floor["star"])
		stars = append(stars, app.Text(floor["star"]))
	}

	ranks := map[string]any{}
	for _, key := range []string{"damage", "take_damage", "defeat", "normal_skill", "energy_skill"} {
		list, _ := data[key+"_rank"].([]any)
		id, value := "10000007", 0
		if len(list) > 0 {
			first, _ := list[0].(map[string]any)
			id, value = app.Text(first["avatar_id"]), app.Int(first["value"])
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
		id := app.Text(reveal["avatar_id"])
		// Upstream shows the badge only when the entry carries "life".
		used = append(used, map[string]any{"life": app.Int(reveal["life"]), "rarity": app.Int(reveal["rarity"]), "icon": portrait(id, "face"), "value": app.Int(reveal["value"])})
	}
	return app.Image{Template: "abyss", Data: map[string]any{
		"uid": result.Role.UID, "time": strconv.Itoa(int(start.Month())) + "月",
		"max_floor": app.Text(data["max_floor"]), "total_star": strconv.Itoa(total) + "（" + strings.Join(stars, "-") + "）",
		"list": used, "total_battle_times": app.Int(data["total_battle_times"]), "ranks": ranks,
	}, Resources: resources}, true
}

// characterNames maps character IDs to the names miao keeps its images under.
func characterNames(context app.ImageContext) map[string]string {
	names := map[string]string{}
	if context.Game.Calc == nil {
		return names
	}
	for _, record := range context.Game.Calc.Metadata().Characters {
		names[record.ID] = record.Name
	}
	return names
}
