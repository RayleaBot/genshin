// Package showcase reads Genshin Impact panels from Enka, the showcase service
// miao-plugin refreshes from without an account.
package showcase

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

// Source is Enka's Genshin Impact API.
var Source = app.ShowcaseSource{Name: "Enka", URL: func(uid string) string { return "https://enka.network/api/uid/" + uid }, Parse: Parse}

type enkaAnswer struct {
	TTL        int `json:"ttl"`
	PlayerInfo *struct {
		Nickname string `json:"nickname"`
		Level    int    `json:"level"`
	} `json:"playerInfo"`
	AvatarInfoList []enkaAvatar `json:"avatarInfoList"`
}

type enkaAvatar struct {
	AvatarID      int                          `json:"avatarId"`
	PropMap       map[string]struct{ Val any } `json:"propMap"`
	TalentIDList  []int                        `json:"talentIdList"`
	FightPropMap  map[string]float64           `json:"fightPropMap"`
	SkillLevelMap map[string]int               `json:"skillLevelMap"`
	EquipList     []struct {
		ItemID    int `json:"itemId"`
		Reliquary *struct {
			Level            int   `json:"level"`
			AppendPropIDList []int `json:"appendPropIdList"`
		} `json:"reliquary"`
		Weapon *struct {
			Level        int            `json:"level"`
			PromoteLevel int            `json:"promoteLevel"`
			AffixMap     map[string]int `json:"affixMap"`
		} `json:"weapon"`
		Flat struct {
			ItemType          string     `json:"itemType"`
			EquipType         string     `json:"equipType"`
			RankLevel         int        `json:"rankLevel"`
			ReliquaryMainstat *enkaStat  `json:"reliquaryMainstat"`
			ReliquarySubstats []enkaStat `json:"reliquarySubstats"`
			WeaponStats       []enkaStat `json:"weaponStats"`
		} `json:"flat"`
	} `json:"equipList"`
}

type enkaStat struct {
	MainPropID   string  `json:"mainPropId"`
	AppendPropID string  `json:"appendPropId"`
	StatValue    float64 `json:"statValue"`
}

// fightProps are the game's property IDs, which the official panel uses as
// property types, by Enka's names; percent marks values shown with %.
var fightProps = map[string]struct {
	id      string
	name    string
	percent bool
}{
	"FIGHT_PROP_BASE_ATTACK": {"4", "基础攻击力", false}, "FIGHT_PROP_HP": {"2", "生命值", false}, "FIGHT_PROP_HP_PERCENT": {"3", "生命值", true},
	"FIGHT_PROP_ATTACK": {"5", "攻击力", false}, "FIGHT_PROP_ATTACK_PERCENT": {"6", "攻击力", true}, "FIGHT_PROP_DEFENSE": {"8", "防御力", false},
	"FIGHT_PROP_DEFENSE_PERCENT": {"9", "防御力", true}, "FIGHT_PROP_CRITICAL": {"20", "暴击率", true}, "FIGHT_PROP_CRITICAL_HURT": {"22", "暴击伤害", true},
	"FIGHT_PROP_CHARGE_EFFICIENCY": {"23", "元素充能效率", true}, "FIGHT_PROP_HEAL_ADD": {"26", "治疗加成", true}, "FIGHT_PROP_ELEMENT_MASTERY": {"28", "元素精通", false},
	"FIGHT_PROP_PHYSICAL_ADD_HURT": {"30", "物理伤害加成", true}, "FIGHT_PROP_FIRE_ADD_HURT": {"40", "火元素伤害加成", true}, "FIGHT_PROP_ELEC_ADD_HURT": {"41", "雷元素伤害加成", true},
	"FIGHT_PROP_WATER_ADD_HURT": {"42", "水元素伤害加成", true}, "FIGHT_PROP_GRASS_ADD_HURT": {"43", "草元素伤害加成", true}, "FIGHT_PROP_WIND_ADD_HURT": {"44", "风元素伤害加成", true},
	"FIGHT_PROP_ROCK_ADD_HURT": {"45", "岩元素伤害加成", true}, "FIGHT_PROP_ICE_ADD_HURT": {"46", "冰元素伤害加成", true},
}

