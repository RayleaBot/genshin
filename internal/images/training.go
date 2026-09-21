package images

import (
	"unicode/utf8"

	gamekit "github.com/RayleaBot/game-plugin-kit"
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

// Training draws 练度统计 the way miao's character/profile-stat does: every
// character in miao's order with level, constellation, friendship, the three
// talents coloured by their original level, the weapon, and the artifact
// sets with the pinned scoring's grade and score.
func Training(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	list, _ := result.Data["list"].([]any)
	if len(list) == 0 {
		return gamekit.Image{}, false
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range trainingArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	entries, _ := buildRoster(context, resources, list)
	rows := []any{}
	for _, item := range entries {
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
				level, original = talent["level"], gamekit.Int(talent["original"])
			}
			talents = append(talents, map[string]any{"level": level, "class": trainingLevels[min(max(original, 0), 10)],
				"plus": gamekit.Int(level) > original})
		}
		row := map[string]any{"no": len(rows) + 1, "star": item.star, "face": card["face"], "name": card["abbr"], "level": item.level, "cons": item.cons, "fetter": fetter, "talents": talents}
		if weapon, _ := card["weapon"].(map[string]any); weapon != nil && item.panel != nil && item.panel.Weapon != nil {
			// miao shortens names longer than four characters.
			name := item.panel.Weapon.Name
			if entry, ok := context.Catalog.Get(item.panel.Weapon.ID); ok {
				name = entry.Name
				if utf8.RuneCountInString(name) > 4 && len(entry.Aliases) > 0 {
					name = entry.Aliases[0]
				}
			}
			row["weapon"] = map[string]any{"star": weapon["star"], "level": weapon["level"], "icon": weapon["icon"], "affix": weapon["affix"],
				"badge": gamekit.Int(weapon["affix"]) + 1, "name": name}
		}
		row["artis"] = card["artis"]
		// Characters with artifacts get the pinned scoring's grade and score.
		if item.panel != nil && len(item.panel.Equipment) > 0 && context.Score != nil {
			if scored, err := context.Score(*item.panel); err == nil && scored.ScoreDetail != nil {
				row["mark"], row["grade"] = scored.ScoreDetail.Mark, scored.ScoreDetail.Grade
			}
		}
		rows = append(rows, row)
	}
	return gamekit.Image{Template: "training", Data: map[string]any{
		"uid": result.Role.UID, "count": len(rows), "rows": rows, "updated": context.Now.In(chinaTime).Format("2006-01-02 15:04"),
	}, Resources: resources.List}, true
}
