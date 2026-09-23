package images

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// trainingLevels is miao's talent colour step for each original level.
var trainingLevels = []int{0, 1, 1, 1, 2, 2, 3, 3, 3, 4, 5}

// trainingArtwork maps the images named in the converted stylesheets (miao's
// common and character/profile-stat, with the page's own background) to
// their paths.
var trainingArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-bg-01", "resources/character/imgs/bg-01.jpg"},
	{"character-imgs-main-01", "resources/character/imgs/main-01.png"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
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
	{"common-item-crown-o", "resources/common/item/crown-o.png"},
	{"common-item-fetter", "resources/common/item/fetter.png"},
}

var (
	talentWeekWord = regexp.MustCompile(`周([1-6]|一|二|三|四|五|六)`)
	rosterStarWord = regexp.MustCompile(`(五|四|5|4|)+星`)
	rosterFiveWord = regexp.MustCompile(`(五|5)+星`)
	rosterElements = []string{"风", "岩", "雷", "草", "水", "火", "冰"}
)

// rosterFilter is miao's star and element filter on the command word
// (ProfileStat.getFilterFunc): a word with 星 keeps the four-stars, or the
// five-stars when 五 or 5 comes before it, and naming elements keeps those.
func rosterFilter(word string) func(star int, element string) bool {
	star := 0
	if rosterStarWord.MatchString(word) {
		star = 4
		if rosterFiveWord.MatchString(word) {
			star = 5
		}
	}
	elements := []string{}
	for _, element := range rosterElements {
		if strings.Contains(word, element) {
			elements = append(elements, element)
		}
	}
	return func(entryStar int, element string) bool {
		return (star == 0 || entryStar == star) && (len(elements) == 0 || slices.Contains(elements, element))
	}
}

// talentBook is a talent book's week group (1 for Monday and Thursday) and
// miao's label for it, city and name, from miao's daily.js; 0 and the name
// alone for a book that table does not list.
func talentBook(book string) (int, string) {
	name := strings.TrimSuffix(strings.TrimPrefix(book, "「"), "」的哲学")
	for week, list := range calendarTalentWeeks {
		if city := slices.Index(list, name); city >= 0 {
			return week + 1, calendarCities[city] + "·" + name
		}
	}
	return 0, name
}

