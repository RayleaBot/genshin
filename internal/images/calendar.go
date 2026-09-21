package images

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// Miao's calendar skips these announcements by ID and by title.
var (
	calendarIgnoreIDs = []string{"495", "1263", "423", "422", "762"}
	calendarIgnore    = regexp.MustCompile(`(内容专题页|版本更新说明|调研|防沉迷|米游社|专项意见|更新修复与优化|问卷调查|版本更新通知|更新时间说明|预下载功能|周边限时|周边上新|角色演示)`)
	calendarFulltime  = regexp.MustCompile(`(魔神任务)`)
)

// The patterns miao reads announcement times with.
var (
	calendarVersionTitle = regexp.MustCompile(`(\d\.\d|「[^」]+」)版本更新(通知|说明|维护预告)`)
	calendarUpdateTime   = regexp.MustCompile(`(?:更新时间)\s*〓([^〓]+)(?:〓|$)`)
	calendarTimeText     = regexp.MustCompile(`([0-9/: ]){9,}`)
	calendarWillStart    = regexp.MustCompile(`(将于)(([0-9/: ]){9,})(进行)`)
	calendarTags         = regexp.MustCompile(`(<|&lt;)[\w "%:;=\-/(),.]+(>|&gt;)|(&nbsp;)`)
	calendarSection      = regexp.MustCompile(`(?:活动时间|祈愿介绍|任务开放时间|冒险....包|折扣时间)\s*〓([^〓]+)(〓|$)`)
	calendarRange        = regexp.MustCompile(`(?:活动时间)?(?:〓|\s)*([0-9/: ~]{6,})`)
	calendarAfterVersion = regexp.MustCompile(`(?:\d\.\d|「[^」]+」)版本更新(?:完成)?后`)
	calendarVersionNum   = regexp.MustCompile(`(\d\.\d)版本更新(?:完成)?后`)
	calendarDuring       = regexp.MustCompile(`(\d\.\d|「[^」]+」)版本期间持续开放`)
	calendarWeaponBanner = regexp.MustCompile(`(单手剑|双手剑|长柄武器|弓|法器)·`)
	calendarCharacter    = regexp.MustCompile(`·(.*)\(`)
)

// calendarArtwork maps the images the calendar page names to miao's paths;
// the page is drawn on miao's default hydro background.
var calendarArtwork = [][2]string{
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
	{"common-bg-talent-hydro", "resources/common/bg/talent-hydro.webp"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"wiki-imgs-abyss", "resources/wiki/imgs/abyss.jpg"},
	{"wiki-imgs-abyss-1", "resources/wiki/imgs/abyss-1.jpg"},
	{"wiki-imgs-abyss-2", "resources/wiki/imgs/abyss-2.jpg"},
	{"wiki-imgs-abyss-4", "resources/wiki/imgs/abyss-4.jpg"},
	{"wiki-imgs-newAbyss", "resources/wiki/imgs/newAbyss.jpg"},
	{"abyss-icon", "resources/wiki/imgs/abyss-icon.png"},
	{"newAbyss-icon", "resources/wiki/imgs/newAbyss-icon.png"},
	{"calendar-icon", "resources/wiki/imgs/calendar-icon.png"},
}

// calendarElements are miao's element codes, which class a character banner.
var calendarElements = map[string]string{"火": "pyro", "水": "hydro", "风": "anemo", "雷": "electro", "草": "dendro", "冰": "cryo", "岩": "geo"}

