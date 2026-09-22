package images

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// characterWikiArtwork maps the images the converted stylesheets (miao's
// common, tpl, character/profile-detail and wiki/character-wiki) name to
// their paths; element backgrounds are added for the page's element.
var characterWikiArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
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
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"character-imgs-icon", "resources/character/imgs/icon.png"},
	{"cons0", "resources/common/item/cons0.webp"},
}

// wikiMaterialKeys are miao's material slots in page order with the amounts
// it prints; the talent book prints its days instead.
var wikiMaterialKeys = [][2]string{{"gem", "1/9/9/6"}, {"boss", "46"}, {"normal", "18/30/36"}, {"specialty", "168"}, {"talent", ""}, {"weekly", ""}}

// wikiWeaponTypes are miao's weapon type names.
var wikiWeaponTypes = map[string]string{"sword": "单手剑", "catalyst": "法器", "bow": "弓", "claymore": "双手剑", "polearm": "长柄武器"}

// wikiDesc is miao's CharMeta.getDesc: the description as one line, or
// broken into two at a dash or between its clauses.
func wikiDesc(desc string) string {
	desc = strings.TrimSuffix(desc, "。")
	desc = strings.Replace(desc, "</br>", "，", 1)
	desc = strings.NewReplacer("。", "，", ",", "，").Replace(desc)
	desc = strings.Replace(desc, "——", "，——", 1)
	if utf8.RuneCountInString(desc) < 25 {
		return desc
	}
	if strings.Contains(desc, "-") {
		index := strings.Index(desc, "—")
		if index < 0 {
			return desc
		}
		return desc[:index] + "</br>" + desc[index:]
	}
	const maxChars = 26
	clauses, length := []string{}, 0
	for _, clause := range strings.Split(desc, "，") {
		size := utf8.RuneCountInString(clause)
		if length+size >= maxChars*2 {
			break
		}
		clauses, length = append(clauses, clause), length+size
	}
	if length <= maxChars-6 {
		return strings.Join(clauses, "，")
	}
	// Clauses past the second line are dropped, as upstream prints two.
	lines := [][]string{{}, {}}
	line := 0
	grow := func() {
		for len(lines) <= line {
			lines = append(lines, []string{})
		}
	}
	for _, clause := range clauses {
		if utf8.RuneCountInString(strings.Join(lines[line], " "))+utf8.RuneCountInString(clause) > maxChars {
			line++
			grow()
		}
		lines[line] = append(lines[line], clause)
		if len(clauses) == 2 {
			line++
			grow()
		}
	}
	return strings.Join(lines[0], "，") + "</br>" + strings.Join(lines[1], "，")
}

// wikiDescHTML prints wikiDesc's text with its line break.
func wikiDescHTML(desc string) string {
	parts := strings.Split(wikiDesc(desc), "</br>")
	for index := range parts {
		parts[index] = html.EscapeString(parts[index])
	}
	return strings.Join(parts, "<br>")
}

// wikiStatistics are the public statistics miao's character page reads:
// lelaer's constellation, level and equipment usage by character name and
// yshelper's holding rate, each cached for an hour as upstream does.
var wikiStatistics = struct {
	sync.Mutex
	fetched time.Time
	roles   map[string]map[string]any
	owned   map[string]float64
}{}

const (
	wikiRoleAvgURL   = "https://api.lelaer.com/ys/getRoleAvg.php?star=all&lang=zh-Hans"
	wikiAbyssRankURL = "https://api.yshelper.com/ys/getAbyssRank.php?star=all&role=all&lang=zh-Hans"
)

// wikiFetch reads one statistics response; upstream accepts JSON wrapped in
// other text by taking what lies between the first { and the last }.
func wikiFetch(url string) map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil || response.StatusCode != http.StatusOK {
		return nil
	}
	var data map[string]any
	if json.Unmarshal(body, &data) != nil {
		start, end := strings.IndexByte(string(body), '{'), strings.LastIndexByte(string(body), '}')
		if start < 0 || end <= start || json.Unmarshal(body[start:end+1], &data) != nil {
			return nil
		}
	}
	return data
}

// wikiStats returns a character's lelaer entry and yshelper holding rate
// (negative when unknown), fetching both at most once an hour.
func wikiStats(name string, catalog app.Catalog) (map[string]any, float64) {
	wikiStatistics.Lock()
	defer wikiStatistics.Unlock()
	if wikiStatistics.roles == nil || time.Since(wikiStatistics.fetched) > time.Hour {
		roles, owned := map[string]map[string]any{}, map[string]float64{}
		if data := wikiFetch(wikiRoleAvgURL); data != nil {
			list, _ := data["result"].([]any)
			for _, raw := range list {
				role, _ := raw.(map[string]any)
				if entry, ok := catalog.Resolve(app.Text(role["role"]), "character", nil); ok {
					roles[entry.Name] = role
				}
			}
		}
		if data := wikiFetch(wikiAbyssRankURL); data != nil {
			list, _ := data["has_list"].([]any)
			for _, raw := range list {
				role, _ := raw.(map[string]any)
				if entry, ok := catalog.Resolve(app.Text(role["name"]), "character", nil); ok {
					rate, _ := strconv.ParseFloat(app.Text(role["own_rate"]), 64)
					owned[entry.Name] = rate / 100
				}
			}
		}
		// A failed fetch is retried on the next page rather than cached.
		if len(roles) > 0 {
			wikiStatistics.roles, wikiStatistics.owned, wikiStatistics.fetched = roles, owned, time.Now()
		}
	}
	rate, ok := wikiStatistics.owned[name]
	if !ok {
		rate = -1
	}
	return wikiStatistics.roles[name], rate
}

