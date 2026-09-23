package app

import (
	"cmp"
	"context"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ark-plugin's group 幽境危战 ranking (model/stygian-init.js, apps/user.js
// stygian): 更新面板 in a group enters the sender's own UID with the result
// its showcase shows for the current season, and 幽境危战排名 lists the
// group's entries with their global places on ark and akasha.cv, drawn as
// ark's character/stygian-rank-list.

// StygianEntry is a UID entered in a group's ranking for a season, as ark's
// stygianRank sorted set keeps it: the score is the seconds plus 2048 for each
// difficulty below the highest. The member who entered it names it and gives
// its avatar.
type StygianEntry struct {
	UID      string `json:"uid"`
	Score    int    `json:"score"`
	ActorID  string `json:"actor_id"`
	Nickname string `json:"nickname"`
}

// stygianFirst is the first season ark's ranking may be asked for, 5.7.
const stygianFirst = 5*9 + 7

var stygianDay = 24 * time.Hour

// stygianSeason is ark's getStygianVersion (model/calcVersion.js): a
// version a.b counts as a*9+b, and the season is that of the latest event
// banners, plus one for every 42 days after the season began, seven days and
// four hours after its first half opened. It is -1 without such banners.
// Before the latest season begins, ark counts back one number, which names a
// version that never was when versions skip numbers (6.7 was followed by
// 7.0); the version before it among the banners is taken instead.
func stygianSeason(pools []PoolInfo, now time.Time) int {
	seasons, opened := stygianSeasons(pools)
	if len(seasons) == 0 {
		return -1
	}
	latest := seasons[len(seasons)-1]
	start, ok := opened[latest]
	if !ok {
		return -1
	}
	step := int(math.Floor((now.Sub(start).Hours()/24 - 7.16666) / 42))
	if back := len(seasons) - 1 + step; step < 0 && back >= 0 {
		return seasons[back]
	}
	return latest + step
}

// stygianSeasons are the versions of the event banners as ark counts them,
// in order, with the time each version's first half opened.
func stygianSeasons(pools []PoolInfo) ([]int, map[int]time.Time) {
	seasons, opened := []int{}, map[int]time.Time{}
	for _, pool := range pools {
		season := stygianVersion(pool.Version)
		if pool.Kind != "event" || season < 0 {
			continue
		}
		if !slices.Contains(seasons, season) {
			seasons = append(seasons, season)
		}
		if from, err := time.ParseInLocation("2006-01-02 15:04:05", pool.From, chinaZone); err == nil && pool.Half == "上半" {
			if _, seen := opened[season]; !seen {
				opened[season] = from
			}
		}
	}
	slices.Sort(seasons)
	return seasons, opened
}

// stygianVersion reads a version a.b as ark counts it, -1 when it is none.
func stygianVersion(text string) int {
	major, minor, ok := strings.Cut(text, ".")
	a, errA := strconv.Atoi(major)
	b, errB := strconv.Atoi(minor)
	if !ok || errA != nil || errB != nil {
		return -1
	}
	return a*9 + b
}

// stygianVersionName writes a season as ark shows it.
func stygianVersionName(season int) string {
	return strconv.Itoa(season/9) + "." + strconv.Itoa(season%9)
}

// stygianPeriod is ark's getStygianPeriod: a season runs from ten o'clock
// seven days after its version's first half opens to 03:59:59 on the fifth
// Tuesday. ark counts 42 days per season from 6.0 (54); the banners date
// each season instead, since versions skip numbers and ark's count then
// gives 7.0 the dates of the season after it. A season past the banners
// follows the last one before it by 42 days a season.
func stygianPeriod(pools []PoolInfo, season int) (time.Time, time.Time) {
	start := time.Date(2025, 9, 17, 10, 0, 0, 0, chinaZone).Add(time.Duration(season-54) * 42 * stygianDay)
	seasons, opened := stygianSeasons(pools)
	for _, known := range seasons {
		if from, ok := opened[known]; ok && known <= season {
			start = time.Date(from.Year(), from.Month(), from.Day()+7, 10, 0, 0, 0, chinaZone).Add(time.Duration(season-known) * 42 * stygianDay)
		}
	}
	return start, start.Add(33*stygianDay + 17*time.Hour + 59*time.Minute + 59*time.Second)
}

// chinaZone is the time zone of the banner dates.
var chinaZone = time.FixedZone("UTC+8", 8*3600)

// enterStygian is ark's doRefresh addition: after 更新面板 in a group, a UID
// the sender has bound enters the group's ranking for the current season
// with the result its showcase shows. A showcase without a result enters
// nothing; upstream enters it as -1, which drops an earlier result of the
// season from the list.
func (a *App) enterStygian(ctx context.Context, event *rayleabot.EventContext, uid string, profile ShowcaseProfile) {
	scope := groupScope(event)
	season := stygianSeason(a.Game.Data.Resources.Pools, time.Now())
	if event.Event.EventType != "message.group" || !scope.valid() || !settings(event).Ark.StygianRank || profile.StygianIndex <= 0 || profile.StygianSeconds <= 0 || season < 0 {
		return
	}
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil || !slices.ContainsFunc(a.uidList(listed), func(bound listedUID) bool { return bound.UID == uid }) {
		return
	}
	entry := StygianEntry{UID: uid, Score: profile.StygianSeconds + (6-profile.StygianIndex)*2048, ActorID: event.Event.Actor.ID, Nickname: cmp.Or(event.Event.Actor.Nickname, event.Event.Actor.ID)}
	_ = a.Groups.Update(scope, func(data *GroupData) error {
		key := strconv.Itoa(season)
		if data.Stygian == nil {
			data.Stygian = map[string][]StygianEntry{}
		}
		entries := slices.DeleteFunc(data.Stygian[key], func(old StygianEntry) bool { return old.UID == uid })
		if len(entries) >= 500 {
			return gameError("rank_limit", "本群幽境危战排名记录已达上限。")
		}
		data.Stygian[key] = append(entries, entry)
		return nil
	})
}

// stygianEntries are a group's entries for a season in ark's order, by score
// then UID; a season before it adds the UIDs the requested season lacks.
func stygianEntries(data GroupData, season, older int) []StygianEntry {
	order := func(list []StygianEntry) []StygianEntry {
		list = slices.Clone(list)
		slices.SortStableFunc(list, func(x, y StygianEntry) int {
			return cmp.Or(cmp.Compare(x.Score, y.Score), strings.Compare(x.UID, y.UID))
		})
		return list
	}
	entries := order(data.Stygian[strconv.Itoa(season)])
	if older >= 0 {
		for _, entry := range order(data.Stygian[strconv.Itoa(older)]) {
			if !slices.ContainsFunc(entries, func(e StygianEntry) bool { return e.UID == entry.UID }) {
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

// StygianRankEntry is one row of 幽境危战排名: the UID, name and avatar URL
// shown, the difficulty and seconds, and the global places on ark and
// akasha.cv as ark writes them.
type StygianRankEntry struct {
	UID, Name, Avatar string
	Index, Seconds    int
	Ark, Akasha       string
}

// StygianRankImage is 幽境危战排名 as ark draws it: the season with its
// period, the rows, and whether each service answered.
type StygianRankImage struct {
	Version    string
	Start, End time.Time
	Entries    []StygianRankEntry
	Ark        bool
	Akasha     bool
}

// StygianRankImageBuilder draws 幽境危战排名 with the plugin's template, or
// returns false to answer in text.
type StygianRankImageBuilder func(ImageContext, StygianRankImage) (Image, bool)

// arkStygianPlace is a UID's line of ark's rank/stygian answer.
type arkStygianPlace struct {
	Rank, Sum      string
	Index, Seconds int
}

// akashaStygianPlace is a UID's row of akasha.cv's stygian leaderboard.
type akashaStygianPlace struct {
	Picture, Index      string
	Difficulty, Seconds int
}

// stygianRow merges a UID's results as ark does: the lower difficulty and
// the fewer seconds of akasha.cv, ark and the group's entry, each where it is
// known; a row without seconds or difficulty is left out.
func stygianRow(entry StygianEntry, ark *arkStygianPlace, akasha *akashaStygianPlace) (StygianRankEntry, bool) {
	const unknown = 99999
	known := func(value int) int {
		if value == 0 {
			return unknown
		}
		return value
	}
	var arkIndex, arkSeconds, akashaIndex, akashaSeconds int
	if ark != nil {
		arkIndex, arkSeconds = ark.Index, ark.Seconds
	}
	if akasha != nil {
		akashaIndex, akashaSeconds = akasha.Difficulty, akasha.Seconds
	}
	seconds := min(known(akashaSeconds), known(arkSeconds), known(entry.Score%2048))
	index := min(known(akashaIndex), known(arkIndex), known(6-int(math.Floor(float64(entry.Score)/2048))))
	if seconds == unknown || seconds < 0 || index == unknown {
		return StygianRankEntry{}, false
	}
	row := StygianRankEntry{UID: entry.UID, Index: index, Seconds: seconds, Ark: "? / ?", Akasha: "?"}
	if ark != nil {
		row.Ark = cmp.Or(ark.Rank, "?") + " / " + cmp.Or(ark.Sum, "?")
	}
	if akasha != nil {
		row.Akasha = cmp.Or(akasha.Index, "?")
	}
	return row, true
}

// stygianRankCommand is ark's 幽境危战排名[版本], in groups only: the
// group's entries for the current season, or for the version asked for with
// the current season's entries first, with their places on ark (the current
// season only) and akasha.cv as the ark settings choose, fastest first.
func (a *App) stygianRankCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	scope := groupScope(event)
	if event.Event.EventType != "message.group" || !scope.valid() {
		// Upstream swallows the words outside groups.
		return event.Result(map[string]any{"handled": true})
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	pools := a.Game.Data.Resources.Pools
	season, older := stygianSeason(pools, time.Now()), -1
	if len(args) > 0 {
		if asked := stygianVersion(args[0]); asked >= stygianFirst && asked != season {
			older = asked
		}
	}
	entries := stygianEntries(data, season, older)
	if older >= 0 {
		season = older
	}
	if len(entries) == 0 {
		return event.SendText("当前版本无排名....")
	}
	from := settings(event).Ark.StygianDataFrom
	var arkRows []arkStygianPlace
	var akashaRows map[string]akashaStygianPlace
	akashaTotal := ""
	var group sync.WaitGroup
	if from == 1 || from == 2 {
		group.Go(func() { akashaRows, akashaTotal = a.Cloud.akashaStygianRows(ctx, entries, stygianVersionName(season)) })
	}
	// ark keeps only the current season.
	if (from == 0 || from == 2) && older < 0 {
		arkRows = a.arkStygianPlaces(ctx, event, entries)
	}
	group.Wait()
	rows := []StygianRankEntry{}
	for index, entry := range entries {
		var ark *arkStygianPlace
		if index < len(arkRows) {
			ark = &arkRows[index]
		}
		var akasha *akashaStygianPlace
		if found, ok := akashaRows[entry.UID]; ok {
			akasha = &found
		}
		row, ok := stygianRow(entry, ark, akasha)
		if !ok {
			continue
		}
		if akashaTotal != "" {
			row.Akasha += " / " + akashaTotal
		}
		// As ark, the member who entered the UID names it and gives its
		// avatar, which akasha.cv's picture stands in for outside QQ.
		row.Name = entry.Nickname
		if akasha != nil {
			row.Avatar = akasha.Picture
		}
		if scope.Protocol == "onebot11" && entry.ActorID != "" {
			row.Avatar = "https://q1.qlogo.cn/g?b=qq&nk=" + entry.ActorID + "&s=100"
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return event.SendText("当前版本无排名....")
	}
	slices.SortStableFunc(rows, func(x, y StygianRankEntry) int {
		return cmp.Compare(x.Seconds+(6-x.Index)*2048, y.Seconds+(6-y.Index)*2048)
	})
	start, end := stygianPeriod(pools, season)
	image := StygianRankImage{Version: stygianVersionName(season), Start: start, End: end, Entries: rows, Ark: arkRows != nil, Akasha: akashaRows != nil}
	view := stygianRankView(image)
	if a.stygianRankImage != nil {
		if drawn, ok := a.stygianRankImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// arkStygianPlaces asks ark for the entries' global places; nil when it does not
// answer, as upstream then leaves ark's column out.
func (a *App) arkStygianPlaces(ctx context.Context, event *rayleabot.EventContext, entries []StygianEntry) []arkStygianPlace {
	uids := []string{}
	for _, entry := range entries {
		uids = append(uids, entry.UID)
	}
	decoded, err := a.arkRequest(ctx, event, "rank/stygian", map[string]any{"uid": uids})
	result := asObject(decoded)
	if err != nil || asText(result["retcode"]) != "100" {
		return nil
	}
	rows := []arkStygianPlace{}
	for _, raw := range asList(result["data"]) {
		item := asObject(raw)
		// ark answered hard for the difficulty when the plugin was written
		// and answers index now.
		rows = append(rows, arkStygianPlace{Rank: cloudNumber(item["rank"]), Sum: cloudNumber(item["sum"]), Index: Int(cmp.Or(item["hard"], item["index"])), Seconds: Int(item["time"])})
	}
	return rows
}

// akashaStygianRows reads akasha.cv's leaderboard of a version for the
// entries, as ark does, and the count of its whole leaderboard: the place of
// its last row. The rows are nil when the service does not answer.
func (c *CloudClient) akashaStygianRows(ctx context.Context, entries []StygianEntry, version string) (map[string]akashaStygianPlace, string) {
	filter := ""
	for _, entry := range entries {
		filter += "[uid]" + entry.UID
	}
	version = strings.ReplaceAll(version, ".", "_")
	var list, last map[string]any
	var group sync.WaitGroup
	group.Go(func() {
		list, _ = c.akasha(ctx, url.Values{"sort": {"stygianScore"}, "order": {"-1"}, "size": {"50"}, "page": {"1"}, "uids": {filter}, "p": {""}, "fromId": {""}, "li": {""}, "uid": {""}, "version": {version}})
	})
	group.Go(func() {
		last, _ = c.akasha(ctx, url.Values{"sort": {"stygianScore"}, "order": {"1"}, "size": {"1"}, "page": {"1"}, "uids": {""}, "p": {""}, "fromId": {""}, "li": {""}, "uid": {""}, "version": {version}})
	})
	group.Wait()
	if list == nil {
		return nil, ""
	}
	rows := map[string]akashaStygianPlace{}
	for _, raw := range asList(list["data"]) {
		item := asObject(raw)
		// ark keeps the rows with a nickname, results, picture and place.
		uid, picture := asText(item["uid"]), asText(item["profilePictureLink"])
		if uid == "" || cloudText(fieldAt(item, "playerInfo.nickname")) == "" || picture == "" || item["stygianIndex"] == nil || item["stygianSeconds"] == nil || cloudNumber(item["index"]) == "" {
			continue
		}
		rows[uid] = akashaStygianPlace{Picture: picture, Index: cloudNumber(item["index"]), Difficulty: Int(item["stygianIndex"]), Seconds: Int(item["stygianSeconds"])}
	}
	return rows, cloudNumber(fieldAt(last, "data.0.index"))
}

// stygianRankView is 幽境危战排名 in text, for replies without images.
func stygianRankView(image StygianRankImage) View {
	v := View{Title: "幽境危战排名", Subtitle: "幽境危战版本 " + image.Version, Rows: []Row{}, Note: "统计周期：" + image.Start.Format("2006-01-02 15:04:05") + " - " + image.End.Format("2006-01-02 15:04:05") + "\n数据源：ark-plugin | akasha.cv"}
	for index, entry := range image.Entries {
		value := "难度 " + strconv.Itoa(entry.Index) + " · " + strconv.Itoa(entry.Seconds) + "秒"
		if image.Ark {
			value += " · 全服排名(ark) " + entry.Ark
		}
		if image.Akasha {
			value += " · 全服排名(akasha) " + entry.Akasha
		}
		v.Rows = append(v.Rows, Row{Label: strconv.Itoa(index+1) + ". " + entry.Name + " " + entry.UID, Value: value})
	}
	return v
}
