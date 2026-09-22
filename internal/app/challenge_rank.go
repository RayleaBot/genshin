package app

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type ChallengeMetric struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Lower bool   `json:"lower"`
}
type ChallengeKind struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Operation string            `json:"operation"`
	Metrics   []ChallengeMetric `json:"metrics"`
	Required  []string          `json:"required"`
}
type ChallengeEntry struct {
	Kind      string             `json:"kind"`
	Season    string             `json:"season"`
	Region    string             `json:"region"`
	ActorID   string             `json:"actor_id"`
	Nickname  string             `json:"nickname"`
	UID       string             `json:"uid"`
	Metrics   map[string]float64 `json:"metrics"`
	FirstMS   int64              `json:"first_ms"`
	UpdatedMS int64              `json:"updated_ms"`
}

func challengeKinds() []ChallengeKind {
	defs := [][4]string{{"abyss", "深渊", "abyss", "floor star battle"}, {"theater", "剧诗", "theater", "mode floor medal time borrow"}, {"hard_single", "幽境单人", "hard_challenge", "difficulty time"}, {"hard_mp", "幽境多人", "hard_challenge", "difficulty time"}}
	labels := map[string]string{"floor": "最深层/幕", "star": "星数", "battle": "战斗次数", "mode": "模式", "medal": "星章", "time": "用时（秒）", "borrow": "借出次数", "difficulty": "难度"}
	out := []ChallengeKind{}
	for _, d := range defs {
		v := ChallengeKind{ID: d[0], Label: d[1], Operation: "genshin." + d[2]}
		for _, key := range strings.Fields(d[3]) {
			v.Metrics = append(v.Metrics, ChallengeMetric{key, labels[key], slices.Contains([]string{"time", "battle"}, key)})
		}
		v.Required = []string{v.Metrics[0].Key}
		out = append(out, v)
	}
	return out
}
func challengeKind(id string) (ChallengeKind, bool) {
	for _, k := range challengeKinds() {
		if strings.EqualFold(id, k.ID) || strings.EqualFold(id, k.Label) {
			return k, true
		}
	}
	return ChallengeKind{}, false
}
func fieldAt(data any, path string) any {
	value := data
	for _, key := range strings.Split(path, ".") {
		switch v := value.(type) {
		case map[string]any:
			value = v[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(v) {
				return nil
			}
			value = v[i]
		default:
			return nil
		}
	}
	return value
}
func challengeNumber(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	n, e := strconv.ParseFloat(asText(v), 64)
	return n, e == nil && finiteRange(n, 0, 1e14)
}
func challengeSeason(data map[string]any) string {
	for _, p := range []string{"schedule_id", "schedule.schedule_id", "group.group_id", "groups.0.group_id", "void_front_id"} {
		if id := asText(fieldAt(data, p)); id != "" && len(id) <= 128 {
			return id
		}
	}
	for _, p := range []string{"start_time", "begin_time", "begin_ts", "schedule.start_time", "schedule.start_date_time", "group.begin_time", "groups.0.begin_time"} {
		v := fieldAt(data, p)
		if date := calendarTime(v); date != "" {
			return strings.ReplaceAll(date, " ", "T")
		}
	}
	return ""
}
func challengeExtract(kind ChallengeKind, data map[string]any, period int) (ChallengeEntry, error) {
	out := ChallengeEntry{Kind: kind.ID, Metrics: map[string]float64{}}
	put := func(key string, paths ...string) {
		for _, p := range paths {
			if v, ok := challengeNumber(fieldAt(data, p)); ok {
				out.Metrics[key] = v
				return
			}
		}
	}
	if kind.ID == "theater" || strings.HasPrefix(kind.ID, "hard_") {
		list := asList(data["data"])
		if period < 1 || period > len(list) {
			return out, gameError("challenge_unavailable", "官方未返回所选期次的成绩。")
		}
		data = asObject(list[period-1])
	}
	out.Season = challengeSeason(data)
	switch kind.ID {
	case "abyss":
		parts := strings.Split(asText(data["max_floor"]), "-")
		if len(parts) == 2 {
			f, e := strconv.Atoi(parts[0])
			c, e2 := strconv.Atoi(parts[1])
			if e == nil && e2 == nil && f >= 0 && c >= 0 && c <= 9 {
				out.Metrics["floor"] = float64(f*10 + c)
			}
		}
		highest := -1
		for _, v := range asList(data["floors"]) {
			item := asObject(v)
			idx := number(item["index"])
			if idx > highest {
				highest = idx
				if star, ok := challengeNumber(item["star"]); ok {
					out.Metrics["star"] = star
				} else {
					delete(out.Metrics, "star")
				}
			}
		}
		put("battle", "total_battle_times")
	case "theater":
		put("mode", "stat.difficulty_id")
		if rounds, ok := fieldAt(data, "detail.rounds_data").([]any); ok {
			out.Metrics["floor"] = float64(len(rounds))
		}
		if medals, ok := fieldAt(data, "stat.get_medal_round_list").([]any); ok {
			out.Metrics["medal"] = 0
			for _, v := range medals {
				if n, ok := challengeNumber(v); ok && n == 1 {
					out.Metrics["medal"]++
				}
			}
		}
		put("time", "stat.total_use_time", "detail.fight_statisic.total_use_time")
		put("borrow", "stat.rent_cnt")
	case "hard_single", "hard_mp":
		mode := strings.TrimPrefix(kind.ID, "hard_")
		put("difficulty", mode+".best.difficulty", mode+".difficulty")
		put("time", mode+".best.second", mode+".second")
	}
	if out.Season == "" {
		return out, gameError("challenge_unavailable", "官方未提供可区分的期次，未加入群榜。")
	}
	for _, key := range kind.Required {
		if _, ok := out.Metrics[key]; !ok {
			return out, gameError("challenge_unavailable", "官方未提供此玩法的必要成绩，未加入群榜。")
		}
	}
	return out, nil
}
func challengeCompare(kind ChallengeKind, dimension string, a, b ChallengeEntry) int {
	metrics := kind.Metrics
	if dimension != "" {
		metrics = slices.DeleteFunc(slices.Clone(metrics), func(m ChallengeMetric) bool { return m.Key != dimension })
	}
	for _, m := range metrics {
		x, ok := a.Metrics[m.Key]
		y, ok2 := b.Metrics[m.Key]
		if ok != ok2 {
			if ok {
				return -1
			}
			return 1
		}
		if !ok || x == y {
			continue
		}
		if (x < y) == m.Lower {
			return -1
		}
		return 1
	}
	if a.FirstMS < b.FirstMS {
		return -1
	}
	if a.FirstMS > b.FirstMS {
		return 1
	}
	return strings.Compare(a.ActorID, b.ActorID)
}
func (s *GroupStore) SubmitChallenge(scope GroupScope, entry ChallengeEntry) error {
	return s.Update(scope, func(data *GroupData) error {
		if data.Config.Enabled != nil && !*data.Config.Enabled {
			return gameError("group_disabled", "本群游戏功能已关闭。")
		}
		data.Challenges = slices.DeleteFunc(data.Challenges, func(v ChallengeEntry) bool { return v.UpdatedMS < time.Now().AddDate(0, 0, -180).UnixMilli() })
		i := slices.IndexFunc(data.Challenges, func(v ChallengeEntry) bool {
			return v.Kind == entry.Kind && v.Season == entry.Season && v.Region == entry.Region && v.ActorID == entry.ActorID
		})
		if i >= 0 {
			entry.FirstMS = data.Challenges[i].FirstMS
			data.Challenges[i] = entry
			return nil
		}
		if len(data.Challenges) >= 2000 {
			return gameError("rank_limit", "本群挑战记录已满，请清理旧期次。")
		}
		data.Challenges = append(data.Challenges, entry)
		return nil
	})
}
func (a *App) challengeList(scope GroupScope, kind ChallengeKind, season, dimension string, offset int) (map[string]any, error) {
	if offset < 0 || offset > 2000 || len(season) > 128 {
		return nil, gameError("input_invalid", "挑战榜筛选无效。")
	}
	if dimension != "" && !slices.ContainsFunc(kind.Metrics, func(v ChallengeMetric) bool { return v.Key == dimension }) {
		return nil, gameError("input_invalid", "此玩法没有所选指标。")
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return nil, err
	}
	groups := map[string][]ChallengeEntry{}
	for _, v := range data.Challenges {
		if v.Kind == kind.ID && (season == "" || v.Season == season) && v.UpdatedMS >= time.Now().AddDate(0, 0, -180).UnixMilli() {
			key := v.Region + " · " + v.Season
			groups[key] = append(groups[key], v)
		}
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	view := View{Title: a.Game.Name + kind.Label + "群榜", Rows: []Row{}, Note: "本群主动提交的官方成绩快照，按区服及期次分组，保存 180 天；查询不会刷新成绩。相同指标按首次提交排序。"}
	sections := []map[string]any{}
	for _, key := range keys {
		rows := groups[key]
		slices.SortStableFunc(rows, func(x, y ChallengeEntry) int { return challengeCompare(kind, dimension, x, y) })
		start := min(offset, len(rows))
		end := min(start+20, len(rows))
		var next *int
		if end < len(rows) {
			next = &end
		}
		sections = append(sections, map[string]any{"partition": key, "items": rows[start:end], "total": len(rows), "next_offset": next})
		section := Section{Title: key, Rows: []Row{}}
		for i, r := range rows[start:end] {
			values := []string{}
			for _, m := range kind.Metrics {
				if value, ok := r.Metrics[m.Key]; ok {
					values = append(values, m.Label+" "+strconv.FormatFloat(value, 'f', -1, 64))
				}
			}
			section.Rows = append(section.Rows, Row{Label: fmt.Sprintf("%d. %s · %s", start+i+1, r.Nickname, r.UID), Value: strings.Join(values, " · ")})
		}
		view.Sections = append(view.Sections, section)
	}
	return map[string]any{"groups": sections, "view": view, "revision": data.Revision}, nil
}
func (a *App) manageChallenge(action string, input map[string]any) (map[string]any, error) {
	if action == "challenge.schema" {
		return map[string]any{"kinds": challengeKinds()}, nil
	}
	var q struct {
		Scope     GroupScope `json:"scope"`
		Kind      string     `json:"kind"`
		Season    string     `json:"season"`
		Dimension string     `json:"dimension"`
		Offset    int        `json:"offset"`
		Revision  uint64     `json:"revision"`
		Confirmed bool       `json:"confirmed"`
	}
	if decodeObject(input, &q) != nil || !q.Scope.valid() {
		return nil, gameError("input_invalid", "挑战榜输入无效。")
	}
	kind, ok := challengeKind(q.Kind)
	if !ok && !(action == "challenge.clear" && q.Kind == "") {
		return nil, gameError("input_invalid", "请选择本游戏支持的玩法。")
	}
	if action == "challenge.list" {
		return a.challengeList(q.Scope, kind, q.Season, q.Dimension, q.Offset)
	}
	if action != "challenge.clear" || !q.Confirmed {
		return nil, gameError("input_invalid", "请明确确认要清空的群榜。")
	}
	err := a.Groups.Update(q.Scope, func(data *GroupData) error {
		if data.Revision != q.Revision {
			return gameError("group_changed", "群榜已变化，请刷新后再清空。")
		}
		data.Challenges = slices.DeleteFunc(data.Challenges, func(v ChallengeEntry) bool {
			return (q.Kind == "" || v.Kind == kind.ID) && (q.Season == "" || q.Season == v.Season)
		})
		return nil
	})
	return map[string]any{"cleared": err == nil}, err
}
func (a *App) challengeCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if event.Event.EventType != "message.group" || event.Event.Target.Type != "group" || !groupScope(event).valid() || event.Event.Actor.ID == "" {
		return event.SendText("请在群内使用挑战群榜。")
	}
	scope := groupScope(event)
	kindID := ""
	if len(args) > 0 {
		kindID = args[0]
	}
	kind, ok := challengeKind(kindID)
	withdraw := command == "challenge-withdraw"
	clear := command == "challenge-clear"
	if (withdraw || clear) && (kindID == "" || ok) {
		if clear && !groupAdministrator(event) {
			return event.SendText("清空群榜需要群管理员身份。")
		}
		removed := 0
		err := a.Groups.Update(scope, func(data *GroupData) error {
			before := len(data.Challenges)
			data.Challenges = slices.DeleteFunc(data.Challenges, func(v ChallengeEntry) bool {
				return (clear || v.ActorID == event.Event.Actor.ID) && (kindID == "" || v.Kind == kind.ID)
			})
			removed = before - len(data.Challenges)
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(fmt.Sprintf("已移除本群 %d 条挑战参评记录。", removed))
	}
	if !ok {
		names := []string{}
		for _, v := range challengeKinds() {
			names = append(names, v.Label)
		}
		return event.SendText("可选玩法：" + strings.Join(names, "、") + "。\n提交挑战 玩法 [本期/上期] [UID]：向本群公开自己的 UID、昵称及官方成绩。\n挑战排行 玩法 [期次] [指标] [页码]；退出挑战 [玩法]。")
	}
	if command == "challenge-rank" {
		season, dimension := "", ""
		offset := 0
		if len(args) > 1 && args[1] != "全部" {
			season = args[1]
		}
		if len(args) > 2 {
			dimension = args[2]
			if dimension == "综合" {
				dimension = ""
			}
		}
		if len(args) > 3 {
			page, e := strconv.Atoi(args[3])
			if e != nil || page < 1 || page > 100 {
				return event.SendText("页码须为 1–100。")
			}
			offset = (page - 1) * 20
		}
		result, err := a.challengeList(scope, kind, season, dimension, offset)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendView(ctx, event, result["view"].(View))
	}
	period := 1
	uid := ""
	for _, arg := range args[1:] {
		switch arg {
		case "本期", "1":
			period = 1
		case "上期", "2":
			period = 2
		default:
			if uid != "" || !uidPattern.MatchString(arg) {
				return event.SendText("格式：提交挑战 玩法 [本期/上期] [UID]。")
			}
			uid = arg
		}
	}
	client := a.accountClient(event)
	accounts, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, role, err := Choose(accounts, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	input := map[string]any{}
	op, _ := a.operation(kind.Operation)
	if op.Input == "period" || op.Input == "peak_period" {
		input["schedule_type"] = period
	}
	result, err := client.Execute(ctx, choice, kind.Operation, input)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	entry, err := challengeExtract(kind, result.Data, period)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	entry.ActorID = event.Event.Actor.ID
	entry.Nickname = plainGameText(event.Event.Actor.Nickname)
	if entry.Nickname == "" {
		entry.Nickname = entry.ActorID
	}
	entry.UID = role.UID
	entry.Region = role.Region
	entry.FirstMS = time.Now().UnixMilli()
	entry.UpdatedMS = entry.FirstMS
	if ctx.Err() != nil {
		return event.SendText("成绩读取超时，请重试。")
	}
	if err = a.Groups.SubmitChallenge(scope, entry); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("已提交本群" + kind.Label + "成绩 · " + entry.Season + " · UID " + role.UID + "。\n发送“" + a.Game.Prefix + "退出挑战 " + kind.Label + "”可移除。")
}
