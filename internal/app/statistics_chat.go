package app

import (
	"context"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// miao's stat commands: 角色持有率 and 命座分布 (stat/character), 深渊 and
// 幽境使用率 (stat/abyss-pct) from the public statistics, and 深渊配队
// (stat/abyss-team) matching the usage statistics to the sender's
// characters.

// StatisticsPage is one of the pages; exactly one field is set.
type StatisticsPage struct {
	Cons  *ConsStatPage
	Usage *AbyssUsagePage
	Teams *AbyssTeamPage
}

// StatisticsImageBuilder draws a StatisticsPage.
type StatisticsImageBuilder func(ImageContext, StatisticsPage) (Image, bool)

// ConsStatPage is miao's ConsStat: every character's holding rate and
// constellation shares; in the 持有 mode the shares are of all players.
type ConsStatPage struct {
	Holding    bool
	Constel    int // the constellation sorted by, -1 for the holding rate
	Characters []ConsStatCharacter
	TotalCount int
	LastUpdate string
}

type ConsStatCharacter struct {
	Entry Entry
	Hold  *float64
	Cons  [7]float64
}

// AbyssUsagePage is miao's AbyssPct: the characters of each rank of the last
// floor by usage rate.
type AbyssUsagePage struct {
	Name       string // 深渊 or 幽境危战
	FloorName  string
	Chosen     bool
	Ranks      []UsageRank
	TotalCount string
	LastUpdate string
}

type UsageRank struct {
	Name       string
	Characters []UsageCharacter
}

type UsageCharacter struct {
	Entry Entry
	Value float64
}

// AbyssTeamPage is miao's AbyssTeam: the best four pairs of floor 12 halves,
// with the sender's characters and those the pairs need but they lack.
type AbyssTeamPage struct {
	Pairs   []AbyssTeamPair
	Avatars map[string]TeamAvatar
}

type TeamAvatar struct {
	Entry       Entry
	Level, Cons int
}

var consDigits = []*regexp.Regexp{regexp.MustCompile(`0|零`), regexp.MustCompile(`1|一`), regexp.MustCompile(`2|二`), regexp.MustCompile(`3|三`), regexp.MustCompile(`4|四`), regexp.MustCompile(`5|五`), regexp.MustCompile(`6|六|满`)}

// statNumberOf is a statistics number, 0 when absent.
func statNumberOf(value any) float64 {
	n, _ := strconv.ParseFloat(asText(value), 64)
	return n
}

// consStat builds ConsStat from lelaer's averages and yshelper's holding
// rates. miao keys the averages by character ID, which JavaScript walks in
// ascending order, sorts ascending and reverses.
func (a *App) consStat(ctx context.Context, word string) (*ConsStatPage, error) {
	averages, err := a.statistic(ctx, "ownership")
	if err != nil {
		return nil, err
	}
	page := &ConsStatPage{Holding: strings.Contains(word, "持有"), Constel: -1, LastUpdate: asText(averages["last_update"])}
	if !page.Holding {
		for index, pattern := range consDigits {
			if pattern.MatchString(word) {
				page.Constel = index
				break
			}
		}
	}
	owned := map[string]float64{}
	if holding, err := a.statistic(ctx, "abyss"); err == nil {
		for _, raw := range asList(holding["has_list"]) {
			item := asObject(raw)
			if entry, ok := a.Catalog.Resolve(asText(item["name"]), "character", nil); ok {
				owned[entry.ID] = statNumberOf(item["own_rate"])
			}
		}
	}
	rows := map[string]map[string]any{}
	for _, raw := range asList(averages["result"]) {
		item := asObject(raw)
		if entry, ok := a.Catalog.Resolve(asText(item["role"]), "character", nil); ok {
			rows[entry.ID] = item
			page.TotalCount = max(page.TotalCount, int(statNumberOf(item["role_sum"])))
		}
	}
	ids := make([]string, 0, len(rows))
	for id := range rows {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return numericLess(ids[i], ids[j]) })
	for _, id := range ids {
		entry, ok := a.Catalog.Get(id)
		if !ok || id == "10000005" || id == "10000007" || id == "20000000" {
			continue
		}
		character := ConsStatCharacter{Entry: entry}
		// miao counts a zero holding rate as unknown.
		if rate := owned[id]; rate != 0 {
			hold := rate / 100
			character.Hold = &hold
		}
		for index := range character.Cons {
			character.Cons[index] = statNumberOf(rows[id]["c"+strconv.Itoa(index)]) / 100
			if page.Holding {
				hold := 0.0
				if character.Hold != nil {
					hold = *character.Hold
				}
				character.Cons[index] *= hold
			}
		}
		page.Characters = append(page.Characters, character)
	}
	key := func(c ConsStatCharacter) float64 {
		if page.Constel >= 0 {
			return c.Cons[page.Constel]
		}
		if c.Hold == nil {
			return -1
		}
		return *c.Hold
	}
	sort.SliceStable(page.Characters, func(i, j int) bool { return key(page.Characters[i]) < key(page.Characters[j]) })
	slices.Reverse(page.Characters)
	return page, nil
}

