package images

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// roleAreas are Yunzai's names for the exploration areas, by ID.
var roleAreas = map[int]string{1: "蒙德", 2: "璃月", 3: "雪山", 4: "稻妻", 5: "渊下宫", 6: "层岩巨渊", 7: "层岩地下", 8: "须弥", 9: "枫丹", 10: "沉玉谷",
	11: "来歆山", 12: "沉玉谷·南陵", 13: "沉玉谷·上谷", 14: "旧日之海", 15: "纳塔", 16: "远古圣山", 17: "挪德卡莱", 18: "风息山", 19: "空之神殿", 20: "至冬"}

// roleTotals are the totals Yunzai's defSet/role/index.yaml gives the
// explore page, from the pinned snapshot; its cryoculus total is filed under a
// key the page does not read, so Cryoculi show none.
var roleTotals = map[string]int{"achievement": 1844, "avatar": 121, "magic_chest": 416, "luxurious_chest": 410, "precious_chest": 1073,
	"exquisite_chest": 3372, "common_chest": 3838, "way_point": 839, "domain": 74, "anemoculus": 66, "geoculus": 131, "electroculus": 181,
	"dendroculus": 271, "hydroculus": 271, "pyroculus": 271, "moonoculus": 271}

// roleElements are the classes the role card colours characters by.
var roleElements = map[string]string{"Pyro": "火", "Hydro": "水", "Anemo": "风", "Electro": "雷", "Dendro": "草", "Cryo": "冰", "Geo": "岩"}

// roleAreaName is Yunzai's name for an area, or the official one cut to six
// characters as lodash.truncate does.
func roleAreaName(area map[string]any) string {
	if name := roleAreas[gamekit.Int(area["id"])]; name != "" {
		return name
	}
	name := []rune(gamekit.Text(area["name"]))
	if len(name) > 6 {
		return string(name[:3]) + "..."
	}
	return string(name)
}

// roleExplorations are the areas newest first, as Yunzai orders them.
func roleExplorations(data map[string]any) []map[string]any {
	list, _ := data["world_explorations"].([]any)
	areas := []map[string]any{}
	for _, raw := range list {
		if area, ok := raw.(map[string]any); ok {
			areas = append(areas, area)
		}
	}
	slices.SortStableFunc(areas, func(a, b map[string]any) int { return gamekit.Int(b["id"]) - gamekit.Int(a["id"]) })
	return areas
}

func rolePercent(area map[string]any) string {
	return strconv.FormatFloat(float64(gamekit.Int(area["exploration_percentage"]))/10, 'f', -1, 64) + "%"
}

func roleChests(stats map[string]any) int {
	return gamekit.Int(stats["precious_chest_number"]) + gamekit.Int(stats["luxurious_chest_number"]) + gamekit.Int(stats["exquisite_chest_number"]) +
		gamekit.Int(stats["common_chest_number"]) + gamekit.Int(stats["magic_chest_number"])
}

// Profile draws the account summary with Yunzai's player pages: 角色卡片 and
// 角色3 draw html/player/role-card, the other words html/player/role-explore.
func Profile(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	if strings.Contains(context.Word, "角色") {
		return RoleCard(context, result)
	}
	return RoleExplore(context, result)
}

