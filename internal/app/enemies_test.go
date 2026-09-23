package app

import (
	"encoding/json"
	"testing"
)

func TestEnemyValueReadsTheMessageAsAtlas(t *testing.T) {
	var table EnemyTable
	if err := json.Unmarshal(pluginFile(t, "internal/assets/data/enemies.json"), &table); err != nil {
		t.Fatal(err)
	}
	for msg, want := range map[string]string{
		"#原魔无相之火生命值":     "你查询的90级无相之火生命值为24354.1*7.00*1.00=170478.6\n修饰因子:\n·普通(1倍生命值)",
		"#原魔100级无相之火攻击力": "你查询的100级无相之火攻击力为2462.2*6.00*1.00=14773.3\n修饰因子:\n·普通(1倍攻击力)",
		// A factor sets the level when none is written, and its multiplier
		// for the enemy.
		"#原魔12-1丘丘人暴徒生命值": "你查询的95级丘丘人生命值为27748.8*1.00*2.50=69372.1\n修饰因子:\n·12-1(2.5倍生命值)",
		"#原魔4人无相之水生命值":    "你查询的90级无相之水生命值为24354.1*9.00*2.50=547967.0\n修饰因子:\n·4人(2.5倍生命值)",
		// Only the stat's factor groups count, one factor from each.
		"#原魔蒙德天赋本丘丘人攻击力 4人": "你查询的90级丘丘人攻击力为1903.1*1.80*1.40=4795.9\n修饰因子:\n·4人(1.4倍攻击力)",
		"#原魔火元素增幅黑泥无相之火攻击力": "你查询的90级无相之火攻击力为1903.1*6.00*1.40=15986.4\n修饰因子:\n·火元素增幅(1.4倍攻击力)",
		// The longest name the message holds is the enemy.
		"#原魔至纯的无相之雷生命值": "你查询的90级至纯的无相之雷生命值为24354.1*11.00*1.00=267895.0\n修饰因子:\n·普通(1倍生命值)",
		// 级 without a number before it is no level.
		"#原魔无相之火生命值级": "你查询的90级无相之火生命值为24354.1*7.00*1.00=170478.6\n修饰因子:\n·普通(1倍生命值)",
		// A curve Atlas's data lacks.
		"#原魔帽子水母生命值": "固定资料没有此等级的属性曲线。",
		// An OtherName name that misses its enemy by a slip finds it.
		"#原魔遗迹侦察者生命值": "你查询的90级遗迹侦查者生命值为37100.3*4.20*1.00=155821.2\n修饰因子:\n·普通(1倍生命值)",
		// An OtherName name no enemy has, a message without a stat, and one
		// that does not start with 原魔 get no reply.
		"#原魔沙虫炮充能装置生命值": "",
		"#原魔无相之火":       "",
		"#原魔":           "",
		"#90级无相之火生命值":   "",
	} {
		if got, ok := table.value(msg); got != want || ok != (want != "") {
			t.Errorf("%q: %q %v, want %q", msg, got, ok, want)
		}
	}
}