// Training draws 练度统计 the way miao's character/profile-stat does: every
// character in miao's order with level, constellation, friendship, the three
// talents coloured by their original level, the weapon, and the artifact
// sets with the pinned scoring's grade and score. 天赋统计 is the same page's
// talent mode: the weekly boss material and talent book in place of the
// weapon and artifacts, today's books highlighted (the day turns at 04:00),
// with miao's weekday filter from the command word. Both modes keep the
// stars and elements the word names, as 五星列表 and 火角色统计.
func Training(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	list, _ := result.Data["list"].([]any)
	if len(list) == 0 {
		return app.Image{}, false
	}
	talent := strings.Contains(context.Word, "天赋") || strings.Contains(context.Word, "技能")
	resources := &app.ImageResources{Context: context}
	for _, item := range trainingArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	entries, cards := buildRoster(context, resources, list, nil)
	// 剧诗练度统计 keeps who may enter instead.
	keep := rosterFilter(context.Word)
	if theater, ok := context.Input["theater"].(map[string]any); ok {
		entries = theaterRoster(entries, cards, theater)
		keep = func(int, string) bool { return true }
	}
	day := context.Now.In(chinaTime)
	if day.Hour() < 4 {
		day = day.AddDate(0, 0, -1)
	}
	weekday := (int(day.Weekday()) + 6) % 7
	weekFilter := 0
	if talent {
		if match := talentWeekWord.FindStringSubmatch(context.Word); match != nil {
			weekFilter = strings.Index("123456", match[1]) + 1
			if weekFilter == 0 {
				weekFilter = slices.Index([]string{"一", "二", "三", "四", "五", "六"}, match[1]) + 1
			}
		}
		switch {
		case strings.Contains(context.Word, "今日") || strings.Contains(context.Word, "今天"):
			weekFilter = weekday + 1
		case strings.Contains(context.Word, "明日") || strings.Contains(context.Word, "明天"):
			weekFilter = (weekday+1)%7 + 1
		}
	}
	rows := []any{}
	for _, item := range entries {
		entry, _ := context.Catalog.Get(item.id)
		week, label := talentBook(entry.Materials["天赋材料"])
		if !keep(item.star, entry.Element) || (weekFilter != 0 && weekFilter != 7 && week != (weekFilter-1)%3+1) {
			continue
		}
		card := item.card
		fetter := item.fetter
		if traveler(item.id) {
			fetter = 10
		}
		talents := []any{}
		levels, _ := card["talents"].([]any)
		for index := range 3 {
			level, original := any("-"), 1
			if index < len(levels) {
				talent := levels[index].(map[string]any)
				level, original = talent["level"], app.Int(talent["original"])
			}
			talents = append(talents, map[string]any{"level": level, "class": trainingLevels[min(max(original, 0), 10)],
				"plus": app.Int(level) > original})
		}
		row := map[string]any{"no": len(rows) + 1, "star": item.star, "face": card["face"], "name": card["abbr"], "level": item.level, "cons": item.cons, "fetter": fetter, "talents": talents}
		if weapon, _ := card["weapon"].(map[string]any); weapon != nil && item.panel != nil && item.panel.Weapon != nil {
			// miao shortens names longer than four characters.
			name := item.panel.Weapon.Name
			if entry, ok := context.Catalog.Get(item.panel.Weapon.ID); ok {
				name = entry.Name
				if utf8.RuneCountInString(name) > 4 && entry.Abbr != "" {
					name = entry.Abbr
				}
			}
			row["weapon"] = map[string]any{"star": weapon["star"], "level": weapon["level"], "icon": weapon["icon"], "affix": weapon["affix"],
				"badge": app.Int(weapon["affix"]) + 1, "name": name}
		}
		row["artis"] = card["artis"]
		if talent {
			weekText := "-"
			if week > 0 {
				weekText = []string{"1/4", "2/5", "3/6"}[week-1]
			}
			if label == "" {
				label = "-"
			}
			row["book"] = map[string]any{"today": weekday == 6 || weekday%3+1 == week, "label": label, "week": weekText,
				"icon":   resources.Artwork("book-"+strconv.Itoa(len(resources.List)), "miao-plugin", "resources/meta-gs/material/talent/"+entry.Materials["天赋材料"]+".webp"),
				"weekly": resources.Artwork("weekly-"+strconv.Itoa(len(resources.List)), "miao-plugin", "resources/meta-gs/material/weekly/"+entry.Materials["周本材料"]+".webp")}
			rows = append(rows, row)
			continue
		}
		// Characters with artifacts get the pinned scoring's grade and score.
		if item.panel != nil && len(item.panel.Equipment) > 0 && context.Score != nil {
			if scored, err := context.Score(*item.panel); err == nil && scored.ScoreDetail != nil {
				row["mark"], row["grade"] = scored.ScoreDetail.Mark, scored.ScoreDetail.Grade
			}
		}
		rows = append(rows, row)
	}
	return app.Image{Template: "training", Data: map[string]any{
		"uid": result.Role.UID, "count": len(rows), "rows": rows, "updated": context.Now.In(chinaTime).Format("2006-01-02 15:04"), "talent": talent,
	}, Resources: resources.List}, true
}

// theaterRoster is miao's 剧诗练度统计 roster: the month's opening characters
// join at level 90 with every talent at 8 unless the player's own is at
// least that level, the roster is ordered by element and then as usual,
// highest first, and only characters of level 70 or more that are invited or
// of an allowed element remain, the Manekin twins aside.
func theaterRoster(entries []rosterEntry, cards *avatarCards, theater map[string]any) []rosterEntry {
	list := func(key string) []string {
		values, _ := theater[key].([]string)
		return values
	}
	initial, invite, elements := list("initial"), list("invite"), list("elements")
	for _, id := range initial {
		index := slices.IndexFunc(entries, func(entry rosterEntry) bool { return entry.id == id })
		if index >= 0 && entries[index].level >= 90 {
			continue
		}
		card := cards.guest(id, 90, 0)
		if card["known"] != true {
			continue
		}
		talents := []any{}
		for _, key := range []string{"a", "e", "q"} {
			talents = append(talents, map[string]any{"key": key, "level": 8, "original": 8})
		}
		card["talents"], card["type"] = talents, "mini"
		start := rosterEntry{id: id, card: card, level: 90, star: app.Int(card["star"]), aeq: 24}
		if index >= 0 {
			entries[index] = start
		} else {
			entries = append(entries, start)
		}
	}
	element := func(entry rosterEntry) string { return app.Text(entry.card["elem"]) }
	// miao sorts ascending and reverses, so equal characters swap order too.
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if element(a) != element(b) {
			return element(a) < element(b)
		}
		for _, pair := range [][2]int{{a.level, b.level}, {a.star, b.star}, {a.aeq, b.aeq}, {a.cons, b.cons}, {a.weaponLevel, b.weaponLevel}, {a.weaponStar, b.weaponStar}, {a.refinement, b.refinement}, {a.fetter, b.fetter}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
	slices.Reverse(entries)
	kept := []rosterEntry{}
	for _, entry := range entries {
		if entry.level >= 70 && entry.id != "10000117" && entry.id != "10000118" && (slices.Contains(invite, entry.id) || slices.Contains(elements, element(entry))) {
			kept = append(kept, entry)
		}
	}
	return kept
}
