package images

import (
	"math"
	"sort"
	"strconv"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

var (
	// rosterCities are the regions miao's avatar-list shows, in its sprite
	// order.
	rosterCities = []string{"蒙德", "龙脊雪山", "璃月", "层岩巨渊", "稻妻", "渊下宫", "须弥", "枫丹"}
	// rosterChests are miao's chest kinds with the maxima of the pinned
	// snapshot's meta-gs/info chestInfo.
	rosterChests = []struct {
		key, title string
		max        int
	}{{"common", "普通宝箱", 3838}, {"exquisite", "精致宝箱", 3372}, {"precious", "珍贵宝箱", 1073}, {"luxurious", "华丽宝箱", 410}, {"magic", "奇馈宝箱", 416}}
	rosterStats = []struct{ key, title string }{{"achievement", "成就"}, {"wayPoint", "锚点"}, {"avatar", "角色"}, {"avatar5", "五星角色"}, {"goldCount", "金卡总数"}}
)

// rosterArtwork maps the images named in the converted stylesheets (miao's
// common, character/avatar-list and common/tpl) to their paths.
var rosterArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-chest", "resources/character/imgs/chest.webp"},
	{"character-imgs-exploration", "resources/character/imgs/exploration.webp"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"common-item-fetter", "resources/common/item/fetter.png"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"common-item-bg1-o", "resources/common/item/bg1-o.png"},
	{"common-item-bg2-o", "resources/common/item/bg2-o.png"},
	{"common-item-bg3-o", "resources/common/item/bg3-o.png"},
	{"common-item-bg4-o", "resources/common/item/bg4-o.png"},
	{"common-item-bg5-o", "resources/common/item/bg5-o.png"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
}

// Characters draws the character list the way miao's character/avatar-list
// does: the profile banner with the strongest character, nickname, level,
// days active and counts, each region's exploration, the chests, and every
// character as an avatar card, strongest first.
func Characters(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	list, _ := result.Data["list"].([]any)
	if len(list) == 0 {
		return gamekit.Image{}, false
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range rosterArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	entries, cards := buildRoster(context, resources, list)
	five, gold := 0, 0
	avatars := []any{}
	for _, item := range entries {
		avatars = append(avatars, item.card)
		// miao counts a five-star's constellations, travelers aside, and each
		// five-star weapon's refinements as gold cards.
		if item.weaponStar == 5 {
			gold += item.refinement
		}
		if item.star == 5 {
			five++
			if !traveler(item.id) {
				gold += item.cons + 1
			}
		}
	}

	player := map[string]any{"name": "#" + result.Role.UID, "level": result.Role.Level}
	stats := map[string]int{"avatar": len(list), "avatar5": five, "goldCount": gold}
	var exploration, chests []any
	if context.Query != nil {
		if index, err := context.Query("genshin.profile", nil); err == nil {
			role, _ := index.Data["role"].(map[string]any)
			if nickname := gamekit.Text(role["nickname"]); nickname != "" {
				player["name"], player["level"] = nickname, gamekit.Int(role["level"])
			}
			raw, _ := index.Data["stats"].(map[string]any)
			stats["achievement"], stats["wayPoint"] = gamekit.Int(raw["achievement_number"]), gamekit.Int(raw["way_point_number"])
			stats["avatar"] = max(stats["avatar"], gamekit.Int(raw["avatar_number"]))
			if days := gamekit.Int(raw["active_day_number"]); days > 0 {
				player["active"] = activeDays(days)
			}
			regions := map[string]int{}
			worlds, _ := index.Data["world_explorations"].([]any)
			for _, rawWorld := range worlds {
				world, _ := rawWorld.(map[string]any)
				switch name := gamekit.Text(world["name"]); name {
				case "层岩巨渊":
				case "层岩巨渊·地下矿区":
					regions["层岩巨渊"] = gamekit.Int(world["exploration_percentage"])
				default:
					regions[name] = gamekit.Int(world["exploration_percentage"])
				}
			}
			if _, ok := regions["蒙德"]; ok {
				for index, city := range rosterCities {
					exploration = append(exploration, map[string]any{"name": city, "position": strconv.Itoa(index) + "0% 0", "value": strconv.FormatFloat(float64(regions[city])/10, 'f', -1, 64) + "%"})
				}
			}
			if gamekit.Int(raw["common_chest_number"]) > 0 {
				for index, chest := range rosterChests {
					value := gamekit.Int(raw[chest.key+"_chest_number"])
					chests = append(chests, map[string]any{"title": chest.title, "value": value, "max": max(chest.max, value), "position": strconv.Itoa(index*2) + "0% 0"})
				}
			}
		}
	}
	player["show_level"] = gamekit.Int(player["level"]) > 1
	shown := []any{}
	for _, stat := range rosterStats {
		if value := stats[stat.key]; value != 0 {
			shown = append(shown, map[string]any{"title": stat.title, "value": value})
		}
	}
	// The banner shows the strongest character, as upstream without a
	// profile picture.
	path := "resources/meta-gs/character/" + cards.records[entries[0].id].Name + "/imgs/"
	player["banner"], player["face"] = cards.miao(path+"banner.webp"), cards.miao(path+"face-q.webp")
	if player["face"] == "" {
		player["face"] = cards.miao(path + "face.webp")
	}
	return gamekit.Image{Template: "characters", Data: map[string]any{
		"uid": result.Role.UID, "player": player, "stats": shown, "exploration": exploration, "chests": chests,
		"avatars": avatars, "updated": context.Now.In(chinaTime).Format("2006-01-02 15:04"),
	}, Resources: resources.List}, true
}

// activeDays writes days active the way miao does: years, months and days.
func activeDays(days int) string {
	text := ""
	rest := float64(days % 365)
	year, month, day := days/365, int(rest/30.41), int(math.Mod(rest, 30.41))
	if year > 0 {
		text += strconv.Itoa(year) + "年"
	}
	if month > 0 {
		text += strconv.Itoa(month) + "个月"
	}
	if day > 0 {
		text += strconv.Itoa(day) + "天"
	}
	return text
}

// rosterEntry is one owned character with what miao sorts rosters by.
type rosterEntry struct {
	id                                  string
	card                                map[string]any
	panel                               *gamekit.CharacterPanel
	level, star, aeq, cons              int
	weaponLevel, weaponStar, refinement int
	fetter                              int
}

// buildRoster reads the requester's characters from the official list with
// their details, as avatar cards in miao's order: level, rarity, original
// talents, constellation, weapon level, rarity and refinement, then
// friendship, highest first.
func buildRoster(context gamekit.ImageContext, resources *gamekit.ImageResources, list []any) ([]rosterEntry, *avatarCards) {
	ids := []string{}
	for _, raw := range list {
		avatar, _ := raw.(map[string]any)
		ids = append(ids, gamekit.Text(avatar["id"]))
	}
	cards := newAvatarCards(context, resources, ids)
	entries := []rosterEntry{}
	for _, raw := range list {
		avatar, _ := raw.(map[string]any)
		id := gamekit.Text(avatar["id"])
		card, ok := cards.own(id)
		if !ok {
			card = cards.guest(id, gamekit.Int(avatar["level"]), gamekit.Int(avatar["actived_constellation_num"]))
		}
		card["type"] = "mini"
		item := rosterEntry{id: id, card: card, level: gamekit.Int(card["level"]), star: gamekit.Int(card["star"]), cons: gamekit.Int(card["cons"]), fetter: gamekit.Int(avatar["fetter"])}
		if panel, ok := cards.panels[id]; ok {
			item.panel = &panel
		}
		// Without talents miao counts three.
		item.aeq = 3
		if talents, _ := card["talents"].([]any); talents != nil {
			item.aeq = 0
			for _, talent := range talents {
				item.aeq += gamekit.Int(talent.(map[string]any)["original"])
			}
		}
		if weapon, _ := card["weapon"].(map[string]any); weapon != nil {
			item.weaponLevel, item.weaponStar, item.refinement = gamekit.Int(weapon["level"]), gamekit.Int(weapon["star"]), gamekit.Int(weapon["affix"])
		}
		entries = append(entries, item)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		for _, pair := range [][2]int{{a.level, b.level}, {a.star, b.star}, {a.aeq, b.aeq}, {a.cons, b.cons}, {a.weaponLevel, b.weaponLevel}, {a.weaponStar, b.weaponStar}, {a.refinement, b.refinement}, {a.fetter, b.fetter}} {
			if pair[0] != pair[1] {
				return pair[0] > pair[1]
			}
		}
		return false
	})
	return entries, cards
}

// traveler reports the two travelers, whom miao leaves out of gold cards
// and shows at full friendship.
func traveler(id string) bool { return id == "10000005" || id == "10000007" }
