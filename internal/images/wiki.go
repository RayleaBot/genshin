package images

import (
	"encoding/json"
	"html"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// wikiArtwork maps the images the talent page's converted stylesheets name
// to miao's paths.
var wikiArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"wiki-imgs-card-bg", "resources/wiki/imgs/card-bg.png"},
}

// wikiGrowth names the ascension stat, as miao's getLineData does.
var wikiGrowth = map[string]string{"atkPct": "大攻击", "hpPct": "大生命", "defPct": "大防御", "cpct": "暴击", "cdmg": "爆伤", "recharge": "充能", "mastery": "精通", "heal": "治疗", "phy": "物伤"}

// wikiCharacter is the part of miao's character data.json the pages read.
type wikiCharacter struct {
	ID                                                                          int
	Name, Abbr, Title, Desc, Elem, Weapon, Allegiance, Birth, Astro, Cncv, Jpcv string
	Star                                                                        int
	Materials                                                                   map[string]string
	BaseAttr                                                                    map[string]float64
	GrowAttr                                                                    struct {
		Key   string
		Value float64
	}
	TalentCons map[string]int
	Talent     map[string]wikiTalent
	Cons       map[string]wikiTalent
	Passive    []wikiTalent
}

type wikiTalent struct {
	Name   string
	Desc   []string
	Tables []struct {
		Name, Unit string
		IsSame     bool
		Values     []string
	}
}

// Entry draws the reference pages miao draws for a character: 天赋 and 命座
// as wiki/character-talent, 图鉴 and 资料 as wiki/character-wiki. The pages
// read miao's character data, which comes with the downloaded miao artwork.
func Entry(context app.ImageContext, page app.EntryImage) (app.Image, bool) {
	if page.Command != "talent-wiki" && page.Command != "catalog" || page.Entry.Kind != "character" || context.Artwork == nil || traveler(page.Entry.ID) {
		return app.Image{}, false
	}
	raw, err := context.Artwork.Open("miao-plugin", "resources/meta-gs/character/"+page.Entry.Name+"/data.json")
	if err != nil {
		return app.Image{}, false
	}
	var character wikiCharacter
	if json.Unmarshal(raw, &character) != nil || len(character.Talent) == 0 {
		return app.Image{}, false
	}
	if page.Command == "catalog" {
		return CharacterWiki(context, character)
	}
	return CharacterTalent(context, character, strings.Contains(page.Word, "命"))
}

// CharacterTalent draws miao's wiki/character-talent: the card, title, name
// and description with the level-100 base stats and ascension stat, then
// either every talent with its description and its Lv6 to Lv13 table and the
// passives, or the six constellations.
func CharacterTalent(context app.ImageContext, character wikiCharacter, cons bool) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range wikiArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	base := "resources/meta-gs/character/" + character.Name + "/"
	image := func(id, path string) string { return resources.Artwork(id, "miao-plugin", path) }
	line := []any{}
	for _, stat := range [][2]string{{"hp", "基础生命"}, {"atk", "基础攻击"}, {"def", "基础防御"}} {
		line = append(line, map[string]any{"num": miaoComma(character.BaseAttr[stat[0]], 1), "label": stat[1]})
	}
	growth := strconv.FormatFloat(character.GrowAttr.Value, 'f', -1, 64)
	if character.GrowAttr.Key == "mastery" {
		growth = miaoComma(character.GrowAttr.Value, 1)
	}
	label := wikiGrowth[character.GrowAttr.Key]
	if character.GrowAttr.Key == "dmg" {
		label = map[string]string{"pyro": "火", "hydro": "水", "anemo": "风", "electro": "雷", "dendro": "草", "cryo": "冰", "geo": "岩"}[character.Elem] + "伤"
	}
	line = append(line, map[string]any{"num": growth, "label": "成长·" + label})
	icons := map[string]string{"a": image("atk-"+character.Weapon, "resources/common/item/atk-"+character.Weapon+".webp")}
	for index := 1; index <= 6; index++ {
		icons["cons"+strconv.Itoa(index)] = image("cons-"+strconv.Itoa(index), base+"icons/cons-"+strconv.Itoa(index)+".webp")
	}
	for index := 0; index <= 4; index++ {
		icons["passive"+strconv.Itoa(index)] = image("passive-"+strconv.Itoa(index), base+"icons/passive-"+strconv.Itoa(index)+".webp")
	}
	for _, key := range []string{"e", "q"} {
		if level := character.TalentCons[key]; level > 0 {
			icons[key] = icons["cons"+strconv.Itoa(level)]
		} else {
			icons[key] = image("talent-"+key, base+"icons/talent-"+key+".webp")
		}
	}
	data := map[string]any{"elem": character.Elem, "name": character.Title + "·" + character.Name, "desc": miaoRichText(character.Desc), "line": line, "cons": cons,
		"card": image("card", base+"imgs/card.webp"), "face": image("face", base+"imgs/face-q.webp")}
	if data["face"] == "" {
		data["face"] = image("face", base+"imgs/face.webp")
	}
	if cons {
		list := []any{}
		for index := 1; index <= 6; index++ {
			if talent, ok := character.Cons[strconv.Itoa(index)]; ok {
				list = append(list, wikiDetail(talent, icons["cons"+strconv.Itoa(index)]))
			}
		}
		data["constellations"] = list
	} else {
		talents := []any{}
		for _, key := range []string{"a", "e", "q"} {
			talent, ok := character.Talent[key]
			if !ok {
				continue
			}
			parts := []any{wikiDetail(talent, icons[key])}
			for _, suffix := range []string{"1", "2"} {
				if extra, ok := character.Talent[key+suffix]; ok {
					parts = append(parts, wikiDetail(extra, icons[key+suffix]))
				}
			}
			_, merged := character.Talent[key+"2"]
			talents = append(talents, map[string]any{"merged": merged, "parts": parts})
		}
		passives := []any{}
		for index, passive := range character.Passive {
			passives = append(passives, wikiDetail(passive, icons["passive"+strconv.Itoa(index)]))
		}
		data["talents"], data["passives"] = talents, passives
	}
	return app.Image{Template: "character-talent", Data: data, Resources: resources.List}, true
}