// calendarTalentWeeks is miao's daily.js: the talent books by weekday group
// (Monday and Thursday first) and by city, in miao's city order; the books
// run in the order miao's material data lists them.
var (
	calendarTalentWeeks = [3][]string{
		{"自由", "繁荣", "浮世", "诤言", "公平", "角逐", "月光", "慈爱"},
		{"抗争", "勤劳", "风雅", "巧思", "正义", "焚燔", "乐园", "坚忍"},
		{"诗文", "黄金", "天光", "笃行", "秩序", "纷争", "浪迹", "荣光"},
	}
	calendarCities      = []string{"蒙德", "璃月", "稻妻", "须弥", "枫丹", "纳塔", "挪德卡莱", "至冬"}
	calendarTalentOrder = []string{"诗文", "浮世", "勤劳", "风雅", "自由", "天光", "黄金", "繁荣", "抗争", "诤言", "巧思", "笃行", "公平", "正义", "秩序",
		"角逐", "焚燔", "纷争", "月光", "乐园", "浪迹", "慈爱", "坚忍"}
	calendarMonths   = []string{"一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}
	calendarWeekdays = []string{"一", "二", "三", "四", "五", "六", "日"}
)

// calendarItem is an announcement placed on the calendar.
type calendarItem struct {
	sort                  int
	id, title, kind       string
	merge                 int
	banner, icon          string
	left, width           float64
	label                 string
	duration              time.Duration
	start                 string
	character, face, elem string
	card                  string
	index                 int
}

// Calendar draws 日历 the way miao's wiki/calendar does: the thirteen days
// from six days ago with the characters whose birthday falls on each, the
// events, banners and notices of the announcements as bars across those days
// (two that do not overlap share a row), the Spiral Abyss and Imaginarium
// Theater periods, the current time, and the characters whose talent books
// are farmable today with their weekly boss material. 日历列表 draws the list
// layout. Miao also reads corrected times from its own HTTP service; this
// plugin reads only the official HTTPS announcements, so those corrections
// are not applied.
func Calendar(context gamekit.ImageContext, calendar gamekit.CalendarImage) (gamekit.Image, bool) {
	if len(calendar.Announcements.Groups) < 2 {
		return gamekit.Image{}, false
	}
	now := context.Now.In(chinaTime)
	times := calendarTimes(calendar.Announcements.Contents)
	first := time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, chinaTime)
	last := time.Date(now.Year(), now.Month(), now.Day()+6, 23, 59, 59, 0, chinaTime)
	total := float64(last.Sub(first))
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range calendarArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	// The banners and icons download together on the first draw.
	urls := []any{}
	for _, group := range calendar.Announcements.Groups {
		for _, ann := range calendarGroup(group) {
			urls = append(urls, ann["banner"], ann["tag_icon"])
		}
	}
	resources.Prefetch("mihoyo", urls...)
	place := func(ann map[string]any, target *[]calendarItem, act bool, kind string) {
		id, title := gamekit.Text(ann["ann_id"]), gamekit.Text(ann["title"])
		if slices.Contains(calendarIgnoreIDs, id) || calendarIgnore.MatchString(title) {
			return
		}
		item := calendarItem{id: id, title: title, kind: kind, sort: 10}
		if item.kind == "" {
			item.kind = "normal"
			if act {
				item.kind = "activity"
			}
		}
		if act {
			item.sort = 5
		}
		switch {
		case strings.Contains(title, "概率UP"):
			switch {
			case calendarWeaponBanner.MatchString(title):
				item.kind, item.title, item.sort = "weapon", calendarWeaponBanner.ReplaceAllString(title, ""), 3
			case strings.Contains(title, "集录"):
				item.sort = 2
			case strings.Contains(title, "祈愿"):
				item.kind = "character"
				if match := calendarCharacter.FindStringSubmatch(title); match != nil && match[1] != "" {
					if entry, ok := context.Catalog.Resolve(match[1], "character", nil); ok {
						item.card = resources.Artwork("card-"+entry.ID, "miao-plugin", "resources/meta-gs/character/"+entry.Name+"/imgs/card.webp")
						item.face = resources.Artwork("face-"+entry.ID, "miao-plugin", "resources/meta-gs/character/"+entry.Name+"/imgs/face.webp")
						item.elem = calendarElements[entry.Element]
					}
					item.character, item.sort = match[1], 1
				}
			}
		case strings.Contains(title, "纪行"):
			item.kind = "pass"
		}
		window := times[id]
		sDate := calendarDate(window[0], gamekit.Text(ann["start_time"]))
		eDate := calendarDate(window[1], gamekit.Text(ann["end_time"]))
		sTime, eTime := sDate, eDate
		if sTime.Before(first) {
			sTime = first
		}
		if eTime.After(last) {
			eTime = last
		}
		item.left = float64(sTime.Sub(first)) / total * 100
		item.width = float64(eTime.Sub(first))/total*100 - item.left
		switch {
		case calendarFulltime.MatchString(title) || eDate.Sub(sDate) > 365*24*time.Hour:
			item.label = "永久有效"
			if sDate.Before(now) {
				item.label = sDate.Format("01-02 15:04") + " 后永久有效"
			}
		case now.After(sDate) && eDate.After(now):
			item.label = eDate.Format("01-02 15:04") + " (" + humanize(eDate.Sub(now)) + "后结束)"
			if limit := map[bool]float64{true: 38, false: 55}[act]; item.width > limit {
				item.label = sDate.Format("01-02 15:04") + " ~ " + item.label
			}
		case sDate.After(now):
			item.label = sDate.Format("01-02 15:04") + " (" + humanize(sDate.Sub(now)) + "后开始)"
		case act:
			item.label = sDate.Format("01-02 15:04") + " ~ " + eDate.Format("01-02 15:04")
		}
		if sDate.After(last) || eDate.Before(first) {
			return
		}
		if act && gamekit.Text(ann["banner"]) != "" {
			item.banner = resources.URL("mihoyo", ann["banner"])
		}
		if gamekit.Text(ann["tag_icon"]) != "" {
			item.icon = resources.URL("mihoyo", ann["tag_icon"])
		}
		if item.kind == "activity" || item.kind == "normal" {
			item.merge = 1
		}
		item.duration, item.start = eTime.Sub(sTime), sDate.Format("01-02 15:04")
		*target = append(*target, item)
	}
	list, abyss := []calendarItem{}, []calendarItem{}
	for _, raw := range calendarGroup(calendar.Announcements.Groups[1]) {
		place(raw, &list, true, "")
	}
	for _, raw := range calendarGroup(calendar.Announcements.Groups[0]) {
		place(raw, &list, false, "")
	}
	for _, period := range calendarAbyss(now, first, last) {
		kind := "abyss"
		if strings.Contains(period.title, "「幻想真境剧诗」") {
			kind = "newAbyss"
		}
		place(map[string]any{"title": period.title, "start_time": period.start.Format("2006-01-02 15:04"), "end_time": period.end.Format("2006-01-02 15:04")}, &abyss, true, kind)
	}
	// Miao sorts by kind, then start as text, then duration.
	slices.SortStableFunc(list, func(a, b calendarItem) int {
		if a.sort != b.sort {
			return a.sort - b.sort
		}
		if a.start != b.start {
			return strings.Compare(a.start, b.start)
		}
		return int(a.duration - b.duration)
	})
	characterCount, characterOld, weaponCount := 0, 0, 0
	rows := []any{}
	for index := range list {
		item := &list[index]
		switch item.kind {
		case "character":
			characterCount++
			if item.left == 0 {
				characterOld++
			}
			item.index = characterCount
		case "weapon":
			weaponCount++
			item.index = weaponCount
		}
		if item.merge == 1 {
			for other := range list {
				if list[other].merge == 1 && item.left+item.width <= list[other].left {
					item.merge, list[other].merge = 2, 2
					rows = append(rows, []any{calendarView(*item, 0), calendarView(list[other], 1)})
					break
				}
			}
		}
		if item.merge != 2 {
			item.merge = 2
			rows = append(rows, []any{calendarView(*item, 0)})
		}
	}
	abyssViews := []any{}
	for _, item := range abyss {
		abyssViews = append(abyssViews, calendarView(item, 0))
	}
	months, births, most := calendarDays(context, resources, now, first)
	// The list layout is narrower, so it has its own template width.
	mode, template := "calendar", "calendar"
	if strings.HasSuffix(calendar.Word, "日历列表") || strings.HasSuffix(calendar.Word, "活动") {
		mode, template = "list", "calendar-list"
	}
	return gamekit.Image{Template: template, Data: map[string]any{
		"mode": mode, "months": months, "births": births, "list_class": fmt.Sprintf("char-%d-%d char-num-%d", characterCount, characterOld, most),
		"rows": rows, "abyss": abyssViews, "now_left": fmt.Sprintf("%.4f", float64(now.Sub(first))/total*100),
		"now_time": now.Format("2006-01-02 15:04"), "talents": calendarTalents(context, resources, now),
	}, Resources: resources.List}, true
}

