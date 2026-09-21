package images

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/reference"
)

// panelArtwork maps the miao images named in the panel stylesheet to their
// repository paths; the stylesheet reads them as --render-resource-<id>.
var panelArtwork = [][2]string{
	{"character-imgs-crown", "resources/character/imgs/crown.png"},
	{"character-imgs-icon", "resources/character/imgs/icon.png"},
	{"character-imgs-up-num-icon0", "resources/character/imgs/up-num-icon0.png"},
	{"character-imgs-up-num-icon1", "resources/character/imgs/up-num-icon1.png"},
	{"character-imgs-up-num-icon2", "resources/character/imgs/up-num-icon2.png"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg1-o", "resources/common/item/bg1-o.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg2-o", "resources/common/item/bg2-o.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg3-o", "resources/common/item/bg3-o.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg4-o", "resources/common/item/bg4-o.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"common-item-bg5-o", "resources/common/item/bg5-o.png"},
	{"common-item-star", "resources/common/item/star.png"},
	// miao's own font families.
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
}

// panelAttrs are the attribute rows of miao's Genshin panel, with the
// official property IDs they read.
var panelAttrs = []struct{ key, title, id string }{
	{"hp", "生命值", "2000"}, {"atk", "攻击力", "2001"}, {"def", "防御力", "2002"}, {"mastery", "元素精通", "28"},
	{"cpct", "暴击率", "20"}, {"cdmg", "暴击伤害", "22"}, {"recharge", "元素充能", "23"}, {"dmg", "伤害加成", ""},
}

var elementDamageIDs = map[string]string{"pyro": "40", "electro": "41", "hydro": "42", "dendro": "43", "anemo": "44", "geo": "45", "cryo": "46"}

// weaponAttrTitles are miao's short names for weapon substats.
var weaponAttrTitles = map[string]string{"atk": "攻击", "mastery": "精通", "hp": "生命", "def": "防御", "cpct": "暴击", "cdmg": "爆伤", "phy": "物伤", "recharge": "充能", "heal": "治疗"}

// shortTitles shortens the combined substat list as miao does.
var shortTitles = map[string]string{"暴击率": "暴击", "暴击伤害": "爆伤", "充能效率": "充能", "元素精通": "精通", "大生命": "生命", "大攻击": "攻击", "大防御": "防御", "小生命": "生命", "小攻击": "攻击", "小防御": "防御"}

var affixPlaceholder = regexp.MustCompile(`\$\[(\d)\]`)

// chinaTime is the time zone the upstream images print times in.
var chinaTime = time.FixedZone("UTC+8", 8*3600)

