package app

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// EnemyInfo is an enemy of Atlas's Enemy.yaml; Aliases, which the
// management page lists, are its OtherName.yaml entry's.
type EnemyInfo struct {
	Name     string   `json:"name"`
	Group    string   `json:"group"`
	Aliases  []string `json:"aliases"`
	HPCurve  string   `json:"hp_curve"`
	HPBase   *float64 `json:"hp_base"`
	ATKCurve string   `json:"atk_curve"`
	ATKBase  *float64 `json:"atk_base"`
}
type EnemyModifier struct {
	ID    string   `json:"id"`
	Group string   `json:"group"`
	Names []string `json:"names"`
	Level int      `json:"level"`
	Value any      `json:"value"`
}

// EnemyTable is the Genshin monster data from the pinned Atlas snapshot.
// Names keep OtherName.yaml's order, including the few names no enemy has.
type EnemyTable struct {
	Version   string               `json:"version"`
	Enemies   []EnemyInfo          `json:"enemies"`
	Names     []Aliased            `json:"names"`
	Curves    []map[string]float64 `json:"curves"`
	Modifiers []EnemyModifier      `json:"modifiers"`
}

// ratio is a factor's multiplier for an enemy, as getRatioValue reads it:
// one number, or the first of a list that names the enemy or names none.
func (m EnemyModifier) ratio(enemy string) (float64, bool) {
	if value, ok := m.Value.(float64); ok {
		return value, true
	}
	for _, raw := range asList(m.Value) {
		v := asObject(raw)
		names := []string{}
		for _, n := range asList(v["enemy"]) {
			names = append(names, asText(n))
		}
		if len(names) == 0 || slices.Contains(names, enemy) {
			return challengeNumber(v["value"])
		}
	}
	return 0, false
}

// curve is a stat curve's value at a level, false when the data has none.
func (t *EnemyTable) curve(level int, name string) (float64, bool) {
	for _, c := range t.Curves {
		if int(c["Level"]) == level {
			value, ok := c[name]
			return value, ok
		}
	}
	return 0, false
}

func (a *App) enemyAction(action string, input map[string]any) (map[string]any, error) {
	enemyData := a.Game.Data.Enemies
	if enemyData == nil {
		return nil, gameError("operation_denied", "此游戏没有固定原魔资料。")
	}
	if action == "enemies.schema" {
		return map[string]any{"enemies": enemyData.Enemies, "modifiers": enemyData.Modifiers, "version": enemyData.Version, "max_level": len(enemyData.Curves)}, nil
	}
	var q struct {
		Name      string   `json:"name"`
		Level     int      `json:"level"`
		Stat      string   `json:"stat"`
		Modifiers []string `json:"modifiers"`
	}
	if decodeObject(input, &q) != nil || len(q.Modifiers) > 5 || len(q.Name) > 128 || (q.Stat != "HP" && q.Stat != "ATK") || q.Level < 0 || q.Level > len(enemyData.Curves) {
		return nil, gameError("input_invalid", "请选择原魔、属性、有效等级和因子。")
	}
	matches := []EnemyInfo{}
	for _, e := range enemyData.Enemies {
		if e.Name == q.Name || slices.Contains(e.Aliases, q.Name) {
			matches = append(matches, e)
		}
	}
	if len(matches) != 1 {
		return nil, gameError("enemy_ambiguous", "原魔名称未唯一匹配，请从列表选择完整名称。")
	}
	enemy := matches[0]
	curve, base := enemy.HPCurve, enemy.HPBase
	if q.Stat == "ATK" {
		curve, base = enemy.ATKCurve, enemy.ATKBase
	}
	if base == nil || !finiteRange(*base, 0, 1e15) || curve == "" {
		return nil, gameError("enemy_missing", "固定资料缺少此原魔属性。")
	}
	level := q.Level
	groups := map[string]bool{}
	ratio := 1.0
	rows := []Row{}
	for _, id := range q.Modifiers {
		i := slices.IndexFunc(enemyData.Modifiers, func(m EnemyModifier) bool { return m.ID == id })
		if i < 0 {
			return nil, gameError("input_invalid", "原魔因子不存在。")
		}
		m := enemyData.Modifiers[i]
		if !strings.HasSuffix(m.Group, q.Stat) || groups[m.Group] {
			return nil, gameError("input_invalid", "每个属性因子类别只能选择一项。")
		}
		groups[m.Group] = true
		if q.Level == 0 && m.Level > 0 {
			level = m.Level
		}
		value, ok := m.ratio(enemy.Name)
		if !ok || !finiteRange(value, 0, 1000) {
			return nil, gameError("enemy_missing", "因子没有此原魔的有效倍率。")
		}
		ratio *= value
		rows = append(rows, Row{strings.Join(m.Names, " / "), fmt.Sprintf("%.4g 倍", value)})
	}
	if level == 0 {
		level = 90
	}
	scale, known := enemyData.curve(level, curve)
	if !known || !finiteRange(scale, 0, 1e15) {
		return nil, gameError("enemy_missing", "固定资料没有此等级的属性曲线。")
	}
	value := scale * *base * ratio
	if !finiteRange(value, 0, 1e30) {
		return nil, gameError("enemy_missing", "计算结果超出有效范围。")
	}
	view := View{Title: fmt.Sprintf("%d级 %s · %s", level, enemy.Name, q.Stat), Rows: []Row{{"计算结果", fmt.Sprintf("%.1f", value)}, {"计算过程", fmt.Sprintf("曲线 %.6g × 基础 %.6g × 因子 %.6g", scale, *base, ratio)}}, Sections: []Section{{Title: "所选修饰因子", Rows: rows}}, Note: "来源：" + enemyData.Version + "。固定数据，未校准快照之外的新版本。"}
	return map[string]any{"value": value, "level": level, "base": *base, "curve": scale, "ratio": ratio, "view": view}, nil
}