// statKeys are miao's attribute keys by property ID, as the official panel
// reading gives them.
var statKeys = map[string]string{"2": "hpPlus", "3": "hp", "5": "atkPlus", "6": "atk", "8": "defPlus", "9": "def", "20": "cpct", "22": "cdmg", "23": "recharge", "26": "heal", "28": "mastery", "30": "phy", "40": "pyro", "41": "electro", "42": "hydro", "43": "dendro", "44": "anemo", "45": "geo", "46": "cryo"}

// affixCodes are the substat codes inside an artifact roll ID (rarity, 0,
// code, tier), by Enka's substat names.
var affixCodes = map[string]int{"FIGHT_PROP_HP": 102, "FIGHT_PROP_HP_PERCENT": 103, "FIGHT_PROP_ATTACK": 105, "FIGHT_PROP_ATTACK_PERCENT": 106, "FIGHT_PROP_DEFENSE": 108, "FIGHT_PROP_DEFENSE_PERCENT": 109, "FIGHT_PROP_CRITICAL": 120, "FIGHT_PROP_CRITICAL_HURT": 122, "FIGHT_PROP_CHARGE_EFFICIENCY": 123, "FIGHT_PROP_ELEMENT_MASTERY": 124}

// slots are the artifact positions by Enka's equipment type, as miao's
// EnkaData reads them.
var slots = map[string]int{"EQUIP_BRACER": 1, "EQUIP_NECKLACE": 2, "EQUIP_SHOES": 3, "EQUIP_RING": 4, "EQUIP_DRESS": 5}

// panelStats are the character properties the official panel lists, with
// the base part the panel shows beside hit points, attack and defense, and
// the character's own crit and recharge.
var panelStats = []struct {
	id, base, name string
	percent        bool
	fixedBase      float64
}{
	{"2000", "1", "生命值上限", false, 0}, {"2001", "4", "攻击力", false, 0}, {"2002", "7", "防御力", false, 0}, {"28", "", "元素精通", false, 0},
	{"20", "", "暴击率", true, 0.05}, {"22", "", "暴击伤害", true, 0.5}, {"23", "", "元素充能效率", true, 1}, {"26", "", "治疗加成", true, 0},
	{"30", "", "物理伤害加成", true, 0}, {"40", "", "火元素伤害加成", true, 0}, {"41", "", "雷元素伤害加成", true, 0}, {"42", "", "水元素伤害加成", true, 0},
	{"43", "", "草元素伤害加成", true, 0}, {"44", "", "风元素伤害加成", true, 0}, {"45", "", "岩元素伤害加成", true, 0}, {"46", "", "冰元素伤害加成", true, 0},
}

func percent(value float64) string { return strconv.FormatFloat(value, 'f', 1, 64) + "%" }
func flat(value float64) string    { return strconv.FormatFloat(math.Round(value), 'f', 0, 64) }

func equipmentStat(stat enkaStat) (app.PanelStat, bool) {
	name := stat.MainPropID + stat.AppendPropID
	prop, ok := fightProps[name]
	if !ok {
		return app.PanelStat{}, false
	}
	value := flat(stat.StatValue)
	if prop.percent {
		value = percent(stat.StatValue)
	}
	return app.PanelStat{ID: prop.id, Key: statKeys[prop.id], Name: prop.name, Value: value}, true
}

