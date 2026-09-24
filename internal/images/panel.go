package images

import (
	"cmp"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
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

// panelSources are miao's names for the services a panel came from; a panel
// ark hands over keeps the name miao gave it there.
var panelSources = map[string]string{"mihoyo": "mysPanel", "enka": "enka", "mihomo": "homo", "change": "面板变换", "share": "share"}

// Panel draws a single character the way miao's profile-detail does: the
// splash with level, constellation, talents and attributes, constellation
// icons, the artifact rating with every substat, the weapon and artifacts,
// and the reference damage table.
func Panel(context app.ImageContext, image app.PanelImage) (app.Image, bool) {
	panel, record := image.Panel, image.Record
	if record.Name == "" {
		return app.Image{}, false
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

	characterPath, iconPath := app.CharacterFolders(panel.ID, record.Name, elem)
	// An uploaded 面板图 the app picked replaces the splash, as miao's.
	if image.Splash.Path != "" {
		resources = append(resources, image.Splash)
	} else {
		add("splash", characterPath+"imgs/splash.webp")
	}
	talentCons, _ := record.Data["talentCons"].(map[string]any)
	talentIcon := map[string]string{"a": "resources/common/item/atk-" + record.WeaponType + ".webp"}
	for _, key := range []string{"e", "q"} {
		if cons := app.Int(talentCons[key]); cons > 0 {
			talentIcon[key] = iconPath + "icons/cons-" + strconv.Itoa(cons) + ".webp"
		} else {
			talentIcon[key] = characterPath + "icons/talent-" + key + ".webp"
		}
	}
	levels := app.PanelTalents(panel, record)
	talents := []any{}
	for _, key := range []string{"a", "e", "q"} {
		level := levels[key]
		add("talent-"+key, talentIcon[key])
		talents = append(talents, map[string]any{"icon": "talent-" + key, "level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10})
	}
	cons := []any{}
	for index := 1; index <= 6; index++ {
		id := "cons-" + strconv.Itoa(index)
		add(id, iconPath+"icons/"+id+".webp")
		cons = append(cons, map[string]any{"icon": id, "off": index > panel.Rank})
	}

	// miao names a panel's source as its data services do, and a changed
	// panel 面板变换.
	updated := context.Now
	if panel.UpdatedAtMS > 0 {
		updated = time.UnixMilli(panel.UpdatedAtMS)
	}
	detail := panel.ScoreDetail
	weight := func(key string) float64 {
		if detail == nil {
			return 0
		}
		return detail.Weights[key]
	}
	stats := map[string]app.PanelStat{}
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
		"mode": "profile", "elem": elem, "name": record.Name, "uid": image.UID, "level": panel.Level, "cons": panel.Rank,
		"talents": talents, "attrs": attrs, "cons_icons": cons,
		"data_source": cmp.Or(panelSources[panel.Source], panel.Source), "update_time": updated.In(chinaTime).Format("2006-01-02 15:04:05"),
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
			var equipment *app.PanelEquipment
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
		// A changed panel compares each result with the kept panel's of the
		// same title, as ark does.
		original := map[string]app.BuildSkillResult{}
		if image.Original != nil {
			for _, result := range image.Original.Baseline.Results {
				if _, seen := original[result.Title]; !seen {
					original[result.Title] = result
				}
			}
		}
		rows := []any{}
		for index, result := range image.Damage.Baseline.Results {
			row := map[string]any{"index": index + 1, "title": result.Title}
			dmg, avg := damageTexts(result)
			if dmg != "" {
				row["dmg"], row["avg"] = dmg, avg
			} else {
				row["full"] = avg
			}
			if old, ok := original[result.Title]; ok {
				oldDmg, oldAvg := damageTexts(old)
				row["dmg_diff"], row["avg_diff"] = damageDiff(dmg, oldDmg), damageDiff(avg, oldAvg)
			}
			rows = append(rows, row)
		}
		if rank := image.Rank; rank != nil {
			// ark adds its ranks as rows of the table.
			for _, row := range rank.Rows {
				rows = append(rows, map[string]any{"index": len(rows) + 1, "title": row.Label, "full": row.Value})
			}
			if chart := rank.Chart; chart != nil {
				data["rank_chart"] = map[string]any{"hint": context.Game.Prefix + abbreviation(context.Catalog, record) + "排名统计", "places": chart.Places,
					"charts": []any{panelRankChart("damage", chart.Damage), panelRankChart("artis", chart.Artis)}}
			}
		}
		// ark's DealLongDmgTitle for a changed panel, over every row with its
		// own: 1 cuts a title past 20 widths, 2 lets them wrap.
		wrap := false
		for _, row := range rows {
			item := row.(map[string]any)
			title := strings.TrimSpace(item["title"].(string))
			switch image.LongTitles {
			case 1:
				item["title"] = cutDamageTitle(title)
			case 2:
				wrap = wrap || damageTitleWidth(title) > 20
			}
		}
		baseline := image.Damage.Baseline
		enemy := baseline.EnemyName
		if enemy == "" {
			enemy = "小宝"
		}
		data["damage"] = map[string]any{"rows": rows, "enemy_level": image.Damage.EnemyLevel, "enemy_name": enemy, "created_by": baseline.CreatedBy,
			"enemy_hint": context.Game.Prefix + "敌人等级" + strconv.Itoa(image.Damage.EnemyLevel), "wrap": wrap}
		if image.DamageMode {
			data["mode"] = "dmg"
			if matrix := baseline.Matrix; matrix != nil {
				data["matrix"] = damageMatrixData(*matrix)
				// The buffs are those of the detail the trades are for.
				buffs := []any{}
				if detail := matrix.Index - 1; detail >= 0 && detail < len(baseline.Results) {
					for _, buff := range baseline.Results[detail].Buffs {
						title, text, _ := strings.Cut(strings.Replace(buff, ":", "：", 1), "：")
						buffs = append(buffs, map[string]any{"title": title, "text": text})
					}
				}
				data["buffs"] = buffs
			}
		}
	}
	if image.Change != "" {
		data["change"] = image.Change
	}
	return app.Image{Template: "panel", Data: data, Resources: resources}, true
}

// damageTexts are a result's critical and average damage as the table shows
// them; a result with one number or a text shows it as the average.
func damageTexts(result app.BuildSkillResult) (string, string) {
	switch {
	case result.Critical != nil && result.Expected != nil:
		return comma(*result.Critical), comma(*result.Expected)
	case result.Expected != nil:
		return "", comma(*result.Expected)
	}
	return "", result.Text
}

// damageNumber reads a shown damage as ark's getDiff does: without its
// thousands separators, the number it starts with, NaN when none.
var damageNumber = regexp.MustCompile(`^\s*[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`)

// damageDiff is ark's getDiff: the change from the kept panel's value in
// percent to one place, with an up or down arrow, or -- when either value is
// not a number or the kept one is 0. Upstream compares the fixed text with
// the number 0, so an unchanged value reads " ↓0%"; it reads 0.0% here, as
// meant.
func damageDiff(current, original string) map[string]any {
	read := func(text string) float64 {
		number, err := strconv.ParseFloat(damageNumber.FindString(strings.ReplaceAll(text, ",", "")), 64)
		if err != nil {
			return math.NaN()
		}
		return number
	}
	now, was := read(current), read(original)
	if math.IsNaN(now) || math.IsNaN(was) || was == 0 {
		return map[string]any{"text": "--", "class": "same"}
	}
	change := math.Round((now-was)/was*1000) / 10
	switch {
	case change > 0:
		return map[string]any{"text": " ↑" + strconv.FormatFloat(change, 'f', -1, 64) + "%", "class": "up"}
	case change < 0:
		return map[string]any{"text": " ↓" + strconv.FormatFloat(-change, 'f', -1, 64) + "%", "class": "down"}
	}
	return map[string]any{"text": "0.0%", "class": "same"}
}

// damageTitleWidth is how wide ark counts a title: 1, ( and ) take half.
func damageTitleWidth(title string) float64 {
	width := 0.0
	for _, char := range title {
		width += damageCharWidth(char)
	}
	return width
}

func damageCharWidth(char rune) float64 {
	if char == '1' || char == '(' || char == ')' {
		return 0.5
	}
	return 1
}

// cutDamageTitle cuts a title before the character that takes it past 20
// widths and adds "...", as ark's truncTitle.
func cutDamageTitle(title string) string {
	width := 0.0
	for index, char := range title {
		if width += damageCharWidth(char); width > 20 {
			return title[:index] + "..."
		}
	}
	return title
}

// panelRankXs are the x of the points ark draws in 排名统计: the bottom, the
// ranking percentiles from the lowest (100 - percentile) and the top.
var panelRankXs = []float64{0, 1, 5, 10, 30, 50, 70, 80, 90, 95, 99, 100}

// rankChart lays out one of ark-plugin's 排名统计 charts, "damage" or
// "artis", as its ECharts option draws it: the distribution with the x
// labels reading 100 - x, and the y axis named by the kind, up to 100 by 20
// for damage and to the top artifact score by 50 (ark's parseInt(top1) / 50 *
// 50, which is parseInt(top1)); at places a point on it. A curve ark did not
// send draws 暂无数据, and at is nil.
func rankChart(kind string, curve *app.RankCurve) (chart map[string]any, at func(x, y float64) (float64, float64)) {
	if curve == nil {
		return map[string]any{}, nil
	}
	name, yMax, interval := "伤害评分", 100.0, 20.0
	if kind == "artis" {
		name, yMax, interval = "圣遗物评分", math.Trunc(curve.Top), 50
	}
	values := []float64{0}
	for index := len(curve.Scores) - 1; index >= 0; index-- {
		values = append(values, curve.Scores[index])
	}
	values = append(values, curve.Top)
	chart, at = app.EChartsDistribution(panelRankXs, values, yMax, interval)
	left, top := at(0, yMax)
	right, bottom := at(100, 0)
	for _, tick := range chart["x_ticks"].([]any) {
		item := tick.(map[string]any)
		item["label"], item["y"] = strconv.FormatFloat(100-item["value"].(float64), 'f', -1, 64), bottom+8
	}
	for _, tick := range chart["y_ticks"].([]any) {
		item := tick.(map[string]any)
		item["label"], item["x"] = strconv.FormatFloat(item["value"].(float64), 'f', -1, 64), left-8
	}
	chart["x_name"] = map[string]any{"x": right + 15, "y": bottom}
	chart["y_name"] = map[string]any{"x": left, "y": top - 15, "text": name}
	chart["id"] = "rank-" + kind
	return chart, at
}

// rankFill is the colour of the area under a 排名统计 curve, and rankMark
// the colour it turns at the panel's place.
var rankFill, rankMark = map[string]any{"color": "rgb(51, 204, 204)", "opacity": 0.5}, map[string]any{"color": "rgb(255, 0, 0)", "opacity": 1}

// rankStop is a stop of the area's gradient: its offset and colour.
type rankStop struct {
	offset float64
	color  map[string]any
}

func rankStops(stops ...rankStop) []any {
	out := []any{}
	for _, stop := range stops {
		out = append(out, map[string]any{"offset": stop.offset, "color": stop.color["color"], "opacity": stop.color["opacity"]})
	}
	return out
}

// panelRankChart is a 排名统计 chart with the panel marked as ark's option
// marks it: its score written 15 right of and 5 above its place, and the
// area's gradient, which turns red from the panel's place for 2.5 points
// (ark's stops fall out of order, and later stops cannot go back).
func panelRankChart(kind string, curve *app.RankCurve) map[string]any {
	chart, at := rankChart(kind, curve)
	if at == nil {
		return chart
	}
	x, y := at(100-curve.Percent, curve.Score)
	chart["mark"] = map[string]any{"x": x + 15, "y": y - 5, "text": strconv.FormatFloat(curve.Score, 'f', 2, 64)}
	place := (100 - curve.Percent) / 100
	chart["stops"] = rankStops(rankStop{0, rankFill}, rankStop{place, rankFill}, rankStop{place - 0.025, rankMark}, rankStop{place + 0.025, rankMark}, rankStop{place + 0.025, rankFill}, rankStop{1, rankFill})
	return chart
}

// damageMatrixData is miao's 词条伤害计算 as ProfileDetail formats it: each
// trade's change of the average, and its average and critical damage.
func damageMatrixData(matrix app.DamageMatrix) map[string]any {
	critical := func(value *float64) string {
		if value == nil {
			return ""
		}
		return comma(*value)
	}
	rows := []any{}
	for index, row := range matrix.Rows {
		cells := []any{}
		for _, cell := range row {
			if cell.Type == "na" {
				cells = append(cells, map[string]any{"type": "na"})
				continue
			}
			change := comma(cell.Avg - matrix.Avg)
			if cell.Avg > matrix.Avg {
				change = "+" + change
			}
			cells = append(cells, map[string]any{"type": cell.Type, "val": change, "avg": comma(cell.Avg), "dmg": critical(cell.Dmg)})
		}
		rows = append(rows, map[string]any{"title": matrix.Attrs[index].Title, "text": matrix.Attrs[index].Text, "cells": cells})
	}
	return map[string]any{"index": matrix.Index, "title": matrix.Title, "avg": comma(matrix.Avg), "dmg": critical(matrix.Dmg), "attrs": matrix.Attrs, "rows": rows}
}

// weaponData is the weapon card: miao's icon, the base attack and substat,
// and the refinement text with this refinement's values.
func weaponData(context app.ImageContext, weapon *app.PanelEquipment, add func(id, name string) bool) map[string]any {
	result := map[string]any{"name": weapon.Name, "affix": weapon.Refinement, "level": weapon.Level}
	attrs := []any{}
	for index, stat := range append(append([]app.PanelStat{}, weapon.Main...), weapon.Sub...) {
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
		text := app.Text(affix["text"])
		// Upstream wraps each refinement value in <nobr>; the template does
		// the same with these text and value pieces.
		parts := []any{}
		last := 0
		for _, match := range affixPlaceholder.FindAllStringSubmatchIndex(text, -1) {
			series, _ := values[text[match[2]:match[3]]].([]any)
			value := ""
			if index := weapon.Refinement - 1; index >= 0 && index < len(series) {
				value = app.Text(series[index])
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
func abbreviation(catalog app.Catalog, record reference.Character) string {
	if entry, ok := catalog.Get(record.ID); ok && entry.Abbr != "" {
		return entry.Abbr
	}
	if abbr := app.Text(record.Data["abbr"]); abbr != "" {
		return abbr
	}
	return record.Name
}
