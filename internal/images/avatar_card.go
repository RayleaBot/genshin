package images

import (
	"strconv"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

// avatarCards builds miao's common avatar-card for the requester's own
// characters: face and gacha portrait, level and constellation, talents with
// their crown and bonus marks, the weapon with its refinement and the
// artifact sets. Characters the query does not return are left out.
type avatarCards struct {
	catalog   app.Catalog
	resources *app.ImageResources
	records   map[string]reference.Character
	weapons   map[string]reference.Weapon
	panels    map[string]app.CharacterPanel
}

func newAvatarCards(context app.ImageContext, resources *app.ImageResources, ids []string) *avatarCards {
	cards := &avatarCards{catalog: context.Catalog, resources: resources, records: map[string]reference.Character{}, weapons: map[string]reference.Weapon{}, panels: map[string]app.CharacterPanel{}}
	if context.Game.Calc != nil {
		metadata := context.Game.Calc.Metadata()
		for _, record := range metadata.Characters {
			cards.records[record.ID] = record
		}
		for _, weapon := range metadata.Weapons {
			cards.weapons[weapon.ID] = weapon
		}
	}
	// The detail query takes at most 50 characters at a time.
	for start := 0; context.Query != nil && start < len(ids); start += 50 {
		list := []any{}
		for _, id := range ids[start:min(start+50, len(ids))] {
			list = append(list, id)
		}
		if result, err := context.Query("genshin.character", map[string]any{"character_ids": list}); err == nil {
			for _, panel := range app.NormalizePanels(result, context.Catalog) {
				cards.panels[panel.ID] = panel
			}
		}
	}
	return cards
}

// miao adds a downloaded miao-plugin image under a generated ID.
func (c *avatarCards) miao(name string) string {
	return c.resources.Artwork("miao-"+strconv.Itoa(len(c.resources.List)), "miao-plugin", name)
}

// base is what every card shows, from the character's catalog record.
func (c *avatarCards) base(id string) map[string]any {
	record := c.records[id]
	star := app.Int(record.Data["star"])
	if star != 4 {
		star = 5
	}
	folder, _ := app.CharacterFolders(id, record.Name, record.Element)
	path := folder + "imgs/"
	return map[string]any{"known": record.Name != "", "name": record.Name, "abbr": abbreviation(c.catalog, record), "elem": record.Element, "star": star,
		"face": c.miao(path + "face.webp"), "gacha": c.miao(path + "gacha.webp")}
}

// own is the card of the requester's character, or false when the query did
// not return it.
func (c *avatarCards) own(id string) (map[string]any, bool) {
	panel, ok := c.panels[id]
	if !ok {
		return nil, false
	}
	card := c.base(id)
	card["level"], card["cons"] = panel.Level, panel.Rank
	levels := app.PanelTalents(panel, c.records[id])
	talents := []any{}
	for _, key := range []string{"a", "e", "q"} {
		level := levels[key]
		talents = append(talents, map[string]any{"key": key, "level": level.Level, "original": level.Original, "crown": level.Original == 10, "plus": level.Level > level.Original})
	}
	if levels["a"].Level > 0 {
		card["talents"] = talents
	}
	if weapon := panel.Weapon; weapon != nil {
		entry := c.weapons[weapon.ID]
		// miao colours refinement 5 with the constellation 6 badge.
		badge := weapon.Refinement
		if badge > 4 {
			badge++
		}
		card["weapon"] = map[string]any{"icon": c.miao("resources/meta-gs/weapon/" + entry.Type + "/" + entry.Name + "/icon.webp"),
			"star": app.Int(weapon.Rarity), "affix": weapon.Refinement, "badge": badge, "level": weapon.Level}
	}
	// miao shows each set worn as two or four pieces, by its flower (or its
	// circlet when the set has no flower).
	counts, order := map[string]int{}, []string{}
	for _, piece := range panel.Equipment {
		if piece.SetName == "" {
			continue
		}
		if counts[piece.SetName] == 0 {
			order = append(order, piece.SetName)
		}
		counts[piece.SetName]++
	}
	sets := []any{}
	for _, name := range order {
		if counts[name] >= 2 {
			sets = append(sets, c.setIcon(name))
		}
	}
	card["artis"] = sets
	return card, true
}

// setIcon is miao's picture of a set: its flower, or its circlet when the set
// has no flower.
func (c *avatarCards) setIcon(name string) string {
	if icon := c.miao("resources/meta-gs/artifact/imgs/" + name + "/1.webp"); icon != "" {
		return icon
	}
	return c.miao("resources/meta-gs/artifact/imgs/" + name + "/5.webp")
}

// guest is the card miao draws for a character the requester does not own at
// that level or constellation: only what the record shows.
func (c *avatarCards) guest(id string, level, cons int) map[string]any {
	card := c.base(id)
	card["level"], card["cons"], card["artis"] = level, cons, []any{}
	return card
}