// miaoPercent is miao's Format.percent: the fraction as a percentage.
func miaoPercent(value float64, digits int) string { return jsFixed(value*100, digits) + "%" }

var wikiSetPart = regexp.MustCompile(`^(.*?)(4|2)$`)

// CharacterWiki draws 图鉴 and 资料 the way miao's wiki/character-wiki does:
// the splash with the title, name and description, the character's weapon
// type, constellation, birthday, allegiance and voice actors, the level-100
// base stats and ascension stat, the ascension and talent materials, then the
// public statistics: average level and constellation with each
// constellation's share, and the weapons and artifact sets used most.
func CharacterWiki(context app.ImageContext, character wikiCharacter) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range characterWikiArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	resources.Artwork("common-bg-bg-"+character.Elem, "miao-plugin", "resources/common/bg/bg-"+character.Elem+".webp")
	miao := func(name string) string {
		return resources.Artwork("miao-"+strconv.Itoa(len(resources.List)), "miao-plugin", "resources/"+name)
	}
	base := "meta-gs/character/" + character.Name + "/"
	attrs := []any{}
	for _, stat := range [][2]string{{"hp", "基础生命"}, {"atk", "基础攻击"}, {"def", "基础防御"}} {
		attrs = append(attrs, map[string]any{"title": stat[1], "value": miaoComma(character.BaseAttr[stat[0]], 1)})
	}
	growth := strconv.FormatFloat(character.GrowAttr.Value, 'f', -1, 64)
	if len(growth) > 10 {
		growth = miaoComma(character.GrowAttr.Value, 1)
	}
	label := wikiGrowth[character.GrowAttr.Key]
	if character.GrowAttr.Key == "dmg" {
		label = map[string]string{"pyro": "火", "hydro": "水", "anemo": "风", "electro": "雷", "dendro": "草", "cryo": "冰", "geo": "岩"}[character.Elem] + "伤"
	}
	attrs = append(attrs, map[string]any{"title": "成长·" + label, "value": growth})
	birthday := ""
	if month, day, ok := strings.Cut(character.Birth, "-"); ok {
		birthday = month + "月" + day + "日"
	}
	data := map[string]any{"elem": character.Elem, "title": character.Title, "name": character.Name, "desc": wikiDescHTML(character.Desc), "star": character.Star,
		"weapon": wikiWeaponTypes[character.Weapon], "astro": character.Astro, "birthday": birthday, "allegiance": character.Allegiance,
		"cncv": character.Cncv, "jpcv": character.Jpcv, "attrs": attrs, "splash": miao(base + "imgs/splash.webp"), "prefix": context.Game.Prefix,
		"abbr": character.Abbr}
	materials := []any{}
	for _, key := range wikiMaterialKeys {
		material, ok := wikiMaterial(context, character.Materials[key[0]])
		if !ok || key[0] == "boss" && traveler(strconv.Itoa(character.ID)) {
			continue
		}
		num := key[1]
		if key[0] == "talent" {
			if week, _ := talentBook(material.name); week > 0 {
				num = []string{"周一/周四", "周二/周五", "周三/周六"}[week-1]
			}
		}
		materials = append(materials, map[string]any{"type": material.kind, "num": num, "star": material.star, "label": material.label, "icon": miao(material.icon)})
	}
	data["materials"] = materials
	role, owned := wikiStats(character.Name, context.Catalog)
	if role != nil {
		level, _ := strconv.ParseFloat(app.Text(role["avg_level"]), 64)
		cons, _ := strconv.ParseFloat(app.Text(role["avg_class"]), 64)
		shares := []any{}
		for index := 0; index <= 6; index++ {
			share, _ := strconv.ParseFloat(app.Text(role["c"+strconv.Itoa(index)]), 64)
			icon := "cons0"
			if index > 0 {
				icon = miao(base + "icons/cons-" + strconv.Itoa(index) + ".webp")
			}
			shares = append(shares, map[string]any{"cons": index, "num": miaoPercent(share/100, 1), "icon": icon, "title": []string{"零", "一", "二", "三", "四", "五", "满"}[index] + "命"})
		}
		// Upstream shows the block whenever lelaer lists the character; an
		// unknown holding rate prints as 0.
		data["holding"] = map[string]any{"num": miaoPercent(max(owned, 0), 1), "level": jsFixed(level, 1), "cons": jsFixed(cons, 2), "shares": shares}
		data["weapons"] = wikiWeaponUsage(context, miao, role["weapon"])
		data["artis"] = wikiSetUsage(context, miao, role["artifacts_set"])
	}
	return app.Image{Template: "character-wiki", Data: data, Resources: resources.List}, true
}

