package images

import (
	"regexp"
	"strings"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

var (
	hardDifficulties = map[int]string{1: "普通", 2: "进阶", 3: "困难", 4: "险恶", 5: "无畏", 6: "绝境"}
	// hardCardTypes is how miao sizes the avatar cards of a team by its size.
	hardCardTypes = map[int][]string{1: {"wide wide2"}, 2: {"wide", "wide"}, 3: {"wide", "mini", "mini"}, 4: {"mini", "mini", "mini", "mini"}}
	hardColor     = regexp.MustCompile(`<color=([^>]+)>(.*?)</color>`)
)

// HardChallenge draws Stygian Onslaught the way miao's stat/hard-summary
// does: the period, best difficulty and total time, then each battle with its
// time, team as avatar cards (marked when the period's popularity buff favours
// them), the monster, the strongest hit and highest total damage, and the
// monster's traits. The command word picks the period and single or
// cooperative mode; without a mode the better record wins, as upstream.
func HardChallenge(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	periods, _ := result.Data["data"].([]any)
	index := 0
	if strings.Contains(context.Word, "上期") {
		index = 1
	}
	if index >= len(periods) {
		return gamekit.Image{}, false
	}
	period, _ := periods[index].(map[string]any)
	single, _ := period["single"].(map[string]any)
	mp, _ := period["mp"].(map[string]any)
	score := func(mode map[string]any) int {
		if has, _ := mode["has_data"].(bool); !has {
			return 0
		}
		best, _ := mode["best"].(map[string]any)
		return gamekit.Int(best["difficulty"])*1000 - gamekit.Int(best["second"])
	}
	var mode map[string]any
	switch word := context.Word; {
	case strings.Contains(word, "单人") || strings.Contains(word, "单挑"):
		mode = single
	case strings.Contains(word, "组队") || strings.Contains(word, "多人") || strings.Contains(word, "合作"):
		mode = mp
	case score(single) >= score(mp):
		mode = single
	default:
		mode = mp
	}
	// Upstream answers in text while the chosen record is empty.
	if has, _ := mode["has_data"].(bool); !has {
		return gamekit.Image{}, false
	}

	resources := &gamekit.ImageResources{Context: context}
	for _, item := range hardArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	challenges, _ := mode["challenge"].([]any)
	ids := []string{}
	for _, raw := range challenges {
		challenge, _ := raw.(map[string]any)
		teams, _ := challenge["teams"].([]any)
		for _, rawAvatar := range teams {
			avatar, _ := rawAvatar.(map[string]any)
			ids = append(ids, gamekit.Text(avatar["avatar_id"]))
		}
	}
	cards := newAvatarCards(context, resources, ids)
	popular := map[string]bool{}
	if context.Query != nil {
		if popularity, err := context.Query("genshin.hard_challenge_popularity", nil); err == nil {
			list, _ := popularity.Data["avatar_list"].([]any)
			for _, raw := range list {
				avatar, _ := raw.(map[string]any)
				popular[gamekit.Text(avatar["avatar_id"])] = true
			}
		}
	}
	// miao guesses a cooperative teammate's character when the requester has
	// none at that level or constellation.
	card := func(avatar map[string]any) map[string]any {
		id, level, cons := gamekit.Text(avatar["avatar_id"]), gamekit.Int(avatar["level"]), gamekit.Int(avatar["rank"])
		result, ok := cards.own(id)
		if !ok || gamekit.Int(result["cons"]) < cons || gamekit.Int(result["level"]) < level {
			result = cards.guest(id, level, cons)
		}
		result["popular"] = popular[id]
		return result
	}

	battles := []any{}
	for _, raw := range challenges {
		challenge, _ := raw.(map[string]any)
		teams, _ := challenge["teams"].([]any)
		team := []any{}
		faces := map[string]string{}
		for position, rawAvatar := range teams {
			avatar, _ := rawAvatar.(map[string]any)
			entry := card(avatar)
			if types := hardCardTypes[len(teams)]; position < len(types) {
				entry["type"] = types[position]
			}
			faces[gamekit.Text(avatar["avatar_id"])] = gamekit.Text(entry["face"])
			team = append(team, entry)
		}
		best := []any{}
		list, _ := challenge["best_avatar"].([]any)
		for _, rawBest := range list {
			avatar, _ := rawBest.(map[string]any)
			id := gamekit.Text(avatar["avatar_id"])
			face, seen := faces[id]
			if !seen {
				face = gamekit.Text(cards.base(id)["face"])
			}
			best = append(best, map[string]any{"face": face, "dps": gamekit.Text(avatar["dps"])})
		}
		monster, _ := challenge["monster"].(map[string]any)
		traits := []any{}
		descs, _ := monster["desc"].([]any)
		for _, rawDesc := range descs {
			if desc := gamekit.Text(rawDesc); desc != "" {
				traits = append(traits, hardTrait(desc))
			}
		}
		battles = append(battles, map[string]any{"name": gamekit.Text(challenge["name"]), "second": gamekit.Text(challenge["second"]), "team": team,
			"monster": map[string]any{"level": gamekit.Text(monster["level"]), "icon": resources.URL("mihoyo", monster["icon"]), "traits": traits}, "best": best})
	}
	stat, _ := mode["best"].(map[string]any)
	schedule, _ := period["schedule"].(map[string]any)
	moment := func(value any) string {
		return time.Unix(int64(gamekit.Int(value)), 0).In(chinaTime).Format("01-02 15:04:05")
	}
	difficulty := gamekit.Int(stat["difficulty"])
	return gamekit.Image{Template: "hard-challenge", Data: map[string]any{
		"uid": result.Role.UID, "begin": moment(schedule["start_time"]), "end": moment(schedule["end_time"]),
		"difficulty": difficulty, "difficulty_name": hardDifficulties[difficulty], "second": gamekit.Text(stat["second"]), "battles": battles,
	}, Resources: resources.List}, true
}

// hardTrait splits a monster trait into plain and coloured pieces; upstream
// turns the official <color> tags into coloured spans.
func hardTrait(text string) []any {
	pieces := []any{}
	last := 0
	for _, match := range hardColor.FindAllStringSubmatchIndex(text, -1) {
		if match[0] > last {
			pieces = append(pieces, map[string]any{"text": text[last:match[0]]})
		}
		pieces = append(pieces, map[string]any{"text": text[match[4]:match[5]], "color": text[match[2]:match[3]]})
		last = match[1]
	}
	if last < len(text) {
		pieces = append(pieces, map[string]any{"text": text[last:]})
	}
	return pieces
}

// hardArtwork maps the images named in the converted stylesheets (miao's
// common, stat/common, common/tpl and stat/hard-summary) to their paths, and
// the fonts to miao's family names.
var hardArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-cont-logo", "resources/common/cont/logo.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg1-o", "resources/common/item/bg1-o.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg2-o", "resources/common/item/bg2-o.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg3-o", "resources/common/item/bg3-o.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg4-o", "resources/common/item/bg4-o.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"common-item-bg5-o", "resources/common/item/bg5-o.png"},
	{"common-item-fetter", "resources/common/item/fetter.png"},
	{"stat-imgs-bg1", "resources/stat/imgs/bg1.png"},
	{"stat-imgs-footer", "resources/stat/imgs/footer.png"},
}
