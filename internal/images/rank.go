package images

import (
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// rankArtwork maps the images named in the converted stylesheets (miao's
// common and character/rank-profile-list) to their paths; element
// backgrounds are added for the elements a ranking shows.
var rankArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-crown", "resources/character/imgs/crown.png"},
	{"character-imgs-mark-icon", "resources/character/imgs/mark-icon.png"},
	{"character-imgs-mark-icon2", "resources/character/imgs/mark-icon2.png"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
}

// Rank draws a group ranking the way miao's character/rank-profile-list
// does: the title and the ranking notes, then a row per entry with its place
// (or, listing every character's best, the character), the member's avatar,
// constellation, name and UID, talents, weapon, artifact sets with their score
// and grade, and the default detail's damage.
func Rank(context app.ImageContext, rank app.RankImage) (app.Image, bool) {
	if context.Game.Calc == nil || len(rank.Entries) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range rankArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	cards := newAvatarCards(context, resources, nil)
	elem := "hydro"
	if record, ok := cards.records[rank.Character.ID]; ok {
		elem = record.Element
	}
	resources.Artwork("common-bg-bg-"+elem, "miao-plugin", "resources/common/bg/bg-"+elem+".webp")
	rows := []any{}
	for index, entry := range rank.Entries {
		record := cards.records[entry.CharacterID]
		cards.panels[entry.CharacterID] = *entry.Panel
		card, _ := cards.own(entry.CharacterID)
		resources.Artwork("common-bg-talent-"+record.Element, "miao-plugin", "resources/common/bg/talent-"+record.Element+".webp")
		name := record.Name
		// miao's sName: the abbreviation for names of four or more characters.
		if utf8.RuneCountInString(name) >= 4 {
			name = card["abbr"].(string)
		}
		row := map[string]any{"place": index + 1, "uid": entry.UID, "star": card["star"], "face": card["face"], "cons": entry.Panel.Rank, "name": name, "elem": record.Element}
		if entry.Avatar != "" {
			row["avatar"] = resources.Remote("avatar-"+strconv.Itoa(index), entry.Avatar)
		}
		path := "resources/meta-gs/character/" + record.Name + "/icons/"
		icons := map[string]string{"a": "resources/common/item/atk-" + record.WeaponType + ".webp", "e": path + "talent-e.webp", "q": path + "talent-q.webp"}
		talentCons, _ := record.Data["talentCons"].(map[string]any)
		for _, key := range []string{"e", "q"} {
			if cons := app.Int(talentCons[key]); cons > 0 {
				icons[key] = path + "cons-" + strconv.Itoa(cons) + ".webp"
			}
		}
		levels := app.PanelTalents(*entry.Panel, record)
		talents := []any{}
		for _, key := range []string{"a", "e", "q"} {
			level := levels[key]
			talents = append(talents, map[string]any{"level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10, "icon": cards.miao(icons[key])})
		}
		row["talents"] = talents
		if weapon := entry.Panel.Weapon; weapon != nil {
			info := card["weapon"].(map[string]any)
			label := weapon.Name
			if entry, ok := context.Catalog.Get(weapon.ID); ok && utf8.RuneCountInString(label) > 4 && entry.Abbr != "" {
				label = entry.Abbr
			}
			row["weapon"] = map[string]any{"icon": info["icon"], "star": info["star"], "name": label, "affix": weapon.Refinement, "badge": weapon.Refinement + 1, "level": weapon.Level}
		}
		sets := card["artis"].([]any)
		row["sets"], row["set_count"], row["set_name"] = sets[:min(2, len(sets))], min(2, len(sets)), app.RankSetName(context.Catalog, *entry.Panel)
		grade := entry.Grade
		if grade == "" {
			grade = "D"
		}
		row["grade"], row["mark"] = grade, miaoComma(entry.Score)
		if entry.Damage != nil {
			value := entry.Damage.Text
			if value == "" {
				value = miaoComma(entry.Damage.Value)
			}
			row["damage"] = map[string]any{"title": app.RankDamageTitle(entry.Damage.Title), "value": value}
		}
		rows = append(rows, row)
	}
	title := "最强排行"
	if rank.Mode == "mark" {
		title = "最高分排行"
	}
	if rank.Character.ID != "" {
		title = rank.Character.Name + map[string]string{"mark": "圣遗物评分"}[rank.Mode] + "排行"
	}
	since := time.UnixMilli(rank.SinceMS).In(chinaTime).Format("01-02 15:04")
	data := map[string]any{"elem": elem, "title": context.Game.Prefix + title, "mode": rank.Mode, "max": rank.Character.ID == "", "since": since, "hash": context.Game.Prefix, "rows": rows}
	return app.Image{Template: "rank", Data: data, Resources: resources.List}, true
}