// wikiDetail is miao's talent-detail block: the icon, name and description,
// the values every level shares, and the Lv6 to Lv13 table.
func wikiDetail(talent wikiTalent, icon string) map[string]any {
	var desc strings.Builder
	for index, line := range talent.Desc {
		desc.WriteString(miaoRichText(line))
		heading := (strings.HasPrefix(line, "<h3>") && strings.HasSuffix(line, "</h3>")) || (strings.HasPrefix(line, "<i>") && strings.HasSuffix(line, "</i>"))
		if index < len(talent.Desc)-1 && !heading {
			desc.WriteString("<br>")
		}
	}
	shared, rows := []any{}, []any{}
	for _, table := range talent.Tables {
		if table.IsSame {
			value := ""
			if len(table.Values) > 0 {
				value = table.Values[0]
			}
			shared = append(shared, map[string]any{"name": table.Name, "unit": table.Unit, "value": value})
			continue
		}
		rows = append(rows, map[string]any{"name": table.Name, "unit": table.Unit, "values": table.Values[min(5, len(table.Values)):min(13, len(table.Values))]})
	}
	levels := []string{}
	for level := 6; level <= 13; level++ {
		levels = append(levels, "Lv"+strconv.Itoa(level))
	}
	return map[string]any{"icon": icon, "name": talent.Name, "desc": desc.String(), "tables": len(talent.Tables) > 0, "shared": shared, "rows": rows, "levels": levels}
}

// jsFixed is JavaScript's toFixed: the exact value rounded half up.
func jsFixed(value float64, digits int) string {
	scaled := new(big.Float).SetPrec(256).SetFloat64(value)
	scaled.Mul(scaled, new(big.Float).SetPrec(256).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)))
	scaled.Add(scaled, new(big.Float).SetPrec(256).SetFloat64(0.5))
	whole, _ := scaled.Int(nil)
	text := whole.String()
	if digits == 0 {
		return text
	}
	for len(text) <= digits {
		text = "0" + text
	}
	return text[:len(text)-digits] + "." + text[len(text)-digits:]
}

// miaoComma is miao's Format.comma: toFixed decimals with thousands commas.
func miaoComma(value float64, digits int) string {
	fixed, _ := strconv.ParseFloat(jsFixed(value, digits), 64)
	integer, decimal, _ := strings.Cut(strconv.FormatFloat(fixed, 'f', -1, 64), ".")
	for index := len(integer) - 3; index > 0; index -= 3 {
		integer = integer[:index] + "," + integer[index:]
	}
	if digits > 0 {
		if decimal == "" {
			decimal = strings.Repeat("0", digits)
		}
		return integer + "." + decimal
	}
	return integer
}

var (
	miaoTag   = regexp.MustCompile(`<[^>]*>`)
	miaoColor = regexp.MustCompile(`^<span style="color:\s*(#[0-9A-Fa-f]{3,8});?">$`)
)

// miaoRichText is a line of miao's description HTML, which upstream prints
// as is, keeping only its headings, line breaks, italics, bold and colours:
// everything else is escaped or dropped.
func miaoRichText(text string) string {
	var out strings.Builder
	last := 0
	for _, bounds := range miaoTag.FindAllStringIndex(text, -1) {
		out.WriteString(html.EscapeString(text[last:bounds[0]]))
		last = bounds[1]
		tag := text[bounds[0]:bounds[1]]
		switch lower := strings.ToLower(tag); lower {
		case "<br>", "<br/>", "<br />":
			out.WriteString("<br>")
		case "<h3>", "</h3>", "<i>", "</i>", "<b>", "</b>", "</span>":
			out.WriteString(lower)
		default:
			if match := miaoColor.FindStringSubmatch(tag); match != nil {
				out.WriteString(`<span style="color:` + match[1] + `">`)
			}
		}
	}
	out.WriteString(html.EscapeString(text[last:]))
	return out.String()
}
