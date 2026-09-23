package images

import (
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// characterCardArtwork maps the images the converted stylesheets name.
var characterCardArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-crown", "resources/character/imgs/crown.png"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"common-item-fetter", "resources/common/item/fetter.png"},
	{"common-item-star-ltr", "resources/common/item/star-ltr.png"},
}

// cardSources are the names miao's card gives panel services.
var cardSources = map[string]string{"mihoyo": "米游社", "enka": "Enka.Network", "mihomo": "Mihomo"}

// CharacterCard draws a bare character name the way miao's
// character/character-card does: the photo, the name with friendship and
// constellation, the UID and level, then the weapon, talents and artifacts
// from the panel. A landscape photo is drawn 480 high beside at 1.45, a
// portrait one 600 wide at 1.2, as miao sizes the page per photo.
func CharacterCard(context app.ImageContext, card app.CharacterCardImage) (app.Image, bool) {
	if card.Picture.Path == "" || card.Width < 1 || card.Height < 1 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range characterCardArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	resources.List = append(resources.List, card.Picture)
	width, zoom, mode := 600.0, 1.2, "bottom"
	if card.Width > card.Height {
		width, zoom, mode = 480*float64(card.Width)/float64(card.Height), 1.45, "left"
	}
	name := card.Entry.Name
	if utf8.RuneCountInString(name) > 4 && card.Entry.Abbr != "" {
		name = card.Entry.Abbr
	}
	data := map[string]any{"mode": mode, "width": width, "zoom": zoom, "name": name, "uid": card.UID, "bg": card.Picture.ID}
	panel := card.Panel
	if panel == nil || context.Game.Calc == nil {
		return app.Image{Template: "character-card", Data: data, Resources: resources.List}, true
	}
	record := card.Record
	data["level"], data["cons"], data["has_cons"], data["elem"] = panel.Level, panel.Rank, true, record.Element
	if base, _ := panel.Official["base"].(map[string]any); base != nil {
		data["fetter"] = app.Int(base["fetter"])
	}
	resources.Artwork("common-bg-talent-"+record.Element, "miao-plugin", "resources/common/bg/talent-"+record.Element+".webp")
	if weapon := panel.Weapon; weapon != nil {
		item := map[string]any{"name": weapon.Name, "star": app.Int(weapon.Rarity), "level": weapon.Level, "affix": weapon.Refinement}
		for _, entry := range context.Game.Calc.Metadata().Weapons {
			if entry.ID == weapon.ID {
				item["img"] = resources.Artwork("weapon", "miao-plugin", "resources/meta-gs/weapon/"+entry.Type+"/"+entry.Name+"/icon.webp")
				if abbr, ok := context.Catalog.Get(entry.ID); ok && utf8.RuneCountInString(weapon.Name) > 4 && abbr.Abbr != "" {
					item["name"] = abbr.Abbr
				}
			}
		}
		data["weapon"] = item
	}
	if record.Name != "" {
		characterPath, iconPath := characterFolders(panel.ID, record.Name, record.Element)
		talentCons, _ := record.Data["talentCons"].(map[string]any)
		icons := map[string]string{"a": "resources/common/item/atk-" + record.WeaponType + ".webp"}
		for _, key := range []string{"e", "q"} {
			if cons := app.Int(talentCons[key]); cons > 0 {
				icons[key] = iconPath + "icons/cons-" + strconv.Itoa(cons) + ".webp"
			} else {
				icons[key] = characterPath + "icons/talent-" + key + ".webp"
			}
		}
		levels := app.PanelTalents(*panel, record)
		if levels["a"].Level > 0 {
			talents := []any{}
			for _, key := range []string{"a", "e", "q"} {
				level := levels[key]
				talents = append(talents, map[string]any{"icon": resources.Artwork("talent-"+key, "miao-plugin", icons[key]), "level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10})
			}
			data["talents"] = talents
		}
	}
	artis := []any{}
	for slot := 1; slot <= 5; slot++ {
		item := map[string]any{}
		for _, piece := range panel.Equipment {
			if piece.Slot == slot {
				item = map[string]any{"level": piece.Level, "img": resources.Artwork("artifact-"+strconv.Itoa(slot), "miao-plugin", "resources/meta-gs/artifact/imgs/"+piece.SetName+"/"+strconv.Itoa(slot)+".webp")}
			}
		}
		artis = append(artis, item)
	}
	data["artis"] = artis
	data["artis_set"] = app.SetShortName(context.Catalog, *panel)
	if data["artis_set"] == "" {
		data["artis_set"] = "圣遗物"
	}
	source := cardSources[panel.Source]
	if source == "" {
		source = panel.Source
	}
	updated := context.Now
	if panel.UpdatedAtMS > 0 {
		updated = time.UnixMilli(panel.UpdatedAtMS)
	}
	data["source"], data["update_time"] = source, updated.In(chinaTime).Format("01-02 15:04")
	return app.Image{Template: "character-card", Data: data, Resources: resources.List}, true
}
