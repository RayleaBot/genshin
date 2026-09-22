package app

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
)

type EquipmentScore struct {
	Value float64 `json:"value"`
	Grade string  `json:"grade"`
}

// ScoreDetail is what miao's getMarkDetail gives its panel image: the total
// mark and grade, attribute weights, every piece's formatted main stat and
// substats, and the substats summed over all pieces.
type ScoreDetail struct {
	Title    string              `json:"title"`
	Mark     string              `json:"mark"`
	Grade    string              `json:"grade"`
	Weights  map[string]float64  `json:"weights"`
	Titles   map[string]string   `json:"titles"`
	AllAttrs []ScoredAttr        `json:"all_attrs"`
	Pieces   map[int]ScoredPiece `json:"-"`
	// Raw is the whole scoring result, for game-specific fields.
	Raw json.RawMessage `json:"-"`
}

// ScoredPiece is one piece of equipment as upstream's panel shows it.
type ScoredPiece struct {
	Mark  string       `json:"mark"`
	Grade string       `json:"grade"`
	Main  ScoredAttr   `json:"main"`
	Attrs []ScoredAttr `json:"attrs"`
}

// ScoredAttr is a formatted stat: its display value, its mark under the
// character's weights, rolls, and efficiency in maximum rolls ("-" when none).
type ScoredAttr struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Mark  string `json:"mark,omitempty"`
	UpNum int    `json:"upNum"`
	Eff   string `json:"eff"`
}

const scoreNote = "装备评分用于比较角色专属权重下的词条，不代表实际伤害或队伍收益。"

func decimalStat(value string) (float64, bool) {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	value = strings.ReplaceAll(value, ",", "")
	n, err := strconv.ParseFloat(value, 64)
	return n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0
}
func normalizedElement(value string) string {
	value = strings.ToLower(value)
	for key, names := range map[string][]string{"pyro": {"pyro", "火"}, "electro": {"electro", "雷"}, "hydro": {"hydro", "水"}, "dendro": {"dendro", "草"}, "anemo": {"anemo", "风"}, "geo": {"geo", "岩"}, "cryo": {"cryo", "冰"}, "physical": {"physical", "物理"}, "fire": {"fire"}, "ice": {"ice"}, "lightning": {"lightning"}, "wind": {"wind"}, "quantum": {"quantum", "量子"}, "imaginary": {"imaginary", "虚数"}} {
		if slices.Contains(names, value) {
			return key
		}
	}
	return value
}

// scorePanel rates each complete piece of equipment with the character's
// upstream rule, run by the calculation engine: miao's ArtisMark.
func (a *App) scorePanel(ctx context.Context, panel CharacterPanel) (CharacterPanel, error) {
	engine := a.Game.Calc
	if engine == nil {
		return panel, gameError("score_unavailable", "当前资料没有评分规则。")
	}
	record, err := findReferenceCharacter(engine, panel)
	if err != nil {
		return panel, gameError("score_unavailable", "缺少角色基准资料，暂不计算评分。")
	}
	var input map[string]any
	input, err = miaoScoreInput(panel)

	if err != nil {
		return panel, err
	}
	raw, err := engine.Score(ctx, record, input)
	if err != nil {
		return panel, gameError("score_unavailable", "评分计算失败，暂不输出评分。")
	}
	var result struct {
		ScoreDetail
		Pieces []struct {
			ScoredPiece
			Slot  int     `json:"slot"`
			Score float64 `json:"score"`
		} `json:"pieces"`
		Total float64 `json:"total"`
	}
	if json.Unmarshal(raw, &result) != nil || len(result.Pieces) == 0 {
		return panel, gameError("score_unavailable", "评分计算结果无效。")
	}
	scores := map[int]EquipmentScore{}
	detail := result.ScoreDetail
	detail.Raw = raw
	detail.Pieces = map[int]ScoredPiece{}
	for _, piece := range result.Pieces {
		scores[piece.Slot] = EquipmentScore{Value: piece.Score, Grade: piece.Grade}
		detail.Pieces[piece.Slot] = piece.ScoredPiece
	}
	panel.Equipment = append([]PanelEquipment{}, panel.Equipment...)
	scored := 0
	for i := range panel.Equipment {
		panel.Equipment[i].Score = nil
		if score, ok := scores[panel.Equipment[i].Slot]; ok {
			panel.Equipment[i].Score = &score
			delete(scores, panel.Equipment[i].Slot)
			scored++
		}
	}
	panel.TotalScore = &result.Total
	panel.ScoredEquipment = scored
	panel.ScoreRule = result.Title
	panel.ScoreDetail = &detail
	panel.ScoreNote = scoreNote
	if scored != len(panel.Equipment) {
		panel.ScoreNote += "部分装备缺少必要字段，未计算分数。"
	}
	return panel, nil
}

// miaoAttributeKeys maps official panel property IDs to miao attribute keys.
var miaoAttributeKeys = map[string]string{"2000": "hp", "2001": "atk", "2002": "def", "20": "cpct", "22": "cdmg", "23": "recharge", "28": "mastery", "26": "heal", "30": "phy"}

// miaoScoreInput passes the attributes, eidolon or constellation, weapon and
// complete equipment pieces that miao scoring rules read.
func miaoScoreInput(panel CharacterPanel) (map[string]any, error) {
	if !panel.RankKnown || panel.Rank < 0 || panel.Rank > 6 {
		return nil, gameError("score_unavailable", "面板缺少命座信息，暂不计算评分。")
	}
	attributes := map[string]float64{}
	for _, stat := range panel.Stats {
		if key := miaoAttributeKeys[stat.ID]; key != "" {
			if value, ok := buildStatNumber(stat.Value); ok {
				attributes[key] = value
			}
		}
	}
	required := []string{"hp", "atk", "def", "cpct", "cdmg", "recharge", "mastery"}
	for _, key := range required {
		if _, ok := attributes[key]; !ok {
			return nil, gameError("score_unavailable", "面板缺少评分所需的属性，暂不计算评分。")
		}
	}
	weapon := BuildWeapon{Refinement: 1}
	if w := panel.Weapon; w != nil {
		if w.Refinement < 1 || w.Refinement > 5 {
			return nil, gameError("score_unavailable", "缺少武器精炼信息，暂不计算评分。")
		}
		weapon = BuildWeapon{ID: w.ID, Name: w.Name, Refinement: w.Refinement}
	}
	maxSlot := 6
	maxSlot = 5

	gear := []BuildGear{}
	seen := map[int]bool{}
	for _, piece := range panel.Equipment {
		if !piece.Complete || len(piece.Main) != 1 || piece.Slot < 1 || piece.Slot > maxSlot || seen[piece.Slot] || piece.SetName == "" {
			continue
		}
		converted := BuildGear{Slot: piece.Slot, SetName: piece.SetName, Sub: []BuildStat{}}
		valid := true
		for i, stat := range append(append([]PanelStat{}, piece.Main...), piece.Sub...) {
			value, ok := decimalStat(stat.Value)
			if !ok || stat.Key == "" {
				valid = false
				break
			}
			if i == 0 {
				converted.Main = BuildStat{Key: stat.Key, Value: value}
			} else {
				converted.Sub = append(converted.Sub, BuildStat{Key: stat.Key, Value: value, Times: stat.Times})
			}
		}
		if valid {
			seen[piece.Slot] = true
			gear = append(gear, converted)
		}
	}
	if len(gear) == 0 {
		return nil, gameError("score_unavailable", "缺少完整词条，暂不输出评分。")
	}
	return map[string]any{"rank": panel.Rank, "attributes": attributes, "weapon": weapon, "equipment": gear}, nil
}