// calendarView is what the page shows of an item; column is its place in a
// shared row.
func calendarView(item calendarItem, column int) map[string]any {
	classes := []string{"type-" + item.kind}
	if item.index > 0 {
		classes = append(classes, "li-idx-"+strconv.Itoa(item.index))
	}
	if item.elem != "" {
		classes = append(classes, "elem-"+item.elem)
	}
	if item.width < 20 {
		classes = append(classes, "small-mode")
	}
	classes = append(classes, "li-col"+strconv.Itoa(column))
	return map[string]any{"class": strings.Join(classes, " "), "kind": item.kind, "left": fmt.Sprintf("%.4f", item.left), "width": fmt.Sprintf("%.4f", item.width),
		"title": item.title, "label": item.label, "banner": item.banner, "card": item.card, "face": item.face, "icon": item.icon, "character": item.kind == "character"}
}

// calendarGroup is one group of the announcement list.
func calendarGroup(raw any) []map[string]any {
	group, _ := raw.(map[string]any)
	list, _ := group["list"].([]any)
	out := []map[string]any{}
	for _, item := range list {
		if ann, ok := item.(map[string]any); ok {
			out = append(out, ann)
		}
	}
	return out
}

// calendarTimes reads each announcement's window from its body the way
// miao's reqCalData does: the version update times first, then each
// announcement's event, banner or mission period, which may run from a
// version update, through a version, or for good.
func calendarTimes(contents []any) map[string][2]string {
	versions := map[string]string{}
	for _, raw := range contents {
		ann, _ := raw.(map[string]any)
		title, content := gamekit.Text(ann["title"]), gamekit.Text(ann["content"])
		match := calendarVersionTitle.FindStringSubmatch(title)
		if match == nil {
			continue
		}
		if section := calendarUpdateTime.FindStringSubmatch(content); section != nil {
			if found := calendarTimeText.FindString(section[1]); found != "" && versions[match[1]] == "" {
				versions[match[1]] = strings.Replace(found, "06:00", "11:00", 1)
			}
		} else if found := calendarWillStart.FindStringSubmatch(calendarTags.ReplaceAllString(content, "")); found != nil && versions[match[1]] == "" {
			versions[match[1]] = strings.Replace(found[2], "06:00", "11:00", 1)
		}
	}
	times := map[string][2]string{}
	for _, raw := range contents {
		ann, _ := raw.(map[string]any)
		title := gamekit.Text(ann["title"])
		if calendarIgnore.MatchString(title) {
			continue
		}
		section := calendarSection.FindStringSubmatch(calendarTags.ReplaceAllString(gamekit.Text(ann["content"]), ""))
		if section == nil || section[1] == "" {
			continue
		}
		content := section[1]
		var span []string
		match := calendarRange.FindStringSubmatch(content)
		if match != nil {
			span = strings.Split(match[1], "~")
		}
		switch {
		case calendarAfterVersion.MatchString(content):
			version := ""
			if found := calendarVersionNum.FindStringSubmatch(content); found != nil {
				version = versions[found[1]]
			}
			if version == "" {
				continue
			}
			if strings.Contains(content, "永久开放") {
				span = []string{version, "2099/01/01 00:00:00"}
			} else if found := calendarTimeText.FindString(content); found != "" {
				span = []string{version, found}
			}
		case calendarDuring.MatchString(content):
			version := versions[calendarDuring.FindStringSubmatch(content)[1]]
			if version == "" {
				continue
			}
			start := calendarDate(strings.ReplaceAll(strings.TrimSpace(version), "/", "-"), "")
			end := time.Date(start.Year(), start.Month(), start.Day()+42, 6, start.Minute(), start.Second(), 0, chinaTime)
			span = []string{version, end.Format("2006-01-02 15:04:05")}
		case strings.Contains(content, "后永久开放") && match != nil:
			span = []string{match[1], "2099/01/01 00:00:00"}
		}
		if len(span) == 2 {
			times[gamekit.Text(ann["ann_id"])] = [2]string{strings.ReplaceAll(strings.TrimSpace(span[0]), "/", "-"), strings.ReplaceAll(strings.TrimSpace(span[1]), "/", "-")}
		}
	}
	return times
}

