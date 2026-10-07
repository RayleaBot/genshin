package images

import (
	"slices"
	"strings"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
)

var (
	roleDifficulties = map[int]string{1: "简单", 2: "普通", 3: "困难", 4: "卓越", 5: "月谕"}
	// roleAvatarTypes are miao's labels for the cast the player does not
	// field: trial characters and other players' support.
	roleAvatarTypes = map[int]string{2: "试用", 3: "助演"}
)

// RoleSummary draws Imaginarium Theater, this period's or with 上期 the last
// one, the way miao's stat/role-summary does: the month and difficulty, the
// star medal of every act, total time, flowers spent, audience support and
// support lent, then each act in order with its medal, finish time, cast as
// avatar cards, the enemy chosen, the Brilliance level with its bonuses and
// blessings, and the mysterious gains picked.
func RoleSummary(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	periods, _ := result.Data["data"].([]any)
	index := 0
	if strings.Contains(context.Word, "上期") {
		index = 1
	}
	// Upstream answers in text while the period has no detailed record.
	if index >= len(periods) {
		return app.Image{}, false
	}
	period, _ := periods[index].(map[string]any)
	if detailed, _ := period["has_detail_data"].(bool); !detailed {
		return app.Image{}, false
	}
	detail, _ := period["detail"].(map[string]any)
	stat, _ := period["stat"].(map[string]any)
	schedule, _ := period["schedule"].(map[string]any)
	rounds, _ := detail["rounds_data"].([]any)

	// The player's own cast shows their current characters; miao lays the
	// trial and support cast's level and label over the same card.
	type cast struct {
		level int
		label string
	}
	own := []string{}
	others := map[string]cast{}
	urls := []any{}
	for _, rawRound := range rounds {
		round, _ := rawRound.(map[string]any)
		avatars, _ := round["avatars"].([]any)
		for _, rawAvatar := range avatars {
			avatar, _ := rawAvatar.(map[string]any)
			id := app.Text(avatar["avatar_id"])
			if kind := app.Int(avatar["avatar_type"]); kind == 1 {
				own = append(own, id)
			} else {
				others[id] = cast{app.Int(avatar["level"]), roleAvatarTypes[kind]}
			}
		}
		enemies, _ := round["enemies"].([]any)
		if len(enemies) > 0 {
			enemy, _ := enemies[0].(map[string]any)
			urls = append(urls, enemy["icon"])
		}
		splendour, _ := round["splendour_buff"].(map[string]any)
		buffs, _ := splendour["buffs"].([]any)
		choices, _ := round["choice_cards"].([]any)
		for _, list := range [][]any{buffs, choices} {
			for _, raw := range list {
				item, _ := raw.(map[string]any)
				urls = append(urls, item["icon"])
			}
		}
	}

	resources := &app.ImageResources{Context: context}
	for _, item := range statArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	medal := resources.Artwork("stat-imgs-medal", "miao-plugin", "resources/stat/imgs/medal.png")
	noMedal := resources.Artwork("stat-imgs-nomedal", "miao-plugin", "resources/stat/imgs/nomedal.png")
	medalOf := func(won bool) string {
		if won {
			return medal
		}
		return noMedal
	}
	resources.Prefetch("mihoyo", urls...)
	slices.Sort(own)
	cards := newAvatarCards(context, resources, slices.Compact(own))
	card := func(id string) map[string]any {
		entry, owned := cards.own(id)
		other, lent := others[id]
		switch {
		case lent && owned:
			entry["level"], entry["cons"] = other.level, other.label
		case lent:
			entry = cards.guest(id, other.level, 0)
			entry["cons"] = other.label
		case !owned:
			// Characters the query does not return get miao's empty card.
			entry = map[string]any{}
		}
		return entry
	}

	acts := []any{}
	for _, rawRound := range rounds {
		round, _ := rawRound.(map[string]any)
		title := "第 " + app.Text(round["round_id"]) + " 幕"
		if tarot, _ := round["is_tarot"].(bool); tarot {
			title = "圣牌挑战 " + roman(app.Int(round["tarot_serial_no"]))
		}
		avatars, _ := round["avatars"].([]any)
		team := []any{}
		for position, rawAvatar := range avatars {
			avatar, _ := rawAvatar.(map[string]any)
			member := card(app.Text(avatar["avatar_id"]))
			if types := statCardTypes[len(avatars)]; position < len(types) {
				member["type"] = types[position]
			}
			team = append(team, member)
		}
		enemy := ""
		if enemies, _ := round["enemies"].([]any); len(enemies) > 0 {
			first, _ := enemies[0].(map[string]any)
			enemy = resources.URL("mihoyo", first["icon"])
		}
		splendour, _ := round["splendour_buff"].(map[string]any)
		summary, _ := splendour["summary"].(map[string]any)
		level := app.Int(summary["total_level"])
		blessings := []any{}
		buffs, _ := splendour["buffs"].([]any)
		for _, raw := range buffs {
			buff, _ := raw.(map[string]any)
			blessings = append(blessings, map[string]any{"icon": resources.URL("mihoyo", buff["icon"]), "level": app.Text(buff["level"])})
		}
		gains := []any{}
		choices, _ := round["choice_cards"].([]any)
		for _, raw := range choices {
			choice, _ := raw.(map[string]any)
			gains = append(gains, resources.URL("mihoyo", choice["icon"]))
		}
		won, _ := round["is_get_medal"].(bool)
		acts = append(acts, map[string]any{"title": title, "medal": medalOf(won), "time": time.Unix(int64(app.Int(round["finish_time"])), 0).In(chinaTime).Format("01-02 15:04:05"),
			"team": team, "enemy": enemy, "level": level, "hp": level * 800, "atk": level * 50, "def": level * 50, "em": level * 20, "blessings": blessings, "gains": gains})
	}
	medals := []any{}
	list, _ := stat["get_medal_round_list"].([]any)
	for position, won := range list {
		// miao starts a second row of medals at the eleventh act.
		medals = append(medals, map[string]any{"medal": medalOf(app.Int(won) == 1), "line": position == 10})
	}
	fight, _ := detail["fight_statisic"].(map[string]any)
	start, _ := schedule["start_date_time"].(map[string]any)
	return app.Image{Template: "stat-role-summary", Data: map[string]any{
		"uid": result.Role.UID, "month": app.Text(start["month"]), "difficulty": roleDifficulties[app.Int(stat["difficulty_id"])], "medals": medals,
		"time": app.Text(fight["total_use_time"]), "coins": app.Text(stat["coin_num"]), "bonus": app.Text(stat["avatar_bonus_num"]), "rent": app.Text(stat["rent_cnt"]),
		"acts": acts,
	}, Resources: resources.List}, true
}

// roman writes a tarot challenge's number the way miao's intToRoman does.
func roman(number int) string {
	if number < 1 || number > 3999 {
		return ""
	}
	digits := [4][]string{
		{"", "M", "MM", "MMM"},
		{"", "C", "CC", "CCC", "CD", "D", "DC", "DCC", "DCCC", "CM"},
		{"", "X", "XX", "XXX", "XL", "L", "LX", "LXX", "LXXX", "XC"},
		{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX"},
	}
	return digits[0][number/1000] + digits[1][number%1000/100] + digits[2][number%100/10] + digits[3][number%10]
}
