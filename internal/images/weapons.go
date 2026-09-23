package images

import (
	"slices"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// weaponsArtwork maps the images Yunzai's html/avatar/weapon names.
var weaponsArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"bg3", "resources/img/other/bg3.png"},
	{"bg4", "resources/img/other/bg4.png"},
	{"bg5", "resources/img/other/bg5.png"},
	{"fill", "resources/img/other/fill.png"},
	{"item-sword", "resources/img/gacha/items/单手剑.png"},
	{"item-claymore", "resources/img/gacha/items/大剑.png"},
	{"item-bow", "resources/img/gacha/items/弓.png"},
	{"item-polearm", "resources/img/gacha/items/枪.png"},
	{"item-catalyst", "resources/img/gacha/items/法器.png"},
}

// weaponKinds are miao's weapon folders by the type number of the character
// list's weapons.
var weaponKinds = map[int]string{1: "sword", 10: "catalyst", 11: "claymore", 12: "bow", 13: "polearm"}

// eventWeapons are Yunzai's actWeapon: event weapons whose refinement does
// not raise them in the list.
var eventWeapons = []string{"降临之剑", "掠食者", "嘟嘟可故事集", "风花之颂", "腐殖之剑", "衔珠海皇", "「渔获」", "竭泽", "辰砂之纺锤", "证誓之明瞳", "落霞", "笼钓瓶一心", "风信之锋", "东花坊时雨", "饰铁之花", "鹮穿之喙", "灰河渡手", "无垠蔚蓝之歌", "「究极霸王超级魔剑」"}

// Weapons draws 武器 the way Yunzai's html/avatar/weapon does: every equipped
// weapon of three stars or more with its wearer's side portrait, refinement,
// level and name, ordered as Yunzai orders them, and past eight weapons the
// counts by rarity and type. Eight or fewer use the narrow page, as upstream
// narrows its body.
func Weapons(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	list, _ := result.Data["list"].([]any)
	if len(list) == 0 || context.Game.Calc == nil {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range weaponsArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	characters := map[string]string{}
	for _, record := range context.Game.Calc.Metadata().Characters {
		characters[record.ID] = record.Name
	}
	counts := map[string]int{}
	type row struct {
		item             map[string]any
		firstSort, order int
	}
	rows := []row{}
	for index, raw := range list {
		avatar, _ := raw.(map[string]any)
		weapon, _ := avatar["weapon"].(map[string]any)
		rarity, level, affix := app.Int(weapon["rarity"]), app.Int(weapon["level"]), app.Int(weapon["affix_level"])
		if weapon == nil || rarity <= 1 {
			continue
		}
		// Yunzai counts every equipped weapon, drawn or not.
		kind := weaponKinds[app.Int(weapon["type"])]
		counts[strconv.Itoa(rarity)]++
		counts[kind]++
		id := app.Text(avatar["id"])
		name := characters[id]
		if traveler(id) {
			name = "旅行者"
		}
		entry, ok := context.Catalog.Get(app.Text(weapon["id"]))
		if !ok || name == "" || kind == "" {
			continue
		}
		roleLevel, roleRarity := app.Int(avatar["level"]), min(app.Int(avatar["rarity"]), 5)
		// Yunzai's two keys; the second ordering decides and keeps the first's
		// order among equals.
		firstSort := level + (rarity-4)*20
		if level >= 20 {
			firstSort += roleLevel
		}
		if !slices.Contains(eventWeapons, entry.Name) {
			firstSort += affix * 5
		}
		order := rarity*1000000 + affix*100000 + level*1000 + roleRarity*100 + roleLevel
		folder, _ := app.CharacterFolders(id, name, "")
		shown := entry.Name
		if utf8.RuneCountInString(shown) > 4 && entry.Abbr != "" {
			shown = entry.Abbr
		}
		item := map[string]any{"name": shown, "level": strconv.Itoa(level), "bg": "bg" + strconv.Itoa(rarity),
			"side": resources.Artwork("side-"+strconv.Itoa(index), "miao-plugin", folder+"imgs/side.webp"),
			"icon": resources.Artwork("weapon-"+strconv.Itoa(index), "miao-plugin", "resources/meta-gs/weapon/"+kind+"/"+entry.Name+"/icon.webp")}
		if affix > 1 {
			item["affix"], item["affix_class"] = strconv.Itoa(affix), "life"+strconv.Itoa(affix)
		}
		rows = append(rows, row{item, firstSort, order})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].firstSort > rows[j].firstSort })
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].order > rows[j].order })
	items := []any{}
	for _, row := range rows {
		items = append(items, row.item)
	}
	template := "weapons-narrow"
	if len(items) > 8 {
		template = "weapons"
	}
	count := map[string]any{}
	for _, key := range []string{"5", "4", "3", "sword", "claymore", "bow", "polearm", "catalyst"} {
		count[key] = strconv.Itoa(counts[key])
	}
	return app.Image{Template: template, Data: map[string]any{"uid": result.Role.UID, "list": items, "count": count, "counted": len(items) > 8}, Resources: resources.List}, true
}
