package images

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

// miaoGachaArtwork are the fonts and backgrounds of miao's gacha-detail and
// gacha-stat pages.
var miaoGachaArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"gacha-imgs-date-icon", "resources/gacha/imgs/date-icon.webp"},
}

// miaoDetailWords and miaoStatWords are what Gacha.detail and Gacha.stat
// strip before reading the pool; detail keeps 喵喵 there, which sends every
// 喵喵 word to the character wish, so it is stripped here too.
var (
	miaoDetailWords = regexp.MustCompile(`#|抽卡|记录|祈愿|分析|池|喵喵`)
	miaoStatWords   = regexp.MustCompile(`#|统计|分析|池`)
)

// miaoVersion is a period of miao's GachaData: a wish half with its rate-up
// five-stars. The 新版本 after the last known period has no end.
type miaoVersion struct {
	version, half, from, to string
	char5, weapon5          []string
}

// before is whether a pull is older than the period, so the next period has
// to be looked up; a period without a start is looked up again each pull.
func (v miaoVersion) before(at string) bool { return v.from == "" || at < v.from }

// miaoPeriods are miao's poolVersion and mixPoolVersion from the banner data.
type miaoPeriods struct{ event, mix []miaoVersion }

func newMiaoPeriods(context app.ImageContext) miaoPeriods {
	list := func(kind string) []miaoVersion {
		versions := []miaoVersion{}
		for _, pool := range context.Game.Data.Resources.Pools {
			if pool.Kind == kind {
				versions = append(versions, miaoVersion{pool.Version, pool.Half, pool.From, pool.To, pool.Characters5, pool.Weapons5})
			}
		}
		sort.SliceStable(versions, func(i, j int) bool { return versions[i].from < versions[j].from })
		// Upstream ends 新版本 on a fixed date its data has since passed.
		if len(versions) > 0 {
			versions = append(versions, miaoVersion{version: "新版本", half: "?", from: versions[len(versions)-1].to})
		}
		return versions
	}
	return miaoPeriods{list("event"), list("chronicled")}
}

// at is miao's getVersion: the chronicled periods first for the chronicled
// wish, then the event ones; a pull no period holds is 未知, or 全部 where
// periods are not told apart.
func (p miaoPeriods) at(at string, versioned, mix bool) miaoVersion {
	lists := [][]miaoVersion{}
	if mix {
		lists = append(lists, p.mix)
	}
	if versioned {
		lists = append(lists, p.event)
	}
	for _, list := range lists {
		for _, version := range list {
			if at > version.from && (version.to == "" || at < version.to) {
				return version
			}
		}
	}
	if !versioned {
		return miaoVersion{version: "全部"}
	}
	return miaoVersion{version: "未知"}
}

// miaoItem is an item of miao's readJSON: the character or weapon pulled.
type miaoItem struct {
	id, name, abbr, img string
	weapon              bool
	star                int
}

// miaoPull is a pull of an item at a time.
type miaoPull struct {
	item *miaoItem
	at   string
}

// miaoPulls are miao's readJSON over the pools: each record once, newest
// first, with its item. Names the catalog does not know are 未知 as upstream's
// 403 and 404, kept apart by the record's star, which upstream fixes at four
// for a character and three for a weapon.
func miaoPulls(context app.ImageContext, resources *app.ImageResources, archive gacha.Archive, pools ...string) []miaoPull {
	records := []gacha.Record{}
	seen := map[string]bool{}
	for _, record := range archive.Records {
		if slices.Contains(pools, gacha.Pool(record.GachaType)) && !seen[record.ID] {
			seen[record.ID] = true
			records = append(records, record)
		}
	}
	// Newest first, as upstream sorts the pools it reads together; record
	// IDs order the pulls of one time.
	sort.Slice(records, func(i, j int) bool {
		if records[i].Time != records[j].Time {
			return records[i].Time > records[j].Time
		}
		if len(records[i].ID) != len(records[j].ID) {
			return len(records[i].ID) > len(records[j].ID)
		}
		return records[i].ID > records[j].ID
	})
	types := map[string]string{}
	if context.Game.Calc != nil {
		for _, weapon := range context.Game.Calc.Metadata().Weapons {
			types[weapon.ID] = weapon.Type
		}
	}
	items := map[string]*miaoItem{}
	pulls := []miaoPull{}
	for _, record := range records {
		weapon := record.ItemType == "武器" || record.ItemType == "光锥"
		if !weapon && record.ItemType != "角色" {
			continue
		}
		kind := "character"
		if weapon {
			kind = "weapon"
		}
		item := items[kind+"/"+record.Name]
		if item == nil {
			item = &miaoItem{name: "未知", abbr: "未知", weapon: weapon}
			if entry, ok := context.Catalog.Resolve(record.Name, kind, nil); ok && entry.Name == record.Name {
				item.id, item.name, item.abbr, item.star = entry.ID, entry.Name, entry.Name, entry.Rarity
				// miao's Weapon.abbr keeps a name of up to four characters.
				if entry.Abbr != "" && (!weapon || utf8.RuneCountInString(entry.Name) > 4) {
					item.abbr = entry.Abbr
				}
				path := "resources/meta-gs/weapon/" + types[entry.ID] + "/" + entry.Name + "/icon.webp"
				if !weapon {
					portraits, _ := app.CharacterFolders(entry.ID, entry.Name, "")
					path = portraits + "imgs/face.webp"
				}
				item.img = resources.Artwork(kind+"-"+entry.ID, "miao-plugin", path)
			} else {
				item.star, _ = strconv.Atoi(record.Rank)
			}
			items[kind+"/"+record.Name] = item
		}
		pulls = append(pulls, miaoPull{item, record.Time})
	}
	return pulls
}