// numericLess orders numeric IDs as numbers.
func numericLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// abyssUsage builds AbyssPct: yshelper's abyss ranking or lelaer's 幽境危战
// ranking, whose first list holds the S+ to C ranks of the last floor.
func (a *App) abyssUsage(ctx context.Context, word string) (*AbyssUsagePage, error) {
	page := &AbyssUsagePage{Name: "深渊", FloorName: "十二层"}
	source := "abyss"
	if strings.Contains(word, "幽境") || strings.Contains(word, "危战") {
		page.Name, page.FloorName, source = "幽境危战", "5&6层", "stygian"
	}
	data, err := a.statistic(ctx, source)
	if err != nil {
		return nil, err
	}
	page.Chosen = strings.Contains(word, page.FloorName) || strings.Contains(word, "12")
	page.TotalCount, page.LastUpdate = asText(data["top_own"]), asText(data["last_update"])
	if page.TotalCount == "0" {
		page.TotalCount = ""
	}
	order := []string{"S+", "S", "A", "B", "C"}
	results := asList(data["result"])
	if len(results) == 0 {
		return page, nil
	}
	for _, raw := range asList(results[0]) {
		group := asObject(raw)
		name := asText(group["rank_name"])
		if !slices.Contains(order, name) {
			continue
		}
		rank := UsageRank{Name: name}
		for _, rawCharacter := range asList(group["list"]) {
			item := asObject(rawCharacter)
			entry, ok := a.Catalog.Resolve(strings.TrimSpace(asText(item["name"])), "character", nil)
			if rate := statNumberOf(item["use_rate"]); ok && rate > 0 {
				rank.Characters = append(rank.Characters, UsageCharacter{Entry: entry, Value: rate / 100})
			}
		}
		sort.SliceStable(rank.Characters, func(i, j int) bool { return rank.Characters[i].Value < rank.Characters[j].Value })
		slices.Reverse(rank.Characters)
		page.Ranks = append(page.Ranks, rank)
	}
	sort.SliceStable(page.Ranks, func(i, j int) bool {
		return slices.Index(order, page.Ranks[i].Name) < slices.Index(order, page.Ranks[j].Name)
	})
	return page, nil
}

// teamWeight is miao's weight of a character in 深渊配队: the lower of its
// level and its weapon's level times 100, plus its highest talent before
// constellation bonuses times 1000.
func (a *App) teamWeight(panel CharacterPanel) int {
	weapon := 1
	if panel.Weapon != nil && panel.Weapon.Level > 0 {
		weapon = panel.Weapon.Level
	}
	talent := 1
	if record, err := findBuildCharacter(a.Game.Calc, panel); err == nil {
		for _, level := range PanelTalents(panel, record) {
			talent = max(talent, level.Original)
		}
	}
	return min(panel.Level, weapon)*100 + talent*1000
}

