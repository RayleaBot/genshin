package app

import (
	"context"
	"encoding/json"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ark-plugin's panelRank (model/init.js, its ProfileDetail.render): a panel
// with a damage table also shows where ark ranks it among the panels ark
// keeps, by damage, by artifacts or both as rows after the table, or, with
// queryType 3, as 排名统计: the two ranking distributions with the panel
// marked on each.

// PanelRank is what a panel shows of ark's ranks: rows after the damage
// table, or the 排名统计 block.
type PanelRank struct {
	Rows  []Row
	Chart *PanelRankChart
}

// PanelRankChart is 排名统计: the damage and artifact distributions, each
// nil when ark has none, and the ranks written under them.
type PanelRankChart struct {
	Damage, Artis *RankCurve
	Places        [2]string
}

// RankCurve is a ranking distribution ark draws: the scores at the ranking
// percentiles of cloudPercentiles, the score at the top, and the panel's
// place (its percent from the top) and score, -100 each when ark has none.
type RankCurve struct {
	Scores         []float64
	Top            float64
	Percent, Score float64
}

// panelRankTitles are ark's rows by queryType: the index of the answer
// they read and their title.
var panelRankTitles = [][]struct {
	index int
	title string
}{{{0, "总伤害排名"}}, {{0, "圣遗物排名"}}, {{0, "总伤害排名"}, {1, "圣遗物排名"}}, {{0, "总伤害排名"}, {1, "圣遗物排名"}}}

// panelRank asks ark for a panel's ranks as the ark settings choose: with
// localPanelRank the panel itself under ark's stand-in UID 999999999, else
// the UID's panel ark keeps. RankType writes each rank as the place, the
// percent or both; a failed answer is left out unless both are written, as
// 暂无数据. markRankType marks the ranks of the panel as (本地), or (面板变换)
// for a changed one.
func (a *App) panelRank(ctx context.Context, event *rayleabot.EventContext, panel CharacterPanel, uid string, changed bool) *PanelRank {
	ark := settings(event).Ark
	kind := min(max(ark.QueryType, 0), 3)
	id, _ := strconv.Atoi(panel.ID)
	body := map[string]any{"id": id, "uid": uid, "update": 0, "query": []string{"dmg", "mark", "all", "all"}[kind]}
	var answers []any
	send := true
	if ark.LocalPanelRank {
		body["uid"] = "999999999"
		_, raw, err := panelCloudAvatar(ctx, a.Game, panel)
		// ark sends miao's own profile; one the plugin cannot write out has no
		// place.
		body["data"], send = json.RawMessage(raw), err == nil
	}
	if send {
		decoded, err := a.arkRequest(ctx, event, "rank/data", body)
		if answers = asList(decoded); err == nil && answers == nil {
			answers = []any{decoded}
		}
	}
	var distributions []any
	if kind == 3 {
		decoded, _ := a.arkRequest(ctx, event, "rank/specific", map[string]any{"id": id, "percent": 0})
		distributions = asList(decoded)
	}
	return readPanelRank(ark, answers, distributions, changed)
}

// readPanelRank writes ark's answers as the panel shows them: rank/data's
// damage and artifact ranks, and for 排名统计 rank/specific's distributions,
// the artifact one first.
func readPanelRank(ark ArkSettings, answers, distributions []any, changed bool) *PanelRank {
	kind := min(max(ark.QueryType, 0), 3)
	suffix := ""
	if ark.MarkRankType {
		suffix = "(本地)"
		if changed {
			suffix = "(面板变换)"
		}
	}
	rank := &PanelRank{}
	curves := [2]RankCurve{{Percent: -100, Score: -100}, {Percent: -100, Score: -100}}
	places := [2]string{}
	for _, row := range panelRankTitles[kind] {
		var item map[string]any
		if row.index < len(answers) {
			item = asObject(answers[row.index])
		}
		if asText(item["retcode"]) != "100" && ark.RankType != 2 {
			continue
		}
		place, percent := asText(item["rank"]), asText(item["percent"])
		text := "暂无数据"
		switch {
		case ark.RankType == 0 && place != "":
			text = place
		case ark.RankType == 1 && percent != "":
			text = percent
		case ark.RankType == 2 && place != "":
			text = place + " (" + percent + "%)"
		}
		if kind < 3 {
			rank.Rows = append(rank.Rows, Row{Label: row.title + suffix, Value: text})
			continue
		}
		places[row.index] = text
		if value, err := strconv.ParseFloat(percent, 64); err == nil {
			curves[row.index].Percent = value
		}
		if value := cloudFloat(item["score"]); value != 0 {
			curves[row.index].Score = value
		}
	}
	if kind < 3 {
		return rank
	}
	chart := &PanelRankChart{Places: places}
	if len(distributions) > 1 {
		chart.Damage = rankCurve(distributions[1], curves[0], true)
	}
	if len(distributions) > 0 {
		chart.Artis = rankCurve(distributions[0], curves[1], false)
	}
	if chart.Damage != nil || chart.Artis != nil {
		rank.Chart = chart
	}
	return rank
}

// rankCurve reads a distribution of rank/specific for 排名统计: the damage
// scores as percents of the top one, which tops the curve at 100, the
// artifact scores as they are, topped by the best score; the panel's own
// artifact score is a percent of the best one too.
func rankCurve(answer any, own RankCurve, damage bool) *RankCurve {
	data := asObject(asObject(answer)["data"])
	scores := asList(data["scores"])
	if len(scores) != len(cloudPercentiles) {
		return nil
	}
	top, divisor := cloudFloat(data["top1"]), 1.0
	if data["top1"] != nil {
		divisor = top
	}
	curve := &RankCurve{Scores: []float64{}, Top: top, Percent: own.Percent, Score: own.Score}
	for _, score := range scores {
		value := cloudFloat(score)
		if damage {
			value = value / divisor * 100
		}
		curve.Scores = append(curve.Scores, value)
	}
	if damage {
		curve.Top = 100
	} else {
		curve.Score = own.Score * top / 100
	}
	return curve
}

// groupTotalRanks is ark-plugin's groupRank (its renderCharRankList): a
// character's group ranking by damage or artifacts also shows the global
// rank ark gives each listed panel, sent with localGroupRank, else read from
// the UIDs ark keeps. A row without damage shows none, as upstream writes the
// rank into the damage cell; the column is left out when ark ranks none of
// them. Upstream asks for the first row's character on the list of every
// character's best too, which ranks the other rows as the wrong character;
// that list shows no column here.
func (a *App) groupTotalRanks(ctx context.Context, event *rayleabot.EventContext, character Entry, mode string, entries []RankEntry) (string, []string) {
	ark := settings(event).Ark
	if !ark.GroupRank || character.ID == "" || mode != "dmg" && mode != "mark" {
		return "", nil
	}
	id, _ := strconv.Atoi(character.ID)
	uids, panels := []string{}, []any{}
	for _, entry := range entries {
		uids = append(uids, entry.UID)
		if ark.LocalGroupRank {
			// A panel the plugin cannot write out goes as null, as upstream
			// sends a UID without player data.
			var panel any
			if _, raw, err := panelCloudAvatar(ctx, a.Game, *entry.Panel); err == nil {
				panel = json.RawMessage(raw)
			}
			panels = append(panels, panel)
		}
	}
	body := map[string]any{"id": id, "uids": uids, "update": 2, "query": mode, "data": nil}
	if len(panels) > 0 {
		body["data"] = panels
	}
	decoded, err := a.arkRequest(ctx, event, "rank/group", body)
	if err != nil {
		return "", nil
	}
	return groupRankColumn(ark, mode, entries, asObject(decoded))
}

// groupRankColumn reads ark's rank/group answer, one rank per listed UID in
// order, as the column's title and texts.
func groupRankColumn(ark ArkSettings, mode string, entries []RankEntry, result map[string]any) (string, []string) {
	if asText(result["retcode"]) != "100" {
		return "", nil
	}
	title := map[string]string{"dmg": "伤害", "mark": "圣遗物"}[mode] + "排名"
	if ark.MarkRankType && ark.LocalGroupRank {
		title += "(本地)"
	}
	ranks, ranked := make([]string, len(entries)), false
	for index, item := range asList(result["rank"]) {
		if index < len(entries) && entries[index].Damage != nil {
			rank := asText(asObject(item)["rank"])
			ranks[index], ranked = cmpOr(rank, "暂无数据"), ranked || rank != ""
		}
	}
	if !ranked {
		return "", nil
	}
	return title, ranks
}