// RoleCard draws 角色卡片 the way Yunzai's html/player/role-card does: the
// requester's avatar, group card and UID, the active days, achievements,
// characters, level and chests of each kind, the first ten areas' exploration,
// then the Oculi and domains, and the first eight characters with their
// constellations and levels. Upstream strips the UID from the group card; the
// card here is the one the host passes.
func RoleCard(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	stats, _ := result.Data["stats"].(map[string]any)
	if stats == nil {
		return gamekit.Image{}, false
	}
	role, _ := result.Data["role"].(map[string]any)
	item := func(label string, value any) map[string]any {
		return map[string]any{"label": label, "num": gamekit.Text(value)}
	}
	lines := []any{
		[]any{item("活跃天数", stats["active_day_number"]), item("成就", stats["achievement_number"]), item("角色数", stats["avatar_number"]),
			item("等级", gamekit.Int(role["level"])), item("总宝箱", roleChests(stats))},
		[]any{item("华丽宝箱", stats["luxurious_chest_number"]), item("珍贵宝箱", stats["precious_chest_number"]), item("精致宝箱", stats["exquisite_chest_number"]),
			item("普通宝箱", stats["common_chest_number"]), item("奇馈宝箱", stats["magic_chest_number"]), item("传送点", stats["way_point_number"])},
	}
	first, rest := []any{}, []any{}
	for _, area := range roleExplorations(result.Data) {
		if len(first) < 5 {
			first = append(first, item(roleAreaName(area), rolePercent(area)))
		} else {
			rest = append(rest, item(roleAreaName(area), rolePercent(area)))
		}
	}
	for _, oculus := range [][2]string{{"冰神瞳", "iceculus_number"}, {"月神瞳", "moonoculus_number"}, {"火神瞳", "pyroculus_number"}, {"水神瞳", "hydroculus_number"},
		{"草神瞳", "dendroculus_number"}, {"雷神瞳", "electroculus_number"}, {"岩神瞳", "geoculus_number"}, {"风神瞳", "anemoculus_number"}, {"秘境", "domain_number"}} {
		rest = append(rest, item(oculus[0], stats[oculus[1]]))
	}
	lines = append(lines, first, rest[:5])
	resources := &gamekit.ImageResources{Context: context}
	resources.Artwork("tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf")
	resources.Artwork("img-roleCard-bg1", "yunzai-genshin", "resources/img/roleCard/bg1.jpg")
	resources.Artwork("genshin-logo", "yunzai-genshin", "resources/img/other/原神.png")
	avatars := []any{}
	list, _ := result.Data["avatars"].([]any)
	for index, raw := range list {
		if index == 8 {
			break
		}
		avatar, _ := raw.(map[string]any)
		name := gamekit.Text(avatar["name"])
		switch gamekit.Int(avatar["id"]) {
		case 10000005:
			name = "空"
		case 10000007:
			name = "荧"
		}
		avatars = append(avatars, map[string]any{"element": roleElements[gamekit.Text(avatar["element"])], "constellation": gamekit.Int(avatar["actived_constellation_num"]),
			"level": gamekit.Text(avatar["level"]), "image": resources.Artwork("gacha-"+strconv.Itoa(index), "miao-plugin", "resources/meta-gs/character/"+name+"/imgs/gacha.webp")})
	}
	return gamekit.Image{Template: "role-card", Data: map[string]any{"uid": result.Role.UID, "lines": lines, "avatars": avatars}, Resources: resources.List}, true
}

