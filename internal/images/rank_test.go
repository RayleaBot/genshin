package images

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func TestRankSetNameFollowsMiao(t *testing.T) {
	catalog := gamekit.Catalog{SetAbbrs: map[string]string{"炽烈的炎之魔女": "魔女", "角斗士的终幕礼": "角斗"}}
	panel := func(sets ...string) gamekit.CharacterPanel {
		equipment := []gamekit.PanelEquipment{}
		for _, set := range sets {
			equipment = append(equipment, gamekit.PanelEquipment{SetName: set})
		}
		return gamekit.CharacterPanel{Equipment: equipment}
	}
	for want, sets := range map[string][]string{
		// One set whose name and count fit in seven characters keeps its name.
		"深林的记忆4": {"深林的记忆", "深林的记忆", "深林的记忆", "深林的记忆", "角斗士的终幕礼"},
		"魔女4":    {"炽烈的炎之魔女", "炽烈的炎之魔女", "炽烈的炎之魔女", "炽烈的炎之魔女"},
		// Two sets use abbreviations; a set without one keeps its name.
		"魔女2+深林的记忆2": {"炽烈的炎之魔女", "炽烈的炎之魔女", "深林的记忆", "深林的记忆", "角斗士的终幕礼"},
		"":           {"炽烈的炎之魔女", "深林的记忆"},
	} {
		if got := rankSetName(catalog, panel(sets...)); got != want {
			t.Errorf("rankSetName(%v) = %q, want %q", sets, got, want)
		}
	}
	if got := rankDamageTitle("Q·万雷· 凝渊伤害测试标题"); got != "Q万雷凝渊伤害测试标题" {
		t.Errorf("title = %q", got)
	}
	if got := rankDamageTitle("开Q后的长长的重击蒸发伤害"); got != "开Q后的长长的重击蒸发" {
		t.Errorf("title = %q", got)
	}
}