// calendarDate is miao's getDate: the body's time when it is longer than six
// characters, otherwise the list's.
func calendarDate(body, list string) time.Time {
	text := list
	if len(body) > 6 {
		text = body
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-1-2 15:04:05", "2006-1-2 15:04", "2006-01-02"} {
		if at, err := time.ParseInLocation(layout, strings.TrimSpace(text), chinaTime); err == nil {
			return at
		}
	}
	return time.Time{}
}

// humanize is moment's duration.humanize() in the zh-cn locale.
func humanize(span time.Duration) string {
	ms := math.Abs(float64(span.Milliseconds()))
	seconds, minutes, hours := math.Round(ms/1000), math.Round(ms/6e4), math.Round(ms/36e5)
	days := math.Round(ms / 864e5)
	months := math.Round(ms / 864e5 * 4800 / 146097)
	years := math.Round(ms / 864e5 * 4800 / 146097 / 12)
	switch {
	case seconds <= 44:
		return "几秒"
	case seconds < 45:
		return fmt.Sprintf("%.0f 秒", seconds)
	case minutes <= 1:
		return "1 分钟"
	case minutes < 45:
		return fmt.Sprintf("%.0f 分钟", minutes)
	case hours <= 1:
		return "1 小时"
	case hours < 22:
		return fmt.Sprintf("%.0f 小时", hours)
	case days <= 1:
		return "1 天"
	case days < 26:
		return fmt.Sprintf("%.0f 天", days)
	case months <= 1:
		return "1 个月"
	case months < 11:
		return fmt.Sprintf("%.0f 个月", months)
	case years <= 1:
		return "1 年"
	}
	return fmt.Sprintf("%.0f 年", years)
}