// RoleExplore draws 探索 the way Yunzai's html/player/role-explore does: the
// requester's avatar (or the in-game head when there is none), nickname,
// level, server and UID; the overview of active days, Spiral Abyss, Imaginarium
// Theater, Stygian Onslaught, characters, friendships, waypoints, domains,
// achievements, chests with their share and grade, Oculi and the Serenitea
// Pot, each against Yunzai's totals; the TCG level and card collection; and
// every area with its exploration, reputation, underground areas and offering
// level.
func RoleExplore(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	stats, _ := result.Data["stats"].(map[string]any)
	if stats == nil {
		return gamekit.Image{}, false
	}
	role, _ := result.Data["role"].(map[string]any)
	item := func(label string, value any, extra any) map[string]any {
		entry := map[string]any{"label": label, "num": gamekit.Text(value), "color": ""}
		if text := gamekit.Text(extra); text != "" && text != "0" {
			entry["extra"] = text
		}
		return entry
	}
	chests, allChests := roleChests(stats), 0
	for key, total := range roleTotals {
		if strings.HasSuffix(key, "_chest") {
			allChests += total
		}
	}
	percentage := math.Round(float64(chests)*100/float64(allChests)*100) / 100
	grade, color := "D", "#12a182"
	for _, step := range []struct {
		below        float64
		grade, color string
	}{{60, "D", "#12a182"}, {70, "C", "#2775b6"}, {80, "B", "#806d9e"}, {90, "A", "#c04851"}, {math.Inf(1), "S", "#f86b1d"}} {
		if percentage < step.below {
			grade, color = step.grade, step.color
			break
		}
	}
	if grade != "D" {
		grade += "[" + strconv.FormatFloat(percentage, 'f', -1, 64) + "%]"
	}
	// Upstream counts days since release, 2020-09-15 at UTC midnight.
	days := int(context.Now.Sub(time.Date(2020, 9, 15, 0, 0, 0, 0, time.UTC)).Hours()/24) + 1
	combat, _ := stats["role_combat"].(map[string]any)
	theater := "未解锁"
	if unlocked, _ := combat["is_unlock"].(bool); unlocked {
		theater = "-"
		if detail, _ := combat["has_detail_data"].(bool); detail {
			theater = "第" + gamekit.Text(combat["max_round_id"]) + "幕"
			if tarot := gamekit.Int(combat["tarot_finished_cnt"]); tarot > 0 {
				theater += " 圣牌" + strconv.Itoa(tarot)
			}
		}
	}
	hard, _ := stats["hard_challenge"].(map[string]any)
	onslaught := "未解锁"
	if unlocked, _ := hard["is_unlock"].(bool); unlocked {
		onslaught = "-"
		if has, _ := hard["has_data"].(bool); has {
			onslaught = ""
			if difficulty := gamekit.Int(hard["difficulty"]); difficulty >= 1 && difficulty <= 6 {
				onslaught = []string{"I", "II", "III", "IV", "V", "VI"}[difficulty-1]
			}
		}
	}
	lines := []any{
		[]any{item("活跃天数", stats["active_day_number"], days), item("深境螺旋", stats["spiral_abyss"], nil), item("幻想真境剧诗", theater, nil), item("幽境危战", onslaught, nil)},
		[]any{item("角色数", stats["avatar_number"], roleTotals["avatar"]), item("满好感角色", stats["full_fetter_avatar_num"], gamekit.Int(stats["avatar_number"])-3),
			item("传送点", stats["way_point_number"], roleTotals["way_point"]), item("秘境", stats["domain_number"], roleTotals["domain"]),
			item("成就", stats["achievement_number"], roleTotals["achievement"])},
		[]any{item("宝箱总数", chests, allChests), map[string]any{"label": "宝箱获取率", "num": grade, "color": color},
			item("普通宝箱", stats["common_chest_number"], roleTotals["common_chest"]), item("精致宝箱", stats["exquisite_chest_number"], roleTotals["exquisite_chest"]),
			item("珍贵宝箱", stats["precious_chest_number"], roleTotals["precious_chest"])},
		[]any{item("华丽宝箱", stats["luxurious_chest_number"], roleTotals["luxurious_chest"]), item("奇馈宝箱", stats["magic_chest_number"], roleTotals["magic_chest"]),
			item("风神瞳", stats["anemoculus_number"], roleTotals["anemoculus"]), item("岩神瞳", stats["geoculus_number"], roleTotals["geoculus"]),
			item("雷神瞳", stats["electroculus_number"], roleTotals["electroculus"])},
		[]any{item("草神瞳", stats["dendroculus_number"], roleTotals["dendroculus"]), item("水神瞳", stats["hydroculus_number"], roleTotals["hydroculus"]),
			item("火神瞳", stats["pyroculus_number"], roleTotals["pyroculus"]), item("月神瞳", stats["moonoculus_number"], roleTotals["moonoculus"]),
			item("冰神瞳", stats["iceculus_number"], nil)},
	}
	if homes, _ := result.Data["homes"].([]any); len(homes) > 0 {
		home, _ := homes[0].(map[string]any)
		lines = append(lines, []any{item("家园等级", home["level"], nil), item("最高仙力", home["comfort_num"], nil), item("洞天名称", home["comfort_level_name"], nil),
			item("获得摆设", home["item_num"], nil), item("历史访客", home["visit_num"], nil)})
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, artwork := range [][2]string{{"tttgbnumber", "resources/font/tttgbnumber.ttf"}, {"genshin-logo", "resources/img/other/原神.png"},
		{"img-deck-tcg", "resources/img/deck/七圣召唤.png"}, {"img-other-world-exploration-frame", "resources/img/other/world-exploration-frame.png"}} {
		resources.Artwork(artwork[0], "yunzai-genshin", artwork[1])
	}
	// genshinLayout's header shows 闲云's banner from miao.
	resources.Artwork("head-banner", "miao-plugin", "resources/meta-gs/character/闲云/imgs/banner.webp")
	areas := roleExplorations(result.Data)
	explorations := []any{}
	for index, area := range areas {
		id := gamekit.Int(area["id"])
		if slices.Contains([]int{7, 11, 12, 13}, id) {
			continue
		}
		name := roleAreaName(area)
		lines := []any{}
		if id != 10 {
			lines = append(lines, map[string]any{"name": name, "text": rolePercent(area)})
		}
		if slices.Contains([]string{"蒙德", "璃月", "稻妻", "须弥", "枫丹"}, name) {
			lines = append(lines, map[string]any{"name": "声望", "text": gamekit.Text(area["level"]) + "级"})
		}
		underground := map[int][]int{6: {7}, 10: {13, 12, 11}}[id]
		for _, other := range underground {
			for _, candidate := range areas {
				if gamekit.Int(candidate["id"]) == other {
					lines = append(lines, map[string]any{"name": roleAreas[other], "text": rolePercent(candidate)})
				}
			}
		}
		if slices.Contains([]string{"雪山", "稻妻", "层岩巨渊", "须弥", "枫丹", "沉玉谷", "纳塔", "空之神殿"}, name) {
			offerings, _ := area["offerings"].([]any)
			if len(offerings) > 0 {
				offering, _ := offerings[0].(map[string]any)
				label := gamekit.Text(offering["name"])
				for _, short := range []string{"流明石", "摹忆中枢"} {
					if strings.Contains(label, short) {
						label = short
					}
				}
				lines = append(lines, map[string]any{"name": label, "text": gamekit.Text(offering["level"]) + "级"})
			}
		}
		explorations = append(explorations, map[string]any{"icon": resources.Artwork(fmt.Sprintf("area-%d", index), "yunzai-genshin", "resources/img/other/"+name+".png"), "lines": lines})
	}
	data := map[string]any{"uid": result.Role.UID, "nickname": gamekit.Text(role["nickname"]), "level": strconv.Itoa(gamekit.Int(role["level"])),
		"region": gamekit.Text(role["region"]), "head": resources.URL("mihoyo", role["game_head_icon"]), "lines": lines, "explorations": explorations}
	// Upstream reads the TCG summary alongside and leaves it out when it fails.
	if context.Query != nil {
		if tcg, err := context.Query("genshin.tcg", nil); err == nil && gamekit.Int(tcg.Data["level"]) > 0 {
			data["tcg"] = map[string]any{"level": gamekit.Text(tcg.Data["level"]),
				"characters": gamekit.Text(tcg.Data["avatar_card_num_gained"]) + "/" + gamekit.Text(tcg.Data["avatar_card_num_total"]),
				"actions":    gamekit.Text(tcg.Data["action_card_num_gained"]) + "/" + gamekit.Text(tcg.Data["action_card_num_total"])}
		}
	}
	return gamekit.Image{Template: "role-explore", Data: data, Resources: resources.List}, true
}