var (
	// enemyWords is EnemyValue's rule, enemyStat and enemyLevel how it reads
	// the stat and the level.
	enemyWords = regexp.MustCompile(`^[#/](原魔).*(生命值|攻击力).*`)
	enemyStat  = regexp.MustCompile(`(生命值|攻击力)`)
	enemyLevel = regexp.MustCompile(`([01]?\d?\d|200)级`)
)

// enemyValue is Atlas's EnemyValue.query for a message as Yunzai reads it:
// the stat written first; the enemy named by the longest OtherName name
// whose name or an alias the message holds; the level written as N级, else
// the last named factor's level, else 90; from each factor group of the
// stat the first factor the message names, 普通 when none. It returns the
// reply, false for a message EnemyValue leaves.
func (t *EnemyTable) value(msg string) (string, bool) {
	if t == nil || !enemyWords.MatchString(msg) {
		return "", false
	}
	stat, label := "HP", "生命值"
	if enemyStat.FindString(msg) == "攻击力" {
		stat, label = "ATK", "攻击力"
	}
	named := []string{}
	for _, name := range t.Names {
		if strings.Contains(msg, name.Name) || slices.ContainsFunc(name.Aliases, func(alias string) bool { return strings.Contains(msg, alias) }) {
			named = append(named, name.Name)
		}
	}
	if len(named) == 0 {
		return "", false
	}
	slices.SortStableFunc(named, func(a, b string) int { return jsLength(b) - jsLength(a) })
	index := slices.IndexFunc(t.Enemies, func(enemy EnemyInfo) bool { return enemy.Name == named[0] })
	if index < 0 {
		return "", false
	}
	enemy := t.Enemies[index]
	// Upstream reads the level of any message with 级 and fails with no
	// reply when no number comes before it; such a message has no level
	// here.
	level := 0
	if match := enemyLevel.FindStringSubmatch(msg); match != nil {
		level, _ = strconv.Atoi(match[1])
	}
	factors := []EnemyModifier{}
	taken := map[string]bool{}
	for _, m := range t.Modifiers {
		if strings.Contains(m.Group, stat) && !taken[m.Group] && slices.ContainsFunc(m.Names, func(word string) bool { return strings.Contains(msg, word) }) {
			factors = append(factors, m)
			taken[m.Group] = true
		}
	}
	if len(factors) == 0 {
		factors = []EnemyModifier{{Names: []string{"普通"}, Value: 1.0}}
	}
	for _, m := range factors {
		if level == 0 && m.Level > 0 {
			level = m.Level
		}
	}
	if level == 0 {
		level = 90
	}
	curve, base := enemy.HPCurve, enemy.HPBase
	if stat == "ATK" {
		curve, base = enemy.ATKCurve, enemy.ATKBase
	}
	// Some enemies name a curve Atlas's Common.yaml lacks, which upstream
	// answers with NaN; the management page's replies are given instead.
	if base == nil {
		return "固定资料缺少此原魔属性。", true
	}
	scale, ok := t.curve(level, curve)
	if !ok {
		return "固定资料没有此等级的属性曲线。", true
	}
	ratio, lines := 1.0, "修饰因子:"
	for _, m := range factors {
		if value, ok := m.ratio(enemy.Name); ok {
			ratio *= value
			lines += "\n·" + m.Names[0] + "(" + strconv.FormatFloat(value, 'f', -1, 64) + "倍" + label + ")"
		}
	}
	return fmt.Sprintf("你查询的%d级%s%s为", level, enemy.Name, label) + JSFixed(scale, 1) + "*" + JSFixed(*base, 2) + "*" + JSFixed(ratio, 2) + "=" + JSFixed(scale*(*base)*ratio, 1) + "\n" + lines, true
}