type calendarPeriod struct {
	start, end time.Time
	title      string
}

// calendarAbyss is miao's getAbyssCal: last and this month's Imaginarium
// Theater (from the 1st) and Spiral Abyss (from the 16th) that begin or end
// within the window.
func calendarAbyss(now, first, last time.Time) []calendarPeriod {
	month := func(offset, day int) time.Time {
		return time.Date(now.Year(), now.Month()+time.Month(offset), day, 4, 0, 0, 0, chinaTime)
	}
	name := func(offset int) string { return calendarMonths[int(month(offset, 1).Month())-1] }
	periods := []calendarPeriod{
		{month(-1, 1), month(0, 1).Add(-time.Second), "「幻想真境剧诗」· " + name(-1)},
		{month(0, 1), month(1, 1).Add(-time.Second), "「幻想真境剧诗」· " + name(0)},
		{month(-1, 16), month(0, 16).Add(-time.Second), "「深境螺旋」· " + name(-1)},
		{month(0, 16), month(1, 16).Add(-time.Second), "「深境螺旋」· " + name(0)},
	}
	out := []calendarPeriod{}
	for _, period := range periods {
		if (!period.start.After(first) && !first.After(period.end)) || (!period.start.After(last) && !last.After(period.end)) {
			out = append(out, period)
		}
	}
	return out
}

