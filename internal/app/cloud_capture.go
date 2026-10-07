package app

import (
	"cmp"
	"context"
	"encoding/json"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/RayleaBot/genshin/internal/reference"
)

// miaoFlatAttrs are MysPanelMappings' fixedAttrNames, the substats miao
// reads as flat values; it reads the others as percents.
var miaoFlatAttrs = []string{"hpPlus", "defPlus", "mastery", "atkPlus"}

// A panel keeps its substats' shown or summed values and, when its source
// gives it, how often each was upgraded, not the rolls themselves.
// cloudRolls restores them as miao's MysPanelData does for the official
// panel (getArtifactAttrIdCombination): times is the upgrade count, so the
// substat rolled times+1 times, and of every sequence of that many rolls from
// the rarity's table values of the substat, in table order, the first whose
// sum lies closest to the value wins. The value is read with JavaScript's
// parseFloat, a percent without its last character and times 0.01, as the
// table keeps percents as fractions. Like miao it never gives up on a
// character: only a substat the table lacks, or a value parseFloat cannot
// read, restores no rolls. A times below 0 marks a count the panel does not
// keep, as a panel read back from player data keeps only the sums: every
// count an artifact allows is tried then, the closest sum winning and the
// fewer rolls on a tie.
func cloudRolls(d cloudGearData, rarity string, times int, stat PanelStat) []string {
	var target float64
	if slices.Contains(miaoFlatAttrs, stat.Key) {
		target = jsParseFloat(stat.Value)
	} else {
		value := []rune(stat.Value)
		target = jsParseFloat(string(value[:max(0, len(value)-1)])) * 0.01
	}
	ids := []string{}
	for id, attr := range d.AttrIDMap {
		if rarity != "" && strings.HasPrefix(id, rarity) && attr.Key == stat.Key {
			ids = append(ids, id)
		}
	}
	// The table's keys are integers, which JavaScript lists in ascending
	// order.
	slices.SortFunc(ids, func(a, b string) int { return cmp.Or(len(a)-len(b), strings.Compare(a, b)) })
	// An artifact rolls a substat at most six times: once when it drops and
	// at most five upgrades.
	counts := []int{min(times, 5) + 1}
	if times < 0 {
		counts = []int{1, 2, 3, 4, 5, 6}
	}
	best, found := 1e6, []string{}
	for _, n := range counts {
		chosen := make([]string, n)
		var search func(int, float64)
		search = func(index int, total float64) {
			if index == n {
				// A value parseFloat cannot read is NaN, closer to nothing.
				if diff := math.Abs(target - total); diff < best {
					best, found = diff, slices.Clone(chosen)
				}
				return
			}
			for _, id := range ids {
				chosen[index] = id
				search(index+1, total+d.AttrIDMap[id].Value)
			}
		}
		search(0, 0)
	}
	return found
}

// jsFloatPrefix is the longest start of a text JavaScript's parseFloat reads.
var jsFloatPrefix = regexp.MustCompile(`^[+-]?(Infinity|(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?)`)

// jsParseFloat is JavaScript's parseFloat: the number the text starts with
// after leading white space, NaN when it starts with none.
func jsParseFloat(text string) float64 {
	match := jsFloatPrefix.FindString(strings.TrimLeftFunc(text, func(r rune) bool { return unicode.IsSpace(r) || r == 0xFEFF }))
	if match == "" {
		return math.NaN()
	}
	value, _ := strconv.ParseFloat(strings.Replace(match, "Infinity", "Inf", 1), 64)
	return value
}
func mapCloudStat(key string) string {
	if value := map[string]string{"physical": "phy", "lightning": "elec"}[key]; value != "" {
		return value
	}
	return key
}

var gsCloudMain = map[int]map[string]int{1: {"hpPlus": 14001}, 2: {"atkPlus": 12001}, 3: {"hp": 10002, "atk": 10004, "def": 10006, "recharge": 10007, "mastery": 10008}, 4: {"hp": 15002, "atk": 15004, "def": 15006, "mastery": 15007, "phy": 15015, "pyro": 15008, "electro": 15009, "hydro": 15011, "dendro": 15014, "anemo": 15012, "geo": 15013, "cryo": 15010}, 5: {"hp": 13002, "atk": 13004, "def": 13006, "cpct": 13007, "cdmg": 13008, "heal": 13009, "mastery": 13010}}