// abyssTeams reads the sender's characters from the account, as miao
// refreshes them, and pairs the usage statistics' floor 12 teams.
func (a *App) abyssTeams(ctx context.Context, event *rayleabot.EventContext, owner panelOwner) (*AbyssTeamPage, error) {
	panels, err := a.accountPanels(ctx, a.accountClient(event), owner.Choice)
	if err != nil {
		return nil, err
	}
	if len(panels) > 0 {
		_, _ = a.Profiles.Keep(owner.UID, panels, "米游社", &ShowcaseProfile{Nickname: owner.Role.Nickname, Level: owner.Role.Level})
	}
	data, err := a.statistic(ctx, "abyss")
	if err != nil {
		return nil, err
	}
	weights := map[string]float64{}
	page := &AbyssTeamPage{Avatars: map[string]TeamAvatar{}}
	for _, panel := range panels {
		weights[panel.ID] = float64(a.teamWeight(panel))
		entry, _ := a.Catalog.Get(panel.ID)
		if entry.ID == "" {
			entry = Entry{ID: panel.ID, Name: panel.Name, Kind: "character"}
		}
		page.Avatars[panel.ID] = TeamAvatar{Entry: entry, Level: panel.Level, Cons: panel.Rank}
	}
	pairs, missing := abyssTeamPairs(data, a.Catalog, weights)
	page.Pairs = pairs
	for id := range missing {
		if entry, ok := a.Catalog.Get(id); ok {
			page.Avatars[id] = TeamAvatar{Entry: entry}
		}
	}
	return page, nil
}

// statisticsCommand answers 角色持有率, 命座分布, 深渊使用率, 幽境使用率 and 深渊配队.
func (a *App) statisticsCommand(ctx context.Context, event *rayleabot.EventContext, command string) error {
	word := event.Event.Command()
	page := StatisticsPage{}
	view := View{Rows: []Row{}}
	switch command {
	case "stat-cons":
		cons, err := a.consStat(ctx, word)
		if err != nil || len(cons.Characters) == 0 {
			return event.SendText("角色持有数据获取失败，请稍后重试~")
		}
		page.Cons = cons
		view.Title = "角色持有率"
		for _, character := range cons.Characters {
			hold := "未知"
			if character.Hold != nil {
				hold = strconv.FormatFloat(*character.Hold*100, 'f', 2, 64) + "%"
			}
			view.Rows = append(view.Rows, Row{Label: character.Entry.Name, Value: hold})
		}
	case "stat-usage":
		usage, err := a.abyssUsage(ctx, word)
		if err != nil {
			name := "深渊"
			if strings.Contains(word, "幽境") || strings.Contains(word, "危战") {
				name = "幽境危战"
			}
			return event.SendText(name + "使用率数据获取失败，请稍后重试~")
		}
		page.Usage = usage
		view.Title = usage.Name + "使用率"
		for _, rank := range usage.Ranks {
			names := []string{}
			for _, character := range rank.Characters {
				names = append(names, character.Entry.Name+" "+strconv.FormatFloat(character.Value*100, 'f', 2, 64)+"%")
			}
			view.Rows = append(view.Rows, Row{Label: rank.Name, Value: strings.Join(names, "、")})
		}
	case "abyss-team":
		owner, err := a.panelOwner(ctx, event, "")
		if err != nil || !owner.Owned {
			return event.SendText("请绑定ck后再使用" + a.Game.Prefix + word)
		}
		teams, err := a.abyssTeams(ctx, event, owner)
		if err != nil {
			return event.SendText("深渊组队数据获取失败，请稍后重试~")
		}
		page.Teams = teams
		view.Title = "深渊配队建议"
		for index, pair := range teams.Pairs {
			names := func(team AbyssTeam) string {
				out := []string{}
				for _, id := range team.IDs {
					out = append(out, teams.Avatars[id].Entry.Name)
				}
				return strings.Join(out, "、")
			}
			view.Rows = append(view.Rows, Row{Label: "配队" + strconv.Itoa(index+1), Value: "上半 " + names(pair.Up) + " / 下半 " + names(pair.Down)})
		}
	}
	if a.statisticsImage != nil {
		if drawn, ok := a.statisticsImage(a.imageContext(ctx), page); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}