// miaoPlayer is miao's getFace: the banner and face of the player's
// profile-picture character, else 芭芭拉's, and the name kept for the UID.
func miaoPlayer(context app.ImageContext, resources *app.ImageResources, image app.GachaImage) map[string]any {
	entry, ok := context.Catalog.Get(image.Face)
	if !ok || entry.Kind != "character" {
		entry, _ = context.Catalog.Get("10000014")
	}
	portraits, _ := app.CharacterFolders(entry.ID, entry.Name, "")
	face := resources.Artwork("player-face", "miao-plugin", portraits+"imgs/face-q.webp")
	if face == "" {
		face = resources.Artwork("player-face", "miao-plugin", portraits+"imgs/face.webp")
	}
	name := image.Nickname
	if name == "" {
		name = "旅行者"
	}
	return map[string]any{"elem": "hydro", "uid": image.UID, "name": name, "face": face, "banner": resources.Artwork("player-banner", "miao-plugin", portraits+"imgs/banner.webp"), "mark": context.Game.Prefix}
}

// miaoStat is a figure in miao's header or a period's summary.
func miaoStat(stats []any, value any, title string) []any {
	return append(stats, map[string]any{"value": value, "title": title})
}

// GachaDetail draws 喵喵抽卡记录 the way miao's GachaData.analyse and
// gacha/gacha-detail do: the pulls, five-stars, the rate-ups missed, average
// pulls per five-star and the primogems per rate-up, then every five-star,
// newest first, with its date, whether it was the rate-up and the pulls it
// took, after the pulls made since the last one.
func GachaDetail(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	pool := "301"
	switch miaoDetailWords.ReplaceAllString(image.Word, "") {
	case "常驻":
		pool = "200"
	case "武器":
		pool = "302"
	case "集录":
		pool = "500"
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range miaoGachaArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	pulls := miaoPulls(context, resources, image.Archive, pool)
	if len(pulls) == 0 {
		return app.Image{}, false
	}
	periods := newMiaoPeriods(context)
	type five struct {
		up, drawn       bool
		date, name, img string
		count           int
	}
	fives := []five{}
	var current miaoVersion
	all, fiveNum, fourNum, sinceFive, noFive, missed := 0, 0, 0, 0, 0, 0
	for index, pull := range pulls {
		if index == 0 || current.before(pull.at) {
			current = periods.at(pull.at, true, pool == "500")
		}
		all++
		if pull.item.star == 4 {
			fourNum++
		}
		if pull.item.star == 5 {
			fiveNum++
			if len(fives) > 0 {
				fives[len(fives)-1].count = sinceFive
			} else {
				noFive = sinceFive
			}
			sinceFive = 0
			up := slices.Contains(current.char5, pull.item.name)
			if pull.item.weapon {
				up = slices.Contains(current.weapon5, pull.item.name)
			}
			if !up {
				missed++
			}
			fives = append(fives, five{up: up, date: pull.at[5:10], name: pull.item.abbr, img: pull.item.img})
		}
		sinceFive++
	}
	if len(fives) > 0 {
		fives[len(fives)-1].count = sinceFive
	} else {
		noFive = all
	}
	stats := miaoStat(nil, all, "抽卡总数")
	if fiveNum > 0 {
		stats = miaoStat(stats, fiveNum, "金卡数")
	}
	if missed > 0 {
		stats = miaoStat(stats, missed, "歪 T.T")
	}
	if fourNum > 0 {
		stats = miaoStat(stats, fourNum, "紫卡数")
	}
	if fiveNum > 0 {
		stats = miaoStat(stats, jsFixed(float64(all-noFive)/float64(fiveNum), 2), "平均出金")
	}
	// Pulls per rate-up, leaving out the newest five-star when it missed, in
	// primogems.
	valid := 0.0
	if fiveNum > 0 && fiveNum > missed {
		pulled := all - noFive
		if !fives[0].up {
			pulled -= fives[0].count
		}
		valid, _ = strconv.ParseFloat(jsFixed(float64(pulled)/float64(fiveNum-missed), 2), 64)
	}
	primogems := jsFixed(valid*160, 0)
	if valid*160 >= 10000 {
		primogems = jsFixed(valid*160/10000, 2) + "w"
	}
	stats = miaoStat(stats, primogems, "UP原石")
	if noFive > 0 {
		fives = append([]five{{up: true, date: context.Now.In(chinaTime).Format("01-02"), count: noFive, drawn: true, name: "已抽",
			img: resources.Artwork("gacha-imgs-no-avatar", "miao-plugin", "resources/gacha/imgs/no-avatar.webp")}}, fives...)
	}
	limit := 90.0
	if pool == "302" {
		limit = 80
	}
	items := []any{}
	for index, entry := range fives {
		bar := "bad"
		switch count := float64(entry.count); {
		case count <= 10:
			bar = "gold"
		case count < limit*0.5:
			bar = "good"
		case count < limit*0.83:
			bar = "normal"
		}
		items = append(items, map[string]any{"has_date": index == 0 || fives[index-1].date != entry.date, "first": index == 0, "up": entry.up, "drawn": entry.drawn,
			"date": entry.date, "name": entry.name, "star": 5, "img": entry.img, "count": entry.count, "bar": bar,
			"width": strconv.FormatFloat(float64(entry.count)/limit*100, 'f', -1, 64)})
	}
	data := miaoPlayer(context, resources, image)
	data["stats"], data["name_width"], data["items"] = stats, 90, items
	return app.Image{Template: "gacha-detail", Data: data, Resources: resources.List}, true
}

// miaoStatPools are the pools each kind of 喵喵抽卡统计 reads.
var miaoStatPools = map[string][]string{"up": {"301", "302"}, "char": {"301"}, "weapon": {"302"}, "normal": {"200"}, "mix": {"500"}, "all": {"301", "302", "200", "500"}}

// GachaStat draws 喵喵抽卡统计 the way miao's GachaData.stat and gacha/gacha-stat
// do: the totals over the pools the word names, then each period, newest
// first, with its rate-ups, dates and counts and every four- and five-star it
// gave, rate-ups marked; the standard wish and 全部 are one block.
func GachaStat(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	word := miaoStatWords.ReplaceAllString(image.Word, "")
	kind := "up"
	switch {
	case strings.Contains(word, "武器") || strings.Contains(word, "光锥"):
		kind = "weapon"
	case strings.Contains(word, "角色"):
		kind = "char"
	case strings.Contains(word, "常驻"):
		kind = "normal"
	case strings.Contains(word, "集录"):
		kind = "mix"
	case strings.Contains(word, "全部"):
		kind = "all"
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range miaoGachaArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	pulls := miaoPulls(context, resources, image.Archive, miaoStatPools[kind]...)
	if len(pulls) == 0 {
		return app.Image{}, false
	}
	versioned := kind != "normal" && kind != "all"
	periods := newMiaoPeriods(context)
	type block struct {
		version miaoVersion
		counts  map[*miaoItem]int
	}
	blocks := []*block{}
	for _, pull := range pulls {
		if len(blocks) == 0 || versioned && blocks[len(blocks)-1].version.before(pull.at) {
			version := periods.at(pull.at, versioned, kind == "mix")
			if !versioned {
				version.version = map[bool]string{true: "全部统计", false: "常驻池"}[kind == "all"]
			}
			// A pull in no period stays with the unknown block before it.
			if len(blocks) == 0 || !(version.from == "" && blocks[len(blocks)-1].version.version == version.version) {
				blocks = append(blocks, &block{version, map[*miaoItem]int{}})
			}
		}
		blocks[len(blocks)-1].counts[pull.item]++
	}
	keys := []string{"totalNum", "star5Num", "c5UpNum", "w5UpNum", "star4Num", "upNum", "c4Num", "w4Num"}
	total := map[string]int{}
	versions := []any{}
	for _, entry := range blocks {
		version := entry.version
		names := []string{}
		for _, name := range version.char5 {
			if found, ok := context.Catalog.Resolve(name, "character", nil); ok && found.Abbr != "" {
				name = found.Abbr
			}
			names = append(names, name)
		}
		up := map[string]bool{}
		for _, name := range append(slices.Clone(version.char5), version.weapon5...) {
			up[name] = true
		}
		counts := map[string]int{}
		type counted struct {
			item *miaoItem
			num  int
			up   bool
		}
		list := []counted{}
		for item, num := range entry.counts {
			isUp := up[item.name]
			list = append(list, counted{item, num, isUp})
			switch {
			case !item.weapon && item.star == 5:
				counts["c5Num"] += num
				if isUp {
					counts["c5UpNum"] += num
				}
			case !item.weapon:
				counts["c4Num"] += num
			case item.star == 5:
				counts["w5Num"] += num
				if isUp {
					counts["w5UpNum"] += num
				}
			case item.star == 4:
				counts["w4Num"] += num
			default:
				counts["w3Num"] += num
			}
		}
		counts["upNum"] = counts["w5UpNum"] + counts["c5UpNum"]
		counts["star5Num"] = counts["w5Num"] + counts["c5Num"]
		counts["star4Num"] = counts["w4Num"] + counts["c4Num"]
		counts["totalNum"] = counts["star5Num"] + counts["star4Num"] + counts["w3Num"]
		for _, key := range keys {
			total[key] += counts[key]
		}
		// Upstream's sortBy star, count and rate-up, reversed; the rest by
		// ID, as object keys enumerate.
		sort.Slice(list, func(i, j int) bool {
			a, b := list[i], list[j]
			if a.item.star != b.item.star {
				return a.item.star > b.item.star
			}
			if a.num != b.num {
				return a.num > b.num
			}
			if a.up != b.up {
				return a.up
			}
			x, _ := strconv.Atoi(a.item.id)
			y, _ := strconv.Atoi(b.item.id)
			return x > y
		})
		items := []any{}
		for _, item := range list {
			if item.item.star < 4 {
				continue
			}
			name := item.item.name
			if utf8.RuneCountInString(name) > 4 {
				name = item.item.abbr
			}
			items = append(items, map[string]any{"up": item.up, "star": item.item.star, "img": item.item.img, "num": item.num, "name": name})
		}
		stats := []any{}
		for _, key := range []string{"totalNum", "star5Num", "upNum", "c4Num", "w4Num"} {
			if counts[key] > 0 {
				stats = miaoStat(stats, counts[key], map[string]string{"totalNum": "总抽卡", "star5Num": "金卡", "upNum": "UP金卡", "c4Num": "紫角色", "w4Num": "紫武器"}[key])
			}
		}
		from, to := "", ""
		if versioned && version.from != "" {
			from = version.from[2:10]
			if version.to != "" {
				to = version.to[2:10]
			}
		}
		versions = append(versions, map[string]any{"version": version.version, "half": version.half, "from": from, "to": to, "name": strings.Join(names, " / "), "stats": stats, "items": items})
	}
	stats := []any{}
	for _, key := range []string{"totalNum", "star5Num", "c5UpNum", "w5UpNum", "star4Num"} {
		if total[key] > 0 {
			stats = miaoStat(stats, total[key], map[string]string{"totalNum": "抽卡总数", "star5Num": "金卡", "c5UpNum": "UP角色", "w5UpNum": "UP武器", "star4Num": "紫卡"}[key])
		}
	}
	if total["upNum"] > 0 {
		stats = miaoStat(stats, jsFixed(float64(total["totalNum"])/float64(total["upNum"]), 1), "平均UP抽")
	}
	data := miaoPlayer(context, resources, image)
	data["stats"], data["mix"], data["card_width"], data["versions"] = stats, kind == "mix", 69, versions
	return app.Image{Template: "gacha-stat", Data: data, Resources: resources.List}, true
}
