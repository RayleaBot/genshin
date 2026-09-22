package images

import (
	"cmp"
	"encoding/json"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

// dailyMaterialArtwork maps the images named in the converted stylesheets
// (miao's common and wiki/today-material) that the page shows to their paths.
var dailyMaterialArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"wiki-imgs-item-bg", "resources/wiki/imgs/item-bg.png"},
}

// dailyWeaponWeeks are the weapon materials by week group and city, from
// miao's material/daily.js.
var dailyWeaponWeeks = [3][]string{
	{"高塔孤王", "孤云寒林", "远海夷地", "谧林涓露", "悠古弦音", "贡祭炽心", "奇巧秘器", "苍星军势"},
	{"凛风奔狼", "雾海云间", "鸣神御灵", "绿洲花园", "纯圣露滴", "谵妄圣主", "长夜燧火", "藏窖灵浆"},
	{"狮牙斗士", "漆黑陨铁", "今昔剧画", "烈日威权", "无垢之海", "神合秘烟", "终北遗嗣", "凛雪帝皇"},
}

var dailyBookName = regexp.MustCompile(`「(.+)」`)

// dailyMaterialCity is the week group and city (from 1) of a talent book or a
// weapon material, as miao's material index derives them from the name.
func dailyMaterialCity(kind, name string) (int, int) {
	abbr, weeks := string([]rune(name)[:min(4, len([]rune(name)))]), dailyWeaponWeeks
	if kind == "talent" {
		abbr, weeks = name, calendarTalentWeeks
		if match := dailyBookName.FindStringSubmatch(name); match != nil {
			abbr = match[1]
		}
	}
	for week, list := range weeks {
		if city := slices.Index(list, abbr); city >= 0 {
			return week + 1, city + 1
		}
	}
	return 0, 0
}

type dailySection struct {
	kind, material string
	city           int
	icons          []any
	data           []map[string]any
	sort           []float64
}