// Parse reads Enka's answer the way miao's EnkaApi and EnkaData do: every
// character with details and the weapon, artifacts, talents and
// constellations it shows, with the final properties Enka reports.
func Parse(_ context.Context, game app.Game, catalog app.Catalog, raw []byte) (app.ShowcaseProfile, error) {
	var answer enkaAnswer
	if err := json.Unmarshal(raw, &answer); err != nil || answer.PlayerInfo == nil {
		return app.ShowcaseProfile{}, &app.ShowcaseFailure{Status: 200}
	}
	profile := app.ShowcaseProfile{Nickname: answer.PlayerInfo.Nickname, Level: answer.PlayerInfo.Level, TTL: time.Duration(answer.TTL) * time.Second}
	if len(answer.AvatarInfoList) == 0 || answer.AvatarInfoList[0].PropMap == nil {
		return profile, app.ErrShowcaseEmpty
	}
	sets := map[string]string{}
	for id, set := range catalog.ArtifactSets {
		if len(id) == 5 && id[0] >= '0' && id[0] <= '9' {
			// miao finds a set by the first two digits of a piece ID.
			sets[id[:2]] = set
		}
	}
	for _, avatar := range answer.AvatarInfoList {
		if panel, ok := parseAvatar(game, catalog, sets, avatar); ok {
			profile.Panels = append(profile.Panels, panel)
		}
	}
	return profile, nil
}

func parseAvatar(game app.Game, catalog app.Catalog, sets map[string]string, avatar enkaAvatar) (app.CharacterPanel, bool) {
	id := strconv.Itoa(avatar.AvatarID)
	entry, ok := catalog.Get(id)
	record, found := calcRecord(game, id, avatar.SkillLevelMap)
	if !ok || !found {
		return app.CharacterPanel{}, false
	}
	level, _ := strconv.Atoi(text(avatar.PropMap["4001"].Val))
	promote, _ := strconv.Atoi(text(avatar.PropMap["1002"].Val))
	panel := app.CharacterPanel{ID: id, Name: entry.Name, Level: level, Promote: &promote, Rank: len(avatar.TalentIDList), RankKnown: true, Element: record.Element, Source: "enka",
		WeaponKnown: true, EquipmentKnown: true, Stats: []app.PanelStat{}, Equipment: []app.PanelEquipment{}, Skills: []app.PanelSkill{}, Ranks: []app.PanelSkill{}}
	for _, stat := range panelStats {
		final, ok := avatar.FightPropMap[stat.id]
		if !ok {
			continue
		}
		item := app.PanelStat{ID: stat.id, Key: statKeys[stat.id], Name: stat.name}
		if stat.percent {
			item.Value, item.Base, item.Added = percent(final*100), percent(stat.fixedBase*100), percent((final-stat.fixedBase)*100)
		} else {
			base := avatar.FightPropMap[stat.base]
			item.Value, item.Base, item.Added = flat(final), flat(base), flat(final)
			if stat.base != "" {
				item.Added = flat(final - base)
			} else {
				item.Base = "0"
			}
		}
		panel.Stats = append(panel.Stats, item)
	}
	for _, equip := range avatar.EquipList {
		switch {
		case equip.Weapon != nil:
			weaponID := strconv.Itoa(equip.ItemID)
			w := app.PanelEquipment{ID: weaponID, Level: equip.Weapon.Level, Promote: &equip.Weapon.PromoteLevel, Rarity: strconv.Itoa(equip.Flat.RankLevel), Refinement: 1, Main: []app.PanelStat{}, Sub: []app.PanelStat{}, Complete: true}
			if weapon, ok := catalog.Get(weaponID); ok {
				w.Name = weapon.Name
			}
			for _, refine := range equip.Weapon.AffixMap {
				w.Refinement = refine + 1
			}
			for index, raw := range equip.Flat.WeaponStats {
				if stat, ok := equipmentStat(raw); ok {
					if index == 0 {
						w.Main = append(w.Main, stat)
					} else {
						w.Sub = append(w.Sub, stat)
					}
				}
			}
			panel.Weapon = &w
		case equip.Reliquary != nil:
			slot := slots[equip.Flat.EquipType]
			id := strconv.Itoa(equip.ItemID)
			set := catalog.ArtifactSets[id]
			if set == "" && len(id) == 5 {
				set = sets[id[:2]]
			}
			if slot == 0 || set == "" || equip.Flat.ReliquaryMainstat == nil {
				continue
			}
			piece := app.PanelEquipment{ID: id, Slot: slot, Level: min(20, equip.Reliquary.Level-1), Rarity: strconv.Itoa(equip.Flat.RankLevel), SetName: set, Main: []app.PanelStat{}, Sub: []app.PanelStat{}, Complete: true}
			if names := catalog.ArtifactPieces[set]; slot <= len(names) {
				piece.Name = names[slot-1]
			}
			if main, ok := equipmentStat(*equip.Flat.ReliquaryMainstat); ok {
				piece.Main = append(piece.Main, main)
			}
			// Each roll ID names its substat; the official panel counts the
			// rolls after the first as upgrades.
			rolls := map[int]int{}
			for _, roll := range equip.Reliquary.AppendPropIDList {
				rolls[roll/10%1000]++
			}
			for _, raw := range equip.Flat.ReliquarySubstats {
				if stat, ok := equipmentStat(raw); ok {
					stat.Times = max(0, rolls[affixCodes[raw.AppendPropID]]-1)
					piece.Sub = append(piece.Sub, stat)
				}
			}
			piece.Complete = len(piece.Main) == 1
			panel.Equipment = append(panel.Equipment, piece)
		}
	}
	slices.SortFunc(panel.Equipment, func(a, b app.PanelEquipment) int { return a.Slot - b.Slot })
	panel.Skills, panel.Ranks = talents(record, avatar, panel.Rank)
	return panel, true
}

