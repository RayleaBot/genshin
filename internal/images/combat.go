package images

import (
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
)

var (
	combatDifficulties = map[int]string{1: "轻简", 2: "普通", 3: "困难", 4: "卓越", 5: "月谕"}
	combatAvatarTypes  = map[int]string{2: "试用", 3: "助演"}
)

// combatArtwork maps the images named in the theater stylesheet, converted
// from Yunzai's html/abyss/combat stylesheet, to their paths.
var combatArtwork = [][2]string{
	{"img-combat-flower", "resources/img/combat/flower.png"},
	{"img-combat-heraldry-0", "resources/img/combat/heraldry-0.png"},
	{"img-combat-heraldry-3", "resources/img/combat/heraldry-3.png"},
	{"img-combat-heraldry-4", "resources/img/combat/heraldry-4.png"},
	{"img-combat-medal-active", "resources/img/combat/medal-active.png"},
	{"img-combat-medal-default", "resources/img/combat/medal-default.png"},
	{"img-combat-page-bottom", "resources/img/combat/page-bottom.png"},
	{"img-combat-page-top", "resources/img/combat/page-top.png"},
	{"img-combat-panel", "resources/img/combat/panel.png"},
	{"img-combat-round-bottom", "resources/img/combat/round-bottom.png"},
	{"img-combat-round-top", "resources/img/combat/round-top.png"},
	{"img-combat-rounds", "resources/img/combat/rounds.png"},
	{"img-other-bg3", "resources/img/other/bg3.png"},
	{"img-other-bg4", "resources/img/other/bg4.png"},
	{"img-other-bg5", "resources/img/other/bg5.png"},
}

// officialImage caches an image from an official URL on demand, the way
// upstream's page loads it, and returns its resource ID.
func officialImage(context gamekit.ImageContext, add func(rayleabot.RenderImageResource), id, url string) string {
	name, found := strings.CutPrefix(url, "https://")
	if !found {
		return ""
	}
	name, _, _ = strings.Cut(name, "?")
	resource, ok := context.FetchArtworkResource(id, "mihoyo", name)
	if !ok {
		return ""
	}
	add(resource)
	return id
}

// Combat draws Imaginarium Theater the way Yunzai's html/abyss/combat does:
// the traveler card, the difficulty and best act with star medals, flowers,
// audience support and support lends, then every act with its cast, buffs
// and chosen cards.
func Combat(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	list, _ := result.Data["data"].([]any)
	if len(list) == 0 {
		return gamekit.Image{}, false
	}
	first, _ := list[0].(map[string]any)
	detail, _ := first["detail"].(map[string]any)
	stat, _ := first["stat"].(map[string]any)
	if gamekit.Text(result.Data["has_detail_data"]) == "false" || detail == nil || stat == nil {
		return gamekit.Image{}, false
	}
	resources := []rayleabot.RenderImageResource{}
	add := func(resource rayleabot.RenderImageResource) { resources = append(resources, resource) }
	local := func(id, source, name string) string {
		resource, ok := context.ArtworkResource(id, source, name)
		if !ok {
			return ""
		}
		add(resource)
		return id
	}
	local("tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf")
	for _, item := range combatArtwork {
		local(item[0], "yunzai-genshin", item[1])
	}

	player := map[string]any{"uid": result.Role.UID, "nickname": result.Role.Nickname, "level": result.Role.Level}
	if context.Query != nil {
		if index, err := context.Query("genshin.profile", nil); err == nil {
			role, _ := index.Data["role"].(map[string]any)
			player["nickname"], player["level"] = gamekit.Text(role["nickname"]), gamekit.Text(role["level"])
			if head := gamekit.Text(role["game_head_icon"]); head != "" {
				player["head"] = officialImage(context, add, "head", head)
			}
		}
	}

	medals := []any{}
	list, _ = stat["get_medal_round_list"].([]any)
	for _, medal := range list {
		medals = append(medals, gamekit.Int(medal))
	}
	faces, elements := map[string]string{}, map[string]string{}
	icons := 0
	rounds := []any{}
	list, _ = detail["rounds_data"].([]any)
	for _, raw := range list {
		round, _ := raw.(map[string]any)
		cast := []any{}
		avatars, _ := round["avatars"].([]any)
		for _, rawAvatar := range avatars {
			avatar, _ := rawAvatar.(map[string]any)
			name, element := gamekit.Text(avatar["name"]), gamekit.Text(avatar["element"])
			if _, seen := faces[name]; !seen {
				faces[name] = local("face-"+strconv.Itoa(len(faces)), "miao-plugin", "resources/meta-gs/character/"+name+"/imgs/face.webp")
			}
			if _, seen := elements[element]; !seen {
				// The files are lower case; upstream's mixed-case path only
				// resolves on case-insensitive disks.
				elements[element] = local("element-"+strconv.Itoa(len(elements)), "yunzai-genshin", "resources/img/element/"+strings.ToLower(element)+".png")
			}
			kind := gamekit.Int(avatar["avatar_type"])
			cast = append(cast, map[string]any{"rarity": gamekit.Int(avatar["rarity"]), "face": faces[name], "element": elements[element],
				"type": kind, "type_name": combatAvatarTypes[kind], "level": gamekit.Text(avatar["level"])})
		}
		iconsOf := func(field string) []any {
			result := []any{}
			entries, _ := round[field].([]any)
			for _, rawEntry := range entries {
				entry, _ := rawEntry.(map[string]any)
				icons++
				if id := officialImage(context, add, "icon-"+strconv.Itoa(icons), gamekit.Text(entry["icon"])); id != "" {
					result = append(result, id)
				}
			}
			return result
		}
		tarot, _ := round["is_tarot"].(bool)
		medal, _ := round["is_get_medal"].(bool)
		rounds = append(rounds, map[string]any{"tarot": tarot, "tarot_no": gamekit.Text(round["tarot_serial_no"]), "round": gamekit.Text(round["round_id"]),
			"medal": medal, "finish": gamekit.Text(round["finish"]), "cast": cast, "buffs": iconsOf("buffs"), "cards": iconsOf("choice_cards")})
	}
	return gamekit.Image{Template: "combat", Data: map[string]any{
		"player": player, "difficulty": combatDifficulties[gamekit.Int(stat["difficulty_id"])],
		"heraldry": gamekit.Text(stat["heraldry"]), "start_time": gamekit.Text(stat["start_time"]),
		"max_round": gamekit.Text(stat["max_round_id"]), "medals": medals, "coins": gamekit.Text(stat["coin_num"]),
		"bonus": gamekit.Text(stat["avatar_bonus_num"]), "rent": gamekit.Text(stat["rent_cnt"]), "rounds": rounds,
	}, Resources: resources}, true
}