type wikiMaterialEntry struct {
	name, kind, label, icon string
	star                    int
}

// wikiMaterial finds a material in miao's material data, as a material of its
// own or a tier of one, with miao's label: the city and book for talent books,
// the abbreviation for others.
func wikiMaterial(context app.ImageContext, name string) (wikiMaterialEntry, bool) {
	if name == "" || context.Artwork == nil {
		return wikiMaterialEntry{}, false
	}
	raw, err := context.Artwork.Open("miao-plugin", "resources/meta-gs/material/data.json")
	if err != nil {
		return wikiMaterialEntry{}, false
	}
	var data map[string]struct {
		Type  string
		Star  int
		Items map[string]struct {
			Type string
			Star int
		}
	}
	if json.Unmarshal(raw, &data) != nil {
		return wikiMaterialEntry{}, false
	}
	entry := wikiMaterialEntry{name: name}
	if top, ok := data[name]; ok {
		entry.kind, entry.star = top.Type, top.Star
	} else {
		for _, top := range data {
			if item, ok := top.Items[name]; ok {
				entry.kind, entry.star = top.Type, item.Star
				if item.Type != "" {
					entry.kind = item.Type
				}
				break
			}
		}
	}
	if entry.kind == "" {
		return wikiMaterialEntry{}, false
	}
	entry.icon = "meta-gs/material/" + entry.kind + "/" + name + ".webp"
	entry.label = name
	if abbr := context.Catalog.MaterialAbbrs[name]; abbr != "" {
		entry.label = abbr
	}
	if entry.kind == "talent" {
		_, entry.label = talentBook(name)
	}
	return entry, true
}

// wikiWeaponUsage is miao's getWeaponsData: the weapons by usage, each with
// its icon, rarity and short name.
func wikiWeaponUsage(context app.ImageContext, miao func(string) string, raw any) []any {
	list, _ := raw.([]any)
	type usage struct {
		item  map[string]any
		value float64
	}
	usages := []usage{}
	for _, value := range list {
		item, _ := value.(map[string]any)
		name := app.Text(item["name"])
		rate, _ := strconv.ParseFloat(app.Text(item["rate"]), 64)
		weapon := map[string]any{"name": name, "abbr": name}
		if entry, ok := context.Catalog.Resolve(name, "weapon", nil); ok {
			weapon["star"] = entry.Rarity
			if utf8.RuneCountInString(entry.Name) > 4 && entry.Abbr != "" {
				weapon["abbr"] = entry.Abbr
			}
			for kind := range wikiWeaponTypes {
				if icon := miao("meta-gs/weapon/" + kind + "/" + entry.Name + "/icon.webp"); icon != "" {
					weapon["icon"] = icon
					break
				}
			}
		}
		usages = append(usages, usage{weapon, rate / 100})
	}
	slices.SortStableFunc(usages, func(a, b usage) int { return compareDesc(a.value, b.value) })
	out := []any{}
	for _, item := range usages[:min(7, len(usages))] {
		item.item["value"] = miaoPercent(item.value, 1)
		out = append(out, item.item)
	}
	return out
}

// wikiSetUsage is miao's getArtisData: the set combinations by usage, each
// with its sets' pictures and abbreviations.
func wikiSetUsage(context app.ImageContext, miao func(string) string, raw any) []any {
	list, _ := raw.([]any)
	type usage struct {
		item  map[string]any
		value float64
	}
	usages := []usage{}
	for _, value := range list {
		item, _ := value.(map[string]any)
		rate, _ := strconv.ParseFloat(app.Text(item["rate"]), 64)
		images, titles := []any{}, []string{}
		for _, part := range strings.Split(app.Text(item["name"]), "+") {
			if match := wikiSetPart.FindStringSubmatch(part); match != nil {
				abbr := context.Catalog.SetAbbrs[match[1]]
				if abbr == "" {
					abbr = match[1]
				}
				icon := miao("meta-gs/artifact/imgs/" + match[1] + "/1.webp")
				if icon == "" {
					icon = miao("meta-gs/artifact/imgs/" + match[1] + "/5.webp")
				}
				images, titles = append(images, icon), append(titles, abbr+match[2])
			} else if part == "暂无套装" {
				images, titles = append(images, miao("common/item/artifact-icon.webp")), append(titles, "其它")
			}
		}
		if len(titles) > 0 {
			usages = append(usages, usage{map[string]any{"images": images, "count": len(images), "title": strings.Join(titles, "+")}, rate / 100})
		}
	}
	slices.SortStableFunc(usages, func(a, b usage) int { return compareDesc(a.value, b.value) })
	out := []any{}
	for _, item := range usages[:min(7, len(usages))] {
		item.item["value"] = miaoPercent(item.value, 1)
		out = append(out, item.item)
	}
	return out
}

func compareDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	}
	return 0
}