// Panel draws a single character the way miao's profile-detail does: the
// splash with level, constellation, talents and attributes, constellation
// icons, the artifact rating with every substat, the weapon and artifacts,
// and the reference damage table.
func Panel(context gamekit.ImageContext, image gamekit.PanelImage) (gamekit.Image, bool) {
	panel, record := image.Panel, image.Record
	if record.Name == "" {
		return gamekit.Image{}, false
	}
	resources := []rayleabot.RenderImageResource{}
	add := func(id, name string) bool {
		resource, ok := context.ArtworkResource(id, "miao-plugin", name)
		if ok {
			resources = append(resources, resource)
		}
		return ok
	}
	for _, item := range panelArtwork {
		add(item[0], item[1])
	}
	elem := record.Element
	add("common-bg-bg-"+elem, "resources/common/bg/bg-"+elem+".webp")
	add("common-bg-talent-"+elem, "resources/common/bg/talent-"+elem+".webp")

	characterPath := "resources/meta-gs/character/" + record.Name + "/"
	add("splash", characterPath+"imgs/splash.webp")
	talentCons, _ := record.Data["talentCons"].(map[string]any)
	talentIcon := map[string]string{"a": "resources/common/item/atk-" + record.WeaponType + ".webp"}
	for _, key := range []string{"e", "q"} {
		if cons := gamekit.Int(talentCons[key]); cons > 0 {
			talentIcon[key] = characterPath + "icons/cons-" + strconv.Itoa(cons) + ".webp"
		} else {
			talentIcon[key] = characterPath + "icons/talent-" + key + ".webp"
		}
	}
	levels := gamekit.PanelTalents("genshin", panel, record)
	talents := []any{}
	for _, key := range []string{"a", "e", "q"} {
		level := levels[key]
		add("talent-"+key, talentIcon[key])
		talents = append(talents, map[string]any{"icon": "talent-" + key, "level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10})
	}
	cons := []any{}
	for index := 1; index <= 6; index++ {
		id := "cons-" + strconv.Itoa(index)
		add(id, characterPath+"icons/"+id+".webp")
		cons = append(cons, map[string]any{"icon": id, "off": index > panel.Rank})
	}

	detail := panel.ScoreDetail
	weight := func(key string) float64 {
		if detail == nil {
			return 0
		}
		return detail.Weights[key]
	}
	stats := map[string]gamekit.PanelStat{}
	for _, stat := range panel.Stats {
		stats[stat.ID] = stat
	}
	attrs := []any{}
	for _, row := range panelAttrs {
		id := row.id
		if row.key == "dmg" {
			id = elementDamageIDs[elem]
		}
		stat := stats[id]
		value, base, plus := statText(stat.Value), statText(stat.Base), statText(stat.Added)
		if row.key == "dmg" && base == "" {
			base = "0.0%"
		}
		item := map[string]any{"key": row.key, "title": row.title, "value": value, "base": base, "plus": plus, "zero": base == "0.0%"}
		if w := weight(row.key); w > 0 {
			item["weight"] = strconv.FormatFloat(w, 'f', -1, 64)
			item["gold"] = w >= 80
		}
		attrs = append(attrs, item)
	}

	data := map[string]any{
		"elem": elem, "name": record.Name, "uid": image.UID, "level": panel.Level, "cons": panel.Rank,
		"talents": talents, "attrs": attrs, "cons_icons": cons,
		"data_source": "mys", "update_time": context.Now.In(chinaTime).Format("2006-01-02 15:04:05"),
		"artifact_hint": context.Game.Prefix + abbreviation(context.Catalog, record) + "圣遗物",
		"damage_hint":   context.Game.Prefix + abbreviation(context.Catalog, record) + "伤害",
	}
	if detail != nil {
		class := func(key string) string {
			switch w := weight(key); {
			case w > 79.9:
				return "great"
			case w > 0:
				return "useful"
			}
			return "nouse"
		}
		title := func(key string) string { return detail.Titles[key] }
		all := []any{}
		for _, attr := range detail.AllAttrs {
			name := title(attr.Key)
			if short := shortTitles[name]; short != "" {
				name = short
			}
			all = append(all, map[string]any{"class": class(attr.Key), "eff": attr.Eff, "title": name, "value": attr.Value})
		}
		// Upstream shows exactly nine cells: the nine best, padded when fewer.
		all = all[:min(len(all), 9)]
		for len(all) < 9 {
			all = append(all, map[string]any{})
		}
		data["rating"] = map[string]any{"mark": detail.Mark, "grade": detail.Grade, "rule": detail.Title, "all": all}
		pieces := []any{}
		for slot := 1; slot <= 5; slot++ {
			piece, scored := detail.Pieces[slot]
			var equipment *gamekit.PanelEquipment
			for index := range panel.Equipment {
				if panel.Equipment[index].Slot == slot {
					equipment = &panel.Equipment[index]
				}
			}
			if !scored || equipment == nil {
				pieces = append(pieces, map[string]any{"empty": true})
				continue
			}
			id := "artifact-" + strconv.Itoa(slot)
			add(id, "resources/meta-gs/artifact/imgs/"+equipment.SetName+"/"+strconv.Itoa(slot)+".webp")
			subs := []any{}
			for _, attr := range piece.Attrs {
				subs = append(subs, map[string]any{"class": class(attr.Key), "eff": attr.Eff, "up": attr.UpNum, "title": title(attr.Key), "value": attr.Value})
			}
			pieces = append(pieces, map[string]any{"icon": id, "name": equipment.Name, "level": equipment.Level, "mark": piece.Mark, "grade": piece.Grade,
				"main": map[string]any{"title": title(piece.Main.Key), "value": piece.Main.Value}, "attrs": subs})
		}
		data["artifacts"] = pieces
	}
	if weapon := panel.Weapon; weapon != nil {
		data["weapon"] = weaponData(context, weapon, add)
	}
	if image.Damage != nil {
		rows := []any{}
		for index, result := range image.Damage.Baseline.Results {
			row := map[string]any{"index": index + 1, "title": result.Title}
			switch {
			case result.Critical != nil && result.Expected != nil:
				row["dmg"], row["avg"] = comma(*result.Critical), comma(*result.Expected)
			case result.Expected != nil:
				row["full"] = comma(*result.Expected)
			default:
				row["full"] = result.Text
			}
			rows = append(rows, row)
		}
		data["damage"] = map[string]any{"rows": rows, "enemy_level": image.Damage.EnemyLevel}
	}
	return gamekit.Image{Template: "panel", Data: data, Resources: resources}, true
}

// weaponData is the weapon card: miao's icon, the base attack and substat,
// and the refinement text with this refinement's values.
func weaponData(context gamekit.ImageContext, weapon *gamekit.PanelEquipment, add func(id, name string) bool) map[string]any {
	result := map[string]any{"name": weapon.Name, "affix": weapon.Refinement, "level": weapon.Level}
	attrs := []any{}
	for index, stat := range append(append([]gamekit.PanelStat{}, weapon.Main...), weapon.Sub...) {
		title := "攻击"
		if index > 0 {
			key := strings.TrimSuffix(stat.Key, "Plus")
			if title = weaponAttrTitles[key]; title == "" {
				title = "伤害"
			}
		}
		attrs = append(attrs, map[string]any{"title": title, "value": statText(stat.Value)})
	}
	result["attrs"] = attrs
	if context.Game.Calc == nil {
		return result
	}
	for _, entry := range context.Game.Calc.Metadata().Weapons {
		if entry.ID != weapon.ID {
			continue
		}
		add("weapon-icon", "resources/meta-gs/weapon/"+entry.Type+"/"+entry.Name+"/icon.webp")
		affix, _ := entry.Data["affixData"].(map[string]any)
		values, _ := affix["datas"].(map[string]any)
		text := gamekit.Text(affix["text"])
		// Upstream wraps each refinement value in <nobr>; the template does
		// the same with these text and value pieces.
		parts := []any{}
		last := 0
		for _, match := range affixPlaceholder.FindAllStringSubmatchIndex(text, -1) {
			series, _ := values[text[match[2]:match[3]]].([]any)
			value := ""
			if index := weapon.Refinement - 1; index >= 0 && index < len(series) {
				value = gamekit.Text(series[index])
			}
			parts = append(parts, map[string]any{"text": text[last:match[0]], "value": value})
			last = match[1]
		}
		parts = append(parts, map[string]any{"text": text[last:], "value": ""})
		result["desc"] = parts
	}
	return result
}

// statText shows official numbers the way miao does: thousands separators on
// whole numbers, percentages unchanged.
func statText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasSuffix(value, "%") {
		return value
	}
	number, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	if err != nil {
		return value
	}
	return comma(number)
}

// comma formats a whole number with thousands separators, as miao's
// Format.comma does without decimals.
func comma(value float64) string {
	digits := strconv.FormatFloat(value, 'f', 0, 64)
	negative := strings.HasPrefix(digits, "-")
	digits = strings.TrimPrefix(digits, "-")
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}

// abbreviation is the short name upstream uses in command hints and cards:
// miao's abbreviation when the name has one, otherwise the full name, as
// miao's character model does.
func abbreviation(catalog gamekit.Catalog, record reference.Character) string {
	if entry, ok := catalog.Get(record.ID); ok && entry.Abbr != "" {
		return entry.Abbr
	}
	if abbr := gamekit.Text(record.Data["abbr"]); abbr != "" {
		return abbr
	}
	return record.Name
}
