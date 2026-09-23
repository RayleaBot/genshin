package app

import (
	"context"
	"slices"
	"sort"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// AbyssTeam is one half's team of miao's 深渊配队: its characters in ID order,
// how often the sample used it, and whether the player owns all of them.
type AbyssTeam struct {
	IDs   []string `json:"ids"`
	Count float64  `json:"count"`
	Mark  float64  `json:"mark"`
	Owned bool     `json:"owned"`
}

// AbyssTeamPair is a first half and a second half team without a shared
// character.
type AbyssTeamPair struct {
	Up    AbyssTeam `json:"up"`
	Down  AbyssTeam `json:"down"`
	Count float64   `json:"count"`
	Mark  float64   `json:"mark"`
}

// abyssTeamPairs is miao's AbyssTeam over yshelper's floor 12 teams: each team
// scores its sample uses times the sum of the player's character weights, or
// just its uses when a character is missing; teams pair across halves best
// first, each team once, and the four best pairs remain. It also returns the
// characters the teams need that the player lacks.
func abyssTeamPairs(data map[string]any, catalog Catalog, weights map[string]float64) ([]AbyssTeamPair, map[string]bool) {
	characters := map[string]string{}
	for _, raw := range asList(data["has_list"]) {
		item := asObject(raw)
		if entry, ok := catalog.Resolve(asText(item["name"]), "character", nil); ok {
			characters[asText(item["avatar"])] = entry.ID
		}
	}
	// The team list is the first result list whose entries carry roles.
	var list []any
	for _, raw := range asList(data["result"]) {
		items := asList(raw)
		if len(items) > 0 && asList(asObject(items[0])["role"]) != nil {
			list = items
			break
		}
	}
	type sample struct {
		key  string
		rate float64
	}
	samples := map[string][]sample{}
	for _, raw := range list {
		item := asObject(raw)
		ids := []string{}
		for _, role := range asList(item["role"]) {
			if id := characters[asText(asObject(role)["avatar"])]; id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		key := strings.Join(ids, ",")
		for _, half := range []string{"up", "down"} {
			if rate := statNumberOf(item[half+"_use_num"]); rate > 0 {
				samples[half] = append(samples[half], sample{key, rate})
			}
		}
	}
	missing := map[string]bool{}
	type team struct {
		AbyssTeam
		key, half string
		left      int
	}
	teams := []*team{}
	for _, half := range []string{"up", "down"} {
		found := map[string]*team{}
		for _, s := range samples[half] {
			ids := strings.Split(s.key, ",")
			sort.Strings(ids)
			mark := 0.0
			for _, id := range ids {
				if weights[id] == 0 {
					missing[id] = true
					mark = -1
				}
				if mark != -1 {
					mark += weights[id]
				}
			}
			if mark == -1 {
				mark = 1
			}
			key := strings.Join(ids, ",")
			t := found[key]
			if t == nil {
				t = &team{AbyssTeam: AbyssTeam{IDs: ids, Owned: mark > 1}, key: key, half: half, left: 1}
				found[key] = t
				teams = append(teams, t)
			}
			t.Count += s.rate
			t.Mark += s.rate * mark
		}
		// miao sorts ascending and reverses after each half.
		sort.SliceStable(teams, func(i, j int) bool { return teams[i].Mark < teams[j].Mark })
		slices.Reverse(teams)
	}
	pairs := []AbyssTeamPair{}
	seen := map[string]bool{}
	for _, t1 := range teams {
		if t1.left <= 0 {
			continue
		}
		for _, t2 := range teams {
			if t1.half == t2.half || t2.left <= 0 {
				continue
			}
			up, down := t1, t2
			if t1.half != "up" {
				up, down = t2, t1
			}
			if seen[up.key+"+"+down.key] || slices.ContainsFunc(t1.IDs, func(id string) bool { return slices.Contains(t2.IDs, id) }) {
				continue
			}
			seen[up.key+"+"+down.key] = true
			pair := AbyssTeamPair{Up: up.AbyssTeam, Down: down.AbyssTeam, Count: min(t1.Count, t2.Count), Mark: t1.Count + t2.Count}
			if t1.Owned && t2.Owned {
				pair.Mark = t1.Mark + t2.Mark
			}
			pairs = append(pairs, pair)
			t1.left--
			t2.left--
			break
		}
		if len(pairs) >= 20 {
			break
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].Mark < pairs[j].Mark })
	slices.Reverse(pairs)
	return pairs[:min(4, len(pairs))], missing
}

// statisticsTeams matches the abyss statistics the page read to the chosen
// account's characters, the same pairing 深渊配队 answers in chat.
func (a *App) statisticsTeams(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	job, err := a.ContentJobs.Poll(asText(input["ref"]), false)
	if err != nil {
		return nil, err
	}
	if job.State != "completed" || job.Result["source"] != "abyss" {
		return nil, gameError("statistics_missing", "请先读取深渊统计。")
	}
	choice := Selection{asText(input["account_ref"]), asText(input["role_ref"])}
	panels, err := a.accountPanels(ctx, a.accountClient(event), choice)
	if err != nil {
		return nil, err
	}
	data, err := a.statistic(ctx, "abyss")
	if err != nil {
		return nil, err
	}
	weights := map[string]float64{}
	for _, panel := range panels {
		weights[panel.ID] = float64(a.teamWeight(panel))
	}
	pairs, _ := abyssTeamPairs(data, a.Catalog, weights)
	named := []map[string]any{}
	for _, pair := range pairs {
		half := func(team AbyssTeam) map[string]any {
			names := []string{}
			for _, id := range team.IDs {
				entry, _ := a.Catalog.Get(id)
				names = append(names, entry.Name)
			}
			return map[string]any{"names": names, "owned": team.Owned}
		}
		named = append(named, map[string]any{"up": half(pair.Up), "down": half(pair.Down), "count": pair.Count})
	}
	return map[string]any{"pairs": named, "note": "按 miao 的深渊配队：角色与武器较低等级乘 100 加最高原始天赋乘 1000 作为权重，队伍按样本出场数乘权重排序，缺少角色的队伍只按出场数计。", "source": "Yshelper"}, nil
}
