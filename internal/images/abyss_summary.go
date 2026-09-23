package images

import (
	"slices"
	"strconv"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// AbyssSummary draws this period's Spiral Abyss the way miao's
// stat/abyss-summary does: the month and battles, the strongest hit and the
// most damage taken with the character's portrait, the most defeats and
// skill uses, then each floor with the teams of its last chamber as avatar
// cards and every chamber's stars, time and faces of both halves.
func AbyssSummary(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	data := result.Data
	floors, _ := data["floors"].([]any)
	// Upstream answers in text until the first chamber has battles.
	if len(floors) == 0 {
		return app.Image{}, false
	}
	first, _ := floors[0].(map[string]any)
	levels, _ := first["levels"].([]any)
	if len(levels) == 0 {
		return app.Image{}, false
	}
	if _, ok := levels[0].(map[string]any)["battles"]; !ok {
		return app.Image{}, false
	}

	type half struct {
		ids  []string
		time string
	}
	type chamber struct {
		index, star int
		up, down    half
	}
	type floor struct {
		index, star int
		chambers    []chamber
	}
	ids := []string{}
	list := []floor{}
	for _, rawFloor := range floors {
		rawFloor, _ := rawFloor.(map[string]any)
		entry := floor{index: app.Int(rawFloor["index"]), star: app.Int(rawFloor["star"])}
		rawLevels, _ := rawFloor["levels"].([]any)
		for _, rawLevel := range rawLevels {
			rawLevel, _ := rawLevel.(map[string]any)
			level := chamber{index: app.Int(rawLevel["index"]), star: app.Int(rawLevel["star"])}
			battles, _ := rawLevel["battles"].([]any)
			for _, rawBattle := range battles {
				battle, _ := rawBattle.(map[string]any)
				side := half{time: time.Unix(int64(app.Int(battle["timestamp"])), 0).In(chinaTime).Format("01-02 15:04:05")}
				avatars, _ := battle["avatars"].([]any)
				for _, rawAvatar := range avatars {
					avatar, _ := rawAvatar.(map[string]any)
					id := app.Text(avatar["id"])
					side.ids = append(side.ids, id)
					ids = append(ids, id)
				}
				if app.Int(battle["index"]) == 1 {
					level.up = side
				} else {
					level.down = side
				}
			}
			entry.chambers = append(entry.chambers, level)
		}
		slices.SortFunc(entry.chambers, func(a, b chamber) int { return a.index - b.index })
		list = append(list, entry)
	}
	slices.SortFunc(list, func(a, b floor) int { return a.index - b.index })

	leader := func(key string) (string, int) {
		ranks, _ := data[key].([]any)
		if len(ranks) == 0 {
			return "", 0
		}
		rank, _ := ranks[0].(map[string]any)
		return app.Text(rank["avatar_id"]), app.Int(rank["value"])
	}
	type record struct{ title, id, value string }
	records := []record{}
	// miao leaves out a damage record without a character.
	for _, item := range [][2]string{{"最强一击", "damage_rank"}, {"最高承伤", "take_damage_rank"}} {
		if id, value := leader(item[1]); id != "" {
			records = append(records, record{item[0], id, jsFixed(float64(value)/10000, 1) + " W"})
		}
	}
	for _, item := range [][2]string{{"最多击破", "defeat_rank"}, {"元素战技", "normal_skill_rank"}, {"元素爆发", "energy_skill_rank"}} {
		id, value := leader(item[1])
		records = append(records, record{item[0], id, strconv.Itoa(value) + "次"})
	}
	for _, item := range records {
		ids = append(ids, item.id)
	}
	slices.Sort(ids)
	ids = slices.DeleteFunc(slices.Compact(ids), func(id string) bool { return id == "" })

	resources := &app.ImageResources{Context: context}
	for _, item := range statArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	resources.Artwork("stat-imgs-star", "miao-plugin", "resources/stat/imgs/star.png")
	cards := newAvatarCards(context, resources, ids)
	// Characters the query does not return get miao's empty card.
	card := func(id string) map[string]any {
		if own, ok := cards.own(id); ok {
			return own
		}
		return map[string]any{}
	}

	stats := []any{}
	for _, item := range records {
		stat := map[string]any{"title": item.title, "value": item.value}
		if own, ok := cards.own(item.id); ok {
			stat["name"], stat["banner"], stat["shown"] = own["name"], own["gacha"], true
		}
		// Without HuTao's upload miao has no percentiles to compare with.
		if item.title == "最强一击" || item.title == "最高承伤" {
			stat["notes"] = []string{"暂无统计信息"}
		}
		stats = append(stats, stat)
	}
	faces := func(ids []string) []any {
		icons := []any{}
		for _, id := range ids {
			own := card(id)
			star := 5
			if app.Int(own["star"]) == 4 {
				star = 4
			}
			icons = append(icons, map[string]any{"star": star, "face": own["face"]})
		}
		return icons
	}
	team := func(ids []string) []any {
		members := []any{}
		for position, id := range ids {
			member := card(id)
			if types := statCardTypes[len(ids)]; position < len(types) {
				member["type"] = types[position]
			}
			members = append(members, member)
		}
		return members
	}
	floorViews := []any{}
	for _, entry := range list {
		// The floor shows the teams of its last chamber.
		var display chamber
		if len(entry.chambers) > 0 {
			display = entry.chambers[len(entry.chambers)-1]
		}
		chambers := []any{}
		for _, level := range entry.chambers {
			chambers = append(chambers, map[string]any{"index": level.index, "star": level.star, "time": level.up.time, "up": faces(level.up.ids), "down": faces(level.down.ids)})
		}
		floorViews = append(floorViews, map[string]any{"index": entry.index, "star": entry.star, "up": team(display.up.ids), "down": team(display.down.ids), "chambers": chambers})
	}
	start := time.Unix(int64(app.Int(data["start_time"])), 0).In(chinaTime)
	return app.Image{Template: "stat-abyss-summary", Data: map[string]any{
		"uid": result.Role.UID, "month": strconv.Itoa(int(start.Month())) + "月", "total": app.Int(data["total_battle_times"]),
		"stats": stats, "floors": floorViews, "updated": context.Now.In(chinaTime).Format("01-02 15:04:05"),
	}, Resources: resources.List}, true
}