// calcRecord is the character's upstream record; the Traveler's element is
// the one whose talents the showcase lists.
func calcRecord(game app.Game, id string, skills map[string]int) (reference.Character, bool) {
	if game.Calc == nil {
		return reference.Character{}, false
	}
	traveler := id == "10000005" || id == "10000007"
	for _, record := range game.Calc.Metadata().Characters {
		if record.ID != id && !(traveler && record.ID == "10000007") {
			continue
		}
		if !traveler {
			return record, true
		}
		for skill := range asMap(record.Data["talentId"]) {
			if _, ok := skills[skill]; ok {
				return record, true
			}
		}
	}
	return reference.Character{}, false
}

// talents are the character's normal attack, skill and burst with the three
// levels a constellation adds, as miao's talentCons gives them, and its six
// constellations.
func talents(record reference.Character, avatar enkaAvatar, rank int) ([]app.PanelSkill, []app.PanelSkill) {
	ids := asMap(record.Data["talentId"])
	cons := asMap(record.Data["talentCons"])
	names := asMap(record.Data["talent"])
	skills := []app.PanelSkill{}
	keys := []string{}
	for key := range avatar.SkillLevelMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		talent := text(ids[key])
		if talent == "" {
			continue
		}
		skill := app.PanelSkill{SkillType: 1, ID: key, Level: avatar.SkillLevelMap[key], Active: true, Name: text(asMap(names[talent])["name"])}
		if need, _ := strconv.Atoi(text(cons[talent])); need > 0 && rank >= need {
			skill.ExtraLevel = 3
			skill.Level += 3
		}
		skills = append(skills, skill)
	}
	slices.SortFunc(skills, func(a, b app.PanelSkill) int {
		order := func(s app.PanelSkill) int { return strings.Index("aeq", text(ids[s.ID])) }
		return order(a) - order(b)
	})
	ranks := []app.PanelSkill{}
	constellations := asMap(record.Data["cons"])
	for index := 1; index <= 6; index++ {
		item := asMap(constellations[strconv.Itoa(index)])
		ranks = append(ranks, app.PanelSkill{Name: text(item["name"]), Active: index <= rank, Description: text(item["desc"])})
	}
	return skills, ranks
}

func asMap(value any) map[string]any { result, _ := value.(map[string]any); return result }

func text(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		return value.String()
	}
	return ""
}
