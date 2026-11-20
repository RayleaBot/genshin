package app

import "testing"

func TestProfileNumbersUseWeaponGrowthAndPreserveIncompletePanel(t *testing.T) {
	g := testGame(t)
	weaponPromote := 5
	panel := CharacterPanel{ID: "10000003", Element: "anemo", Level: 90, RankKnown: true, Source: "mihoyo",
		Weapon: &PanelEquipment{ID: "11401", Level: 70, Promote: &weaponPromote, Refinement: 2},
		Stats:  []PanelStat{{ID: "2001", Value: "1233", Base: "612"}, {ID: "20", Value: "21.3%", Base: "21.3%"}},
	}
	record, err := findReferenceCharacter(g.Calc, panel)
	if err != nil {
		t.Fatal(err)
	}
	n := g.profileNumbers(t.Context(), panel, record)
	for key, want := range map[string]PanelStat{
		"atk":  {Value: "1,233.0", Base: "611.8", Added: "621.2"},
		"cpct": {Value: "21.3%", Base: "5.0%", Added: "16.3%"},
	} {
		if got := n.Stats[key]; got != want {
			t.Errorf("%s = %+v, want %+v", key, got, want)
		}
	}
	if _, ok := n.Stats["mastery"]; ok {
		t.Error("missing equipment produced a mastery total")
	}
	if len(n.Weapon) != 2 || n.Weapon[0].Value != "372.6" || n.Weapon[1].Key != "recharge" || n.Weapon[1].Value != "50.5%" {
		t.Errorf("weapon = %+v", n.Weapon)
	}
}
