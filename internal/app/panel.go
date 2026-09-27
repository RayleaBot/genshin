package app

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

type PanelStat struct {
	ID    string `json:"id"`
	Key   string `json:"key,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Base  string `json:"base,omitempty"`
	Added string `json:"added,omitempty"`
	// Times is how often an equipment substat was upgraded.
	Times int `json:"times,omitempty"`
}
type PanelEquipment struct {
	Promote    *int            `json:"promote,omitempty"`
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Slot       int             `json:"slot"`
	Level      int             `json:"level"`
	Rarity     string          `json:"rarity"`
	Refinement int             `json:"refinement,omitempty"`
	SetName    string          `json:"set_name,omitempty"`
	Main       []PanelStat     `json:"main"`
	Sub        []PanelStat     `json:"sub"`
	Complete   bool            `json:"complete"`
	Score      *EquipmentScore `json:"score,omitempty"`
	// MainID and AttrIDs are an artifact's main stat and roll IDs as the
	// showcase gives them (its reliquary mainPropId and appendPropIdList),
	// which miao's EnkaData saves as they are. Other sources have none.
	MainID  int   `json:"main_id,omitempty"`
	AttrIDs []int `json:"attr_ids,omitempty"`
}
type PanelSkill struct {
	SkillType   int    `json:"skill_type,omitempty"`
	ID          string `json:"id,omitempty"`
	ExtraLevel  int    `json:"extra_level,omitempty"`
	Name        string `json:"name"`
	Level       int    `json:"level,omitempty"`
	Active      bool   `json:"active"`
	Description string `json:"description,omitempty"`
}
type CharacterPanel struct {
	EquipmentKnown  bool             `json:"equipment_known"`
	WeaponKnown     bool             `json:"weapon_known"`
	Promote         *int             `json:"promote,omitempty"`
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Level           int              `json:"level"`
	Rank            int              `json:"rank"`
	RankKnown       bool             `json:"rank_known"`
	Element         string           `json:"element"`
	Source          string           `json:"source"`
	Stats           []PanelStat      `json:"stats"`
	Weapon          *PanelEquipment  `json:"weapon,omitempty"`
	Equipment       []PanelEquipment `json:"equipment"`
	Skills          []PanelSkill     `json:"skills"`
	Ranks           []PanelSkill     `json:"ranks"`
	ScoreRule       string           `json:"score_rule,omitempty"`
	ScoreNote       string           `json:"score_note,omitempty"`
	TotalScore      *float64         `json:"total_score,omitempty"`
	ScoredEquipment int              `json:"scored_equipment,omitempty"`
	// ScoreDetail is the upstream scoring detail for the panel image; it is
	// not stored with the panel.
	ScoreDetail *ScoreDetail `json:"-"`
	// Official is the official character entry the panel was read from, for
	// images that show fields the panel does not keep. It is not stored.
	Official map[string]any `json:"-"`
	// UpdatedAtMS is when the kept panel was saved; it is not stored with the
	// panel.
	UpdatedAtMS int64 `json:"-"`
}

var markup = regexp.MustCompile(`<[^>]*>`)

func plainGameText(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "<br>", "\n"), "<br/>", "\n")
	return html.UnescapeString(markup.ReplaceAllString(value, ""))
}

var gsStatKeys = map[string]string{"2": "hpPlus", "3": "hp", "5": "atkPlus", "6": "atk", "8": "defPlus", "9": "def", "20": "cpct", "22": "cdmg", "23": "recharge", "26": "heal", "28": "mastery", "30": "phy", "40": "pyro", "41": "electro", "42": "hydro", "43": "dendro", "44": "anemo", "45": "geo", "46": "cryo"}

func oneStat(value any) []any {
	if item := asObject(value); item != nil {
		return []any{item}
	}
	return nil
}

// NormalizePanels reads the characters of the official character detail
// into panels. Property names come from its property_map.
func NormalizePanels(result QueryResult, catalog Catalog) []CharacterPanel {
	metadata := asObject(result.Data["property_map"])
	panels := []CharacterPanel{}
	for _, raw := range asList(result.Data["list"]) {
		item := asObject(raw)
		base := asObject(item["base"])
		id := asText(base["id"])
		if id == "" {
			continue
		}
		entry, hasEntry := catalog.Get(id)
		name := asText(base["name"])
		if name == "" && hasEntry {
			name = entry.Name
		}
		if name == "" {
			name = "角色 " + id
		}
		panel := CharacterPanel{ID: id, Name: plainGameText(name), Level: number(base["level"]), Rank: number(base["actived_constellation_num"]), Element: asText(base["element"]), Source: "mihoyo", Stats: []PanelStat{}, Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}, Official: item}
		if panel.Element == "" && hasEntry {
			panel.Element = entry.Element
		}
		if value, exists := base["promote_level"]; exists {
			n := number(value)
			panel.Promote = &n
		}
		_, panel.RankKnown = base["actived_constellation_num"]
		_, panel.WeaponKnown = item["weapon"]
		_, panel.EquipmentKnown = item["relics"]
		seen := map[string]bool{}
		for _, field := range []string{"base_properties", "extra_properties", "element_properties"} {
			for _, stat := range panelStats(asList(item[field]), metadata, "final") {
				if !seen[stat.ID] {
					panel.Stats = append(panel.Stats, stat)
					seen[stat.ID] = true
				}
			}
		}
		if weapon := asObject(item["weapon"]); weapon != nil && firstText(weapon, "name", "id") != "" {
			w := PanelEquipment{ID: asText(weapon["id"]), Name: plainGameText(asText(weapon["name"])), Level: number(weapon["level"]), Rarity: asText(weapon["rarity"]), Refinement: number(weapon["affix_level"]), Main: panelStats(oneStat(weapon["main_property"]), metadata, "final"), Sub: panelStats(oneStat(weapon["sub_property"]), metadata, "final")}
			if value, exists := weapon["promote_level"]; exists {
				n := number(value)
				w.Promote = &n
			}
			panel.Weapon = &w
		}
		for _, raw := range asList(item["relics"]) {
			gear := asObject(raw)
			equipment := PanelEquipment{ID: asText(gear["id"]), Name: plainGameText(asText(gear["name"])), Slot: number(gear["pos"]), Level: number(gear["level"]), Rarity: asText(gear["rarity"]), SetName: asText(asObject(gear["set"])["name"])}
			if equipment.SetName == "" {
				equipment.SetName = catalog.ArtifactSets[equipment.ID]
				if equipment.SetName == "" {
					equipment.SetName = catalog.ArtifactSets[equipment.Name]
				}
			}
			equipment.Main = panelStats(oneStat(gear["main_property"]), metadata, "value")
			equipment.Sub = panelStats(asList(gear["sub_property_list"]), metadata, "value")
			_, subPresent := gear["sub_property_list"]
			equipment.Complete = subPresent && len(equipment.Main) > 0
			panel.Equipment = append(panel.Equipment, equipment)
		}
		for i, raw := range asList(item["skills"]) {
			skill := asObject(raw)
			name := asText(skill["name"])
			if name == "" {
				name = "技能 " + strconv.Itoa(i+1)
			}
			active := true
			if flag, ok := skill["is_unlock"].(bool); ok {
				active = flag
			}
			panel.Skills = append(panel.Skills, PanelSkill{SkillType: number(skill["skill_type"]), ID: asText(skill["skill_id"]), ExtraLevel: number(skill["extra_level"]), Name: plainGameText(name), Level: number(skill["level"]), Active: active, Description: plainGameText(asText(skill["desc"]))})
		}
		for _, raw := range asList(item["constellations"]) {
			rank := asObject(raw)
			active, _ := rank["is_actived"].(bool)
			panel.Ranks = append(panel.Ranks, PanelSkill{Name: plainGameText(asText(rank["name"])), Active: active, Description: plainGameText(asText(rank["effect"]))})
		}
		panels = append(panels, panel)
	}
	return panels
}

// panelStats reads official properties, named from the detail's
// property_map and shown by their value key ("final" for the character and
// weapon, "value" for artifacts). A property without that value is left out
// rather than shown as zero.
func panelStats(raw []any, metadata map[string]any, key string) []PanelStat {
	result := []PanelStat{}
	for _, value := range raw {
		item := asObject(value)
		id := asText(item["property_type"])
		display := asText(item[key])
		if id == "" || display == "" {
			continue
		}
		name := firstText(asObject(metadata[id]), "name", "filter_name")
		if name == "" {
			name = "属性 " + id
		}
		result = append(result, PanelStat{ID: id, Key: gsStatKeys[id], Name: plainGameText(name), Value: display, Base: asText(item["base"]), Added: asText(item["add"]), Times: number(item["times"])})
	}
	return result
}

func PanelView(game Game, panels []CharacterPanel, uid string) View {
	v := View{Title: game.Name + "角色面板", Subtitle: uid, Rows: []Row{}, Sections: []Section{}, Note: "属性与装备来自米游社；未返回的数值不作推算。"}
	if len(panels) == 0 {
		v.Note = "此账号未返回所选角色的面板。"
		return v
	}
	if len(panels) > 1 {
		for _, p := range panels {
			v.Rows = append(v.Rows, Row{Label: p.Name, Value: p.ID + " · 等级 " + strconv.Itoa(p.Level) + " · " + strconv.Itoa(p.Rank) + " 阶"})
		}
		v.Note = "请指定角色名称或 ID 查看完整面板。"
		return v
	}
	p := panels[0]
	v.Title = p.Name + " · 角色面板"
	v.Rows = append(v.Rows, Row{Label: "等级", Value: strconv.Itoa(p.Level)}, Row{Label: "命之座", Value: strconv.Itoa(p.Rank)})
	rows := []Row{}
	for _, stat := range p.Stats {
		rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
	}
	v.Sections = append(v.Sections, Section{Title: "角色属性", Rows: rows})
	if p.Weapon != nil {
		w := p.Weapon
		rows = []Row{{Label: w.Name, Value: "等级 " + strconv.Itoa(w.Level) + " · 精炼/叠影 " + strconv.Itoa(w.Refinement)}}
		for _, stat := range append(append([]PanelStat{}, w.Main...), w.Sub...) {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		v.Sections = append(v.Sections, Section{Title: "武器", Rows: rows})
	}
	for _, equipment := range p.Equipment {
		rows = []Row{{Label: "等级", Value: strconv.Itoa(equipment.Level)}}
		for _, stat := range equipment.Main {
			rows = append(rows, Row{Label: "主属性 · " + stat.Name, Value: stat.Value})
		}
		for _, stat := range equipment.Sub {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		if equipment.Score != nil {
			rows = append(rows, Row{Label: "评分", Value: strconv.FormatFloat(equipment.Score.Value, 'f', 1, 64) + " · " + equipment.Score.Grade})
		}
		v.Sections = append(v.Sections, Section{Title: strconv.Itoa(equipment.Slot) + "号位 · " + equipment.Name, Rows: rows})
	}
	rows = []Row{}
	for _, skill := range p.Skills {
		status := "未解锁"
		if skill.Active {
			status = "等级 " + strconv.Itoa(skill.Level)
		}
		rows = append(rows, Row{Label: skill.Name, Value: status})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "天赋", Rows: rows})
	}
	rows = []Row{}
	for _, rank := range p.Ranks {
		status := "未解锁"
		if rank.Active {
			status = "已解锁"
		}
		rows = append(rows, Row{Label: rank.Name, Value: status})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "命座", Rows: rows})
	}
	if p.TotalScore != nil {
		v.Rows = append(v.Rows, Row{Label: "装备评分合计", Value: strconv.FormatFloat(*p.TotalScore, 'f', 1, 64) + " · 已评分 " + strconv.Itoa(p.ScoredEquipment) + " / " + strconv.Itoa(len(p.Equipment)) + " 件"})
	}
	if p.ScoreRule != "" {
		v.Note += "\n评分规则：" + p.ScoreRule + "。"
	}
	if p.ScoreNote != "" {
		v.Note += "\n" + p.ScoreNote
	}
	return v
}