// panelCloudGear writes an artifact as miao keeps it in player data: with
// the showcase's own IDs when the artifact keeps them, else with its rolls
// restored from the substats. The official panel and the showcase count each
// substat's upgrades, 0 for one roll; the other sources keep a count only
// above 0. Like miao's
// MysPanelData and EnkaData, it gives nothing for an artifact without its
// main stat, one miao's data does not name or one whose main stat miao cannot
// map, and the character is written without it; the star is the rarity, or 5
// without one.
func panelCloudGear(g Game, gear PanelEquipment, source string) (map[string]any, bool) {
	if len(gear.Main) != 1 {
		return nil, false
	}
	star, _ := strconv.Atoi(gear.Rarity)
	star = cmp.Or(star, 5)
	counted := source == "mihoyo" || source == "enka"
	d := *g.Data.CloudGear
	itemKey := gear.Name
	if _, ok := d.Items[itemKey]; !ok {
		for name, item := range d.Items {
			if item.Set == gear.SetName && item.Slot == gear.Slot {
				itemKey = name
				break
			}
		}
	}

	item, ok := d.Items[itemKey]
	if !ok || item.Slot != gear.Slot {
		return nil, false
	}
	out := map[string]any{"level": gear.Level, "star": star}
	out["name"] = itemKey
	// EnkaData saves the showcase's main stat and roll IDs as they are.
	if gear.MainID != 0 && len(gear.AttrIDs) > 0 {
		attrs := make([]any, len(gear.AttrIDs))
		for index, id := range gear.AttrIDs {
			attrs[index] = id
		}
		out["mainId"], out["attrIds"] = gear.MainID, attrs
		return out, true
	}
	id := gsCloudMain[gear.Slot][gear.Main[0].Key]
	if id == 0 {
		return nil, false
	}
	out["mainId"] = id

	attrs := []any{}
	for _, stat := range gear.Sub {
		times := stat.Times
		if times == 0 && !counted {
			times = -1
		}
		// The official panel's rolls are the table's keys, text.
		for _, id := range cloudRolls(d, gear.Rarity, times, stat) {
			attrs = append(attrs, id)
		}
	}
	out["attrIds"] = attrs
	return out, true
}
func resolveCloudPromotions(ctx context.Context, engine *reference.Engine, record reference.Character, profile BuildProfile) (int, int, error) {
	if profile.Promote != nil && profile.Weapon.Promote != nil {
		return *profile.Promote, *profile.Weapon.Promote, nil
	}
	var input map[string]any
	_ = decodeObject(profile, &input)
	input["resolve_identity"] = true
	raw, err := engine.Run(ctx, record, input)
	if err != nil {
		return 0, 0, gameError("cloud_invalid", "无法确认角色或武器突破阶段，未导出。")
	}
	var out struct {
		Promote int `json:"promote"`
		Weapon  int `json:"weapon_promote"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return 0, 0, gameError("cloud_invalid", "突破阶段还原结果无效。")
	}
	return out.Promote, out.Weapon, nil
}
func originalCloudTalents(record reference.Character, profile BuildProfile) map[string]int {
	out := map[string]int{}
	cons := asObject(record.Data["talentCons"])
	for key, level := range profile.Talents {
		step := 3
		targets := asList(cons[key])
		if targets == nil {
			targets = []any{cons[key]}
		}
		for _, raw := range targets {
			target, ok := cloudInteger(raw, 1, 6)
			if ok && profile.Rank >= target {
				level -= step
			}
		}
		out[key] = max(1, level)
	}
	return out
}

// miaoSources are the _source miao saves a panel under, by the source the
// panel was read from: MysPanelData's mysPanel, EnkaData's enka and
// ProfileChange's change. A panel read back from player data keeps share, as
// it was imported.
var miaoSources = map[string]string{"mihoyo": "mysPanel", "enka": "enka", "change": "change"}

// miaoSource is the _source miao saves the panel under.
func miaoSource(panel CharacterPanel) string {
	return cmp.Or(miaoSources[panel.Source], "share")
}

// panelCloudAvatar writes a panel as miao keeps the character in its player
// data, whatever the panel was read from: the official panel, the showcase,
// or player data read back, under the _source given.
func panelCloudAvatar(ctx context.Context, g Game, panel CharacterPanel, source string) (string, json.RawMessage, error) {
	engine := g.Calc
	if g.Data == nil || g.Data.CloudGear == nil {
		return "", nil, gameError("cloud_invalid", "固定装备资料无法读取。")
	}
	record, err := findBuildCharacter(engine, panel)
	if err != nil {
		return "", nil, err
	}
	profile, err := buildProfile(panel, record)
	if err != nil {
		return "", nil, err
	}
	promote, wp, err := resolveCloudPromotions(ctx, engine, record, profile)
	if err != nil {
		return "", nil, err
	}
	id := panel.ID
	now := time.Now().UnixMilli()
	avatar := map[string]any{"_time": now, "_update": now, "_talent": now, "id": id, "name": record.Name, "elem": record.Element, "level": panel.Level, "promote": promote, "cons": panel.Rank, "talent": originalCloudTalents(record, profile), "trees": profile.Trees, "artis": map[string]any{}}
	if panel.Weapon == nil {
		avatar["weapon"] = nil
	} else {
		weapon := map[string]any{"level": profile.Weapon.Level, "promote": wp, "affix": profile.Weapon.Refinement}
		name := profile.Weapon.Name
		if name == "" {
			metadata := engine.Metadata()
			for _, w := range metadata.Weapons {
				if w.ID == profile.Weapon.ID {
					name = w.Name
					break
				}
			}
		}
		if name == "" {
			return "", nil, gameError("cloud_invalid", "固定资料未收录武器名称。")
		}
		weapon["name"] = name

		avatar["weapon"] = weapon
	}
	for _, gear := range panel.Equipment {
		if converted, ok := panelCloudGear(g, gear, panel.Source); ok {
			avatar["artis"].(map[string]any)[strconv.Itoa(gear.Slot)] = converted
		}
	}
	// miao leaves out artifacts when it keeps none.
	if len(avatar["artis"].(map[string]any)) == 0 {
		delete(avatar, "artis")
	}
	// Normalize through JSON so exported slices use the same representation as imported files.
	var data map[string]any
	_ = decodeObject(avatar, &data)
	return cleanAvatar(data, source)
}
func (a *App) captureCloudPanel(ctx context.Context, client AccountsClient, choice Selection, role Role, archive CloudArchive, id string) (map[string]any, error) {
	if _, err := strconv.Atoi(id); err != nil {
		if entry, ok := a.Catalog.Resolve(id, "character", nil); ok {
			id = entry.ID
		}
	}
	if !cloudIDs([]string{id}, 1, false) {
		return nil, gameError("input_invalid", "请选择角色名称或编号。")
	}
	result, err := client.Execute(ctx, choice, a.Game.ID+".character", map[string]any{"character_ids": []any{id}})
	if err != nil {
		return nil, err
	}
	if result.Role.Ref != role.Ref || result.Role.UID != role.UID || result.Role.Region != role.Region {
		return nil, gameError("role_missing", "角色身份已变化，请重新选择。")
	}
	var panel *CharacterPanel
	for _, p := range NormalizePanels(result, a.Catalog) {
		if p.ID == id {
			copy := p
			panel = &copy
			break
		}
	}
	if panel == nil {
		return nil, gameError("character_missing", "官方未返回此角色。")
	}
	// The archive keeps exchange copies, marked share as imported data.
	exportID, data, err := panelCloudAvatar(ctx, a.Game, *panel, "share")
	if err != nil {
		return nil, err
	}
	if err = a.CloudArchive.Update(client.Provider, choice, archive.Revision, role, map[string]json.RawMessage{exportID: data}, false); err != nil {
		return nil, err
	}
	return map[string]any{"captured": exportID, "note": "已保存官方面板的交换副本。强化档位按显示值还原，存在舍入误差，不代表真实升级顺序；尚未上传云服务。"}, nil
}