// DailyMaterial draws 今日素材 the way miao's wiki/today-material does: for
// the day's week group, each city's talent book with the characters that use
// it (level, constellation and talents, finished ones last and greyed), then
// each city's weapon material with the equipped weapons that use it.
func DailyMaterial(context app.ImageContext, page app.DailyMaterialImage) (app.Image, bool) {
	if context.Game.Calc == nil || context.Artwork == nil {
		return app.Image{}, false
	}
	raw, err := context.Artwork.Open("miao-plugin", "resources/meta-gs/material/data.json")
	var materials map[string]struct {
		Type  string
		Items map[string]struct{ Star int }
	}
	if err != nil || json.Unmarshal(raw, &materials) != nil {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range dailyMaterialArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	metadata := context.Game.Calc.Metadata()
	weapons := map[string]reference.Weapon{}
	for _, weapon := range metadata.Weapons {
		weapons[weapon.ID] = weapon
	}
	picture := func(id, name string) string { return resources.Artwork(id, "miao-plugin", name) }
	sections := map[string]*dailySection{}
	// add puts an entry under its material's city when the material is the
	// day's, with miao's sort: four numbers packed, finished entries last.
	add := func(kind, name string, entry map[string]any, keys []float64, max bool) {
		week, city := dailyMaterialCity(kind, name)
		if week != page.Week {
			return
		}
		key := kind + strconv.Itoa(city)
		section := sections[key]
		if section == nil {
			section = &dailySection{kind: kind, material: name, city: city}
			// The lower-star items, highest first.
			top := materials[name]
			items := []string{}
			for item, data := range top.Items {
				items = append(items, strconv.Itoa(data.Star)+"\x00"+item)
			}
			slices.Sort(items)
			slices.Reverse(items)
			for _, item := range items {
				item = item[strings.IndexByte(item, 0)+1:]
				section.icons = append(section.icons, picture("material-"+item, "resources/meta-gs/material/"+top.Type+"/"+item+".webp"))
			}
			sections[key] = section
		}
		sort := 0.0
		for index, value := range keys {
			sort += value * math.Pow(10, float64(6-2*index))
		}
		if max {
			sort -= 1e8
		}
		entry["max"] = max
		section.data = append(section.data, entry)
		section.sort = append(section.sort, sort)
	}
	for _, kept := range page.Panels {
		panel := kept.Panel
		var record reference.Character
		for _, candidate := range metadata.Characters {
			if candidate.ID == panel.ID && (record.ID == "" || strings.EqualFold(candidate.Element, panel.Element)) {
				record = candidate
			}
		}
		character, ok := context.Catalog.Get(panel.ID)
		if !ok || record.ID == "" {
			continue
		}
		star := app.Int(record.Data["star"])
		id, _ := strconv.Atoi(panel.ID)
		path := "resources/meta-gs/character/" + record.Name + "/imgs/"
		levels := app.PanelTalents(panel, record)
		talents, lowest := []any{}, math.MaxInt
		for _, key := range []string{"a", "e", "q"} {
			level := levels[key]
			talents = append(talents, map[string]any{"level": level.Level, "crown": level.Original >= 10, "plus": level.Level > level.Original})
			lowest = min(lowest, level.Original)
		}
		// miao's isMaxTalent: every talent at the ascension's cap.
		cap := 10
		if promote := dailyPromote(panel); promote >= 1 {
			cap = []int{1, 2, 4, 6, 8, 10}[promote-1]
		}
		add("talent", character.Materials["天赋材料"], map[string]any{"face": picture("face-"+panel.ID, path+"face.webp"), "star": star, "level": panel.Level, "cons": panel.Rank, "talents": talents},
			[]float64{float64(panel.Level), float64(star), float64(panel.Rank), float64(id - 10000000)}, lowest >= cap)
		w := panel.Weapon
		if w == nil {
			continue
		}
		entry, ok := context.Catalog.Get(w.ID)
		weapon, known := weapons[w.ID]
		if !ok || !known {
			continue
		}
		abbr := entry.Abbr
		if abbr == "" {
			abbr = entry.Name
		}
		weaponID, _ := strconv.Atoi(w.ID)
		promote := 0
		if w.Promote != nil {
			promote = *w.Promote
		}
		maxPromote := 6
		if entry.Rarity <= 2 {
			maxPromote = 4
		}
		add("weapon", entry.Materials["武器材料"], map[string]any{"face": picture("side-"+panel.ID, path+"side.webp"), "star": entry.Rarity, "affix": w.Refinement, "affix_shown": w.Refinement > 1,
			"icon": picture("weapon-"+w.ID, "resources/meta-gs/weapon/"+weapon.Type+"/"+entry.Name+"/icon.webp"), "level": w.Level, "abbr": abbr},
			[]float64{float64(w.Level), float64(entry.Rarity), float64(w.Refinement), float64(weaponID) / 100}, promote >= maxPromote)
	}
	// Talent books first, then weapon materials; cities from the last one
	// back, as miao unshifts them.
	list := []*dailySection{}
	for _, section := range sections {
		list = append(list, section)
	}
	slices.SortFunc(list, func(x, y *dailySection) int {
		return cmp.Or(cmp.Compare(x.kind, y.kind), cmp.Compare(y.city, x.city))
	})
	out := []any{}
	for _, section := range list {
		order := make([]int, len(section.data))
		for index := range order {
			order[index] = index
		}
		// lodash orderBy by sort, then reversed.
		slices.SortStableFunc(order, func(x, y int) int { return cmp.Compare(section.sort[x], section.sort[y]) })
		slices.Reverse(order)
		data := []any{}
		for _, index := range order {
			data = append(data, section.data[index])
		}
		city := calendarCities[section.city-1]
		out = append(out, map[string]any{"type": section.kind, "city": city, "city_icon": picture("city-"+city, "resources/common/item/"+city+".png"), "material": section.material, "icons": section.icons, "data": data})
	}
	return app.Image{Template: "daily-material", Data: map[string]any{"uid": page.UID, "prefix": context.Game.Prefix, "sections": out}, Resources: resources.List}, true
}

// dailyPromote is the character's ascension, or the one its level implies
// as miao's calcPromote reads it.
func dailyPromote(panel app.CharacterPanel) int {
	if panel.Promote != nil {
		return *panel.Promote
	}
	steps := []int{1, 20, 40, 50, 60, 70, 80, 90, 100}
	for index := 0; index < len(steps)-1; index++ {
		if panel.Level >= steps[index] && panel.Level <= steps[index+1] {
			return index
		}
	}
	return len(steps) - 1
}