// calendarDays is the day header and each day's birthdays: the days grouped
// by month, the characters born on each day, and the most born on one day.
func calendarDays(context gamekit.ImageContext, resources *gamekit.ImageResources, now, first time.Time) ([]any, map[string]any, int) {
	born := map[string][]gamekit.Entry{}
	entries := map[string]gamekit.Entry{}
	for _, entry := range context.Catalog.Entries {
		if entry.Kind == "character" {
			entries[entry.ID] = entry
		}
	}
	for _, birthday := range context.Game.Data.Resources.Birthdays {
		month, day, _ := strings.Cut(birthday.Date, "-")
		m, _ := strconv.Atoi(month)
		d, _ := strconv.Atoi(day)
		if entry, ok := entries[birthday.ID]; ok {
			key := fmt.Sprintf("%d-%d", m, d)
			born[key] = append(born[key], entry)
		}
	}
	months, births, most := []any{}, map[string]any{}, 0
	var dates []any
	for index := range 13 {
		day := first.AddDate(0, 0, index)
		if index > 0 && day.Month() != day.AddDate(0, 0, -1).Month() {
			months = append(months, map[string]any{"month": int(day.AddDate(0, 0, -1).Month()), "dates": dates})
			dates = nil
		}
		key := fmt.Sprintf("%d-%d", day.Month(), day.Day())
		characters := []any{}
		list := born[key]
		// Miao lists characters by ID.
		slices.SortFunc(list, func(a, b gamekit.Entry) int {
			return strings.Compare(fmt.Sprintf("%012s", a.ID), fmt.Sprintf("%012s", b.ID))
		})
		for _, entry := range list {
			name := entry.Abbr
			if name == "" {
				name = entry.Name
			}
			if len([]rune(name)) < 4 {
				name += "生日"
			}
			characters = append(characters, map[string]any{"star": entry.Rarity, "name": name,
				"face": resources.Artwork("face-"+entry.ID, "miao-plugin", "resources/meta-gs/character/"+entry.Name+"/imgs/face.webp")})
		}
		most = max(most, len(characters))
		births[key] = characters
		dates = append(dates, map[string]any{"date": day.Day(), "week": calendarWeekdays[(int(day.Weekday())+6)%7], "key": key, "current": day.Day() == now.Day()})
	}
	months = append(months, map[string]any{"month": int(first.AddDate(0, 0, 12).Month()), "dates": dates})
	return months, births, most
}

// calendarTalents are miao's getCharData talent books: those farmable today
// (the day turns at 04:00; Sunday has them all), by city, each with its
// characters, rarest and newest first, and their weekly boss material.
func calendarTalents(context gamekit.ImageContext, resources *gamekit.ImageResources, now time.Time) []any {
	day := now
	if day.Hour() < 4 {
		day = day.AddDate(0, 0, -1)
	}
	weekday := (int(day.Weekday()) + 6) % 7
	type book struct {
		abbr, city string
		cid        int
		chars      []gamekit.Entry
	}
	books := []*book{}
	byName := map[string]*book{}
	for _, abbr := range calendarTalentOrder {
		for week, list := range calendarTalentWeeks {
			cid := slices.Index(list, abbr)
			if cid < 0 || (weekday != 6 && week != weekday%3) {
				continue
			}
			entry := &book{abbr: abbr, city: calendarCities[cid], cid: cid + 1}
			books = append(books, entry)
			byName["「"+abbr+"」的哲学"] = entry
		}
	}
	for _, entry := range context.Catalog.Entries {
		if entry.Kind != "character" || entry.ID == "10000005" || entry.ID == "10000007" || strings.HasPrefix(entry.Name, "旅行者") {
			continue
		}
		if target := byName[entry.Materials["天赋材料"]]; target != nil {
			target.chars = append(target.chars, entry)
		}
	}
	slices.SortStableFunc(books, func(a, b *book) int { return a.cid - b.cid })
	out := []any{}
	for _, talent := range books {
		slices.SortStableFunc(talent.chars, func(a, b gamekit.Entry) int {
			if a.Rarity != b.Rarity {
				return b.Rarity - a.Rarity
			}
			return strings.Compare(fmt.Sprintf("%012s", b.ID), fmt.Sprintf("%012s", a.ID))
		})
		chars := []any{}
		for index, entry := range talent.chars {
			position := ""
			switch {
			case index == 0:
				position = "first"
			case index == len(talent.chars)-1:
				position = "last"
			}
			chars = append(chars, map[string]any{"star": entry.Rarity, "position": position, "first": index == 0,
				"face":   resources.Artwork("face-"+entry.ID, "miao-plugin", "resources/meta-gs/character/"+entry.Name+"/imgs/face.webp"),
				"weekly": resources.Artwork("weekly-"+strconv.Itoa(len(resources.List)), "miao-plugin", "resources/meta-gs/material/weekly/"+entry.Materials["周本材料"]+".webp")})
		}
		out = append(out, map[string]any{"cid": talent.cid, "label": talent.city + "·" + talent.abbr, "chars": chars,
			"icon": resources.Artwork("talent-"+strconv.Itoa(talent.cid)+"-"+strconv.Itoa(len(resources.List)), "miao-plugin", "resources/meta-gs/material/talent/「"+talent.abbr+"」的哲学.webp")})
	}
	return out
}
