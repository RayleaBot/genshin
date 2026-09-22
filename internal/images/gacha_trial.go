package images

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// trialArtwork maps the fonts and background Yunzai's html/gacha/gacha-trial
// uses to its paths.
var trialArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"YS", "resources/font/HYWenHei-55W.ttf"},
	{"img-gacha-items-background", "resources/img/gacha/items/background.jpg"},
}

// trialIcons are Yunzai's item icons, by resource ID: the Stardust and
// Starglitter a repeated character converts to, the elements and the weapon
// types.
var trialIcons = map[string]string{"stardust-gold": "星尘金", "stardust-purple": "星尘紫", "starglitter-10": "星辉10", "starglitter-2": "星辉2",
	"pyro": "火", "hydro": "水", "anemo": "风", "electro": "雷", "dendro": "草", "cryo": "冰", "geo": "岩",
	"sword": "单手剑", "claymore": "大剑", "polearm": "枪", "catalyst": "法器", "bow": "弓"}

// trialElements are the element icons' IDs by the catalog's element names.
var trialElements = map[string]string{"火": "pyro", "水": "hydro", "风": "anemo", "雷": "electro", "草": "dendro", "冰": "cryo", "岩": "geo"}

// trialWeaponTypes are miao's weapon directories, which are also the weapon
// type icons' IDs.
var trialWeaponTypes = []string{"sword", "claymore", "polearm", "catalyst", "bow"}

type trialItem struct {
	draw     app.SimulationDraw
	entry    app.Entry
	weapon   bool
	have     bool
	image    string
	category string
}

// GachaTrial draws 十连 the way Yunzai's html/gacha/gacha-trial does: the draws
// by rarity, characters before weapons, first copies before repeats, then in
// draw order. Repeated characters show the Stardust and Starglitter they turn
// into and five-stars the pulls they took. Unless the ten hold four
// five-stars, the corners show the requester, the last five-star listed or the
// pity count, and the featured character or the weapon's Epitomized Path.
func GachaTrial(context app.ImageContext, trial app.SimulationImage) (app.Image, bool) {
	if context.Artwork == nil || len(trial.Draws) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range trialArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	icon := func(id, name string) string {
		return resources.Artwork(id, "yunzai-genshin", "resources/img/gacha/items/"+name+".png")
	}
	items := []trialItem{}
	seen := map[int]map[string]bool{5: {}, 4: {}}
	for _, draw := range trial.Draws {
		item := trialItem{draw: draw, weapon: draw.Item != "character"}
		item.entry, _ = context.Catalog.Resolve(draw.Name, draw.Item, nil)
		if item.entry.Name == "" {
			item.entry.Name = draw.Name
		}
		if names := seen[draw.Rarity]; names != nil {
			item.have = names[item.entry.Name]
			names[item.entry.Name] = true
		}
		if item.weapon {
			for _, category := range trialWeaponTypes {
				if _, ok := context.Artwork.File("miao-plugin", "resources/meta-gs/weapon/"+category+"/"+item.entry.Name+"/gacha.webp"); ok {
					item.category = category
					break
				}
			}
			if item.category != "" {
				item.image = resources.Artwork("gacha-"+strconv.Itoa(len(items)), "miao-plugin", "resources/meta-gs/weapon/"+item.category+"/"+item.entry.Name+"/gacha.webp")
			}
		} else {
			item.category = trialElements[item.entry.Element]
			item.image = resources.Artwork("gacha-"+strconv.Itoa(len(items)), "miao-plugin", "resources/meta-gs/character/"+item.entry.Name+"/imgs/gacha.webp")
		}
		items = append(items, item)
	}
	slices.SortStableFunc(items, func(a, b trialItem) int {
		return cmp.Or(b.draw.Rarity-a.draw.Rarity, trialLast(a.weapon)-trialLast(b.weapon), trialLast(a.have)-trialLast(b.have))
	})
	five := 0
	for _, draw := range trial.Draws {
		if draw.Rarity == 5 {
			five++
		}
	}
	info := "累计「" + strconv.Itoa(trial.Pity.Five) + "抽」"
	list := []any{}
	for _, item := range items {
		star := item.draw.Rarity
		// Yunzai marks repeats of any kind but shows the conversion only for
		// characters.
		repeat := item.have && !item.weapon
		shadow := "shadow-" + strconv.Itoa(star)
		if star == 5 {
			shadow += "-" + strconv.Itoa(rand.IntN(7)+1)
			info = trialShortName(item.entry, item.weapon) + "「" + strconv.Itoa(item.draw.Interval) + "抽」"
			if item.draw.Guaranteed {
				info += "大保底"
			}
			if item.draw.Fate {
				info += "定轨"
			}
		}
		entry := map[string]any{"weapon": item.weapon, "weapon_five": item.weapon && star == 5, "have": repeat, "image": item.image,
			"shadow": icon(shadow, shadow), "star": icon("s-"+strconv.Itoa(star), "s-"+strconv.Itoa(star))}
		if star == 5 && !item.have && five < 4 {
			entry["times"] = item.draw.Interval
		}
		if repeat {
			dust, glitter := "stardust-purple", "starglitter-2"
			if star == 5 {
				dust, glitter = "stardust-gold", "starglitter-10"
			}
			entry["dust"], entry["glitter"] = icon(dust, trialIcons[dust]), icon(glitter, trialIcons[glitter])
		} else if item.category != "" {
			entry["element"] = icon(item.category, trialIcons[item.category])
		}
		list = append(list, entry)
	}
	data := map[string]any{"list": list, "header": five < 4, "info": info, "weapon": trial.Selection.Kind == "weapon", "life": trial.Pity.Fate,
		"bg": icon("bg", "bg"), "bg2": icon("bg2", "bg2"), "bg_weapon": icon("bg-weapon", "bgWeapon")}
	switch trial.Selection.Kind {
	case "character":
		featured, _ := context.Catalog.Resolve(trial.Selection.Featured, "character", nil)
		if featured.Name == "" {
			featured.Name = trial.Selection.Featured
		}
		data["pool"] = "角色池：" + trialShortName(featured, false)
	case "standard":
		data["pool"] = "常驻池"
	case "weapon":
		if trial.Selection.FateTarget != "" {
			target, _ := context.Catalog.Resolve(trial.Selection.FateTarget, "weapon", nil)
			if target.Name == "" {
				target.Name = trial.Selection.FateTarget
			}
			data["bing"] = trialShortName(target, true)
		}
	}
	return app.Image{Template: "gacha-trial", Data: data, Resources: resources.List}, true
}

// trialShortName is Yunzai's shortName: miao's abbreviation, which for
// weapons applies only to names over four characters.
func trialShortName(entry app.Entry, weapon bool) string {
	if entry.Abbr == "" || weapon && utf8.RuneCountInString(entry.Name) <= 4 {
		return entry.Name
	}
	return entry.Abbr
}

// trialLast sorts the flagged draws after the others.
func trialLast(flag bool) int {
	if flag {
		return 1
	}
	return 0
}
