package images

import (
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
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
		row := map[string]any{"place": index + 1, "uid": entry.UID, "star": card["star"], "face": card["face"], "cons": entry.Panel.Rank, "name": rankName(card), "elem": record.Element,
			"talents": rankTalents(cards, record, entry.CharacterID, app.PanelTalents(*entry.Panel, record))}
		if entry.Avatar != "" {
			row["avatar"] = resources.Remote("avatar-"+strconv.Itoa(index), entry.Avatar)
		}
		if weapon := entry.Panel.Weapon; weapon != nil {
			row["weapon"] = rankWeapon(context, cards, cards.weapons[weapon.ID], weapon.Level, weapon.Refinement)
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
	data := map[string]any{"elem": elem, "title": context.Game.Prefix + title, "mode": rank.Mode, "max": rank.Character.ID == "", "time": since, "hash": context.Game.Prefix, "rows": rows, "width": 820}
	return app.Image{Template: "rank", Data: data, Resources: resources.List}, true
}

// CloudRank draws ark-plugin's custom ranking as ark fills miao's
// rank-profile-list: the character's face in each row with the UID ark
// shows, the constellation, talents, weapon, sets and score it returns and
// the ranked detail's average damage; the notes read 全服数据, ark's order
// and filters and the rows asked for, on a page 850 pixels wide.
func CloudRank(context app.ImageContext, rank app.CloudRankImage) (app.Image, bool) {
	if context.Game.Calc == nil || len(rank.Entries) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range rankArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	cards := newAvatarCards(context, resources, nil)
	id := rank.Character.ID
	record := cards.records[id]
	resources.Artwork("common-bg-bg-"+record.Element, "miao-plugin", "resources/common/bg/bg-"+record.Element+".webp")
	resources.Artwork("common-bg-talent-"+record.Element, "miao-plugin", "resources/common/bg/talent-"+record.Element+".webp")
	card := cards.base(id)
	weapons := map[string]reference.Weapon{}
	for _, weapon := range cards.weapons {
		weapons[weapon.Name] = weapon
	}
	rows := []any{}
	for index, entry := range rank.Entries {
		row := map[string]any{"place": index + 1, "uid": entry.UID, "star": card["star"], "face": card["face"], "cons": entry.Cons, "name": rankName(card), "elem": record.Element,
			"talents": rankTalents(cards, record, id, entry.Talents)}
		if weapon, ok := weapons[entry.Weapon]; ok {
			row["weapon"] = rankWeapon(context, cards, weapon, entry.WeaponLevel, entry.WeaponAffix)
		}
		sets := []any{}
		for _, name := range entry.Sets[:min(2, len(entry.Sets))] {
			sets = append(sets, cards.setIcon(name))
		}
		row["sets"], row["set_count"], row["set_name"], row["grade"], row["mark"] = sets, len(sets), entry.SetName, customRankGrade(entry.Mark), miaoComma(entry.Mark)
		if entry.Damage != nil {
			row["damage"] = map[string]any{"title": rank.DamageTitle, "value": miaoComma(*entry.Damage)}
		}
		rows = append(rows, row)
	}
	title := context.Game.Prefix + record.Name + map[string]string{"mark": "圣遗物评分"}[rank.Mode] + "排行"
	data := map[string]any{"elem": record.Element, "title": title, "mode": rank.Mode, "max": false, "time": "全服数据", "hash": context.Game.Prefix, "rows": rows, "width": 850,
		"limit": rank.Limit, "number": rank.Number}
	return app.Image{Template: "rank", Data: data, Resources: resources.List}, true
}

// customRankGrade is ark's grade of a custom ranking row: the average piece
// score against miao's grade steps, MAX past the last.
func customRankGrade(mark float64) string {
	average := mark / 5
	for _, step := range []struct {
		grade string
		below float64
	}{{"D", 7}, {"C", 14}, {"B", 21}, {"A", 28}, {"S", 35}, {"SS", 42}, {"SSS", 49}, {"ACE", 56}, {"MAX", 70}} {
		if average < step.below {
			return step.grade
		}
	}
	return "MAX"
}

// rankName is miao's sName: the abbreviation for names of four or more
// characters.
func rankName(card map[string]any) string {
	name := card["name"].(string)
	if utf8.RuneCountInString(name) >= 4 {
		return card["abbr"].(string)
	}
	return name
}

// rankTalents are a row's talents as miao's rank list draws them: each level
// with its crown and bonus marks on the talent's icon, which is the
// constellation's icon when a constellation raises the talent.
func rankTalents(cards *avatarCards, record reference.Character, id string, levels map[string]app.PanelTalent) []any {
	path, consPath := app.CharacterFolders(id, record.Name, record.Element)
	icons := map[string]string{"a": "resources/common/item/atk-" + record.WeaponType + ".webp", "e": path + "icons/talent-e.webp", "q": path + "icons/talent-q.webp"}
	talentCons, _ := record.Data["talentCons"].(map[string]any)
	for _, key := range []string{"e", "q"} {
		if cons := app.Int(talentCons[key]); cons > 0 {
			icons[key] = consPath + "icons/cons-" + strconv.Itoa(cons) + ".webp"
		}
	}
	talents := []any{}
	for _, key := range []string{"a", "e", "q"} {
		level := levels[key]
		talents = append(talents, map[string]any{"level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10, "icon": cards.miao(icons[key])})
	}
	return talents
}

// rankWeapon is miao's weapon cell: the icon and rarity, the name (its
// abbreviation past four characters), refinement and level.
func rankWeapon(context app.ImageContext, cards *avatarCards, weapon reference.Weapon, level, affix int) map[string]any {
	label, star := weapon.Name, 0
	if entry, ok := context.Catalog.Get(weapon.ID); ok {
		star = entry.Rarity
		if utf8.RuneCountInString(label) > 4 && entry.Abbr != "" {
			label = entry.Abbr
		}
	}
	return map[string]any{"icon": cards.miao("resources/meta-gs/weapon/" + weapon.Type + "/" + weapon.Name + "/icon.webp"), "star": star, "name": label, "affix": affix, "badge": affix + 1, "level": level}
}
