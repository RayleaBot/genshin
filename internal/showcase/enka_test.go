package showcase_test

import (
	"errors"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/showcase"
)

// The fixture follows an Enka answer's shape with one artifact and the
// weapon; its values are made up.
const enkaFixture = `{"ttl":60,"playerInfo":{"nickname":"旅行者","level":60},"avatarInfoList":[{"avatarId":10000021,
"propMap":{"4001":{"type":4001,"ival":"90","val":"90"},"1002":{"type":1002,"ival":"6","val":"6"}},"talentIdList":[211,212,213],
"fightPropMap":{"1":9461.2,"4":897.4,"7":600.6,"2000":14241.2,"2001":2367.0,"2002":951.7,"20":0.519,"22":1.939,"23":1,"28":39.6,"30":1.083},
"skillLevelMap":{"10017":10,"10032":10,"10041":9},
"equipList":[{"itemId":82543,"reliquary":{"level":21,"mainPropId":14001,"appendPropIdList":[501082,501063,501221,501242,501222,501082,501223,501064]},
"flat":{"rankLevel":5,"itemType":"ITEM_RELIQUARY","equipType":"EQUIP_BRACER","reliquaryMainstat":{"mainPropId":"FIGHT_PROP_HP","statValue":4780},
"reliquarySubstats":[{"appendPropId":"FIGHT_PROP_DEFENSE","statValue":37},{"appendPropId":"FIGHT_PROP_ATTACK_PERCENT","statValue":11.1},{"appendPropId":"FIGHT_PROP_CRITICAL_HURT","statValue":18.7},{"appendPropId":"FIGHT_PROP_ELEMENT_MASTERY","statValue":19}]}},
{"itemId":15501,"weapon":{"level":90,"promoteLevel":6,"affixMap":{"115501":1}},"flat":{"rankLevel":5,"itemType":"ITEM_WEAPON","weaponStats":[{"appendPropId":"FIGHT_PROP_BASE_ATTACK","statValue":674},{"appendPropId":"FIGHT_PROP_CRITICAL","statValue":22.1}]}}]}]}`

func TestParseReadsEnkaLikeMiao(t *testing.T) {
	app, err := gamekit.New(assets.Kit(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := showcase.Parse(t.Context(), app.Game, app.Catalog, []byte(enkaFixture))
	if err != nil || profile.Nickname != "旅行者" || len(profile.Panels) != 1 {
		t.Fatal(profile, err)
	}
	panel := profile.Panels[0]
	if panel.Name != "安柏" || panel.Level != 90 || *panel.Promote != 6 || panel.Rank != 3 || panel.Element != "pyro" {
		t.Fatalf("identity = %+v", panel)
	}
	stats := map[string]gamekit.PanelStat{}
	for _, stat := range panel.Stats {
		stats[stat.ID] = stat
	}
	if hp := stats["2000"]; hp.Value != "14241" || hp.Base != "9461" || hp.Added != "4780" || stats["20"].Value != "51.9%" || stats["30"].Value != "108.3%" {
		t.Fatalf("stats = %+v", stats)
	}
	if w := panel.Weapon; w == nil || w.Name != "天空之翼" || w.Refinement != 2 || w.Main[0].Value != "674" || w.Sub[0].Key != "cpct" || w.Sub[0].Value != "22.1%" {
		t.Fatalf("weapon = %+v", panel.Weapon)
	}
	// The set comes from the piece ID's first two digits and the piece name
	// from the set and slot; the rolls after the first count as upgrades.
	piece := panel.Equipment[0]
	if piece.Slot != 1 || piece.SetName != "染血的骑士道" || piece.Name != "染血的铁之心" || piece.Level != 20 || piece.Main[0].Value != "4780" {
		t.Fatalf("artifact = %+v", piece)
	}
	times := map[string]int{}
	for _, stat := range piece.Sub {
		times[stat.Key] = stat.Times
	}
	if times["cdmg"] != 2 || times["defPlus"] != 1 || times["mastery"] != 0 {
		t.Fatalf("upgrades = %v", times)
	}
	// Constellation three adds three levels to Amber's burst.
	levels := []int{}
	for _, skill := range panel.Skills {
		levels = append(levels, skill.Level)
	}
	if len(levels) != 3 || levels[0] != 9 || levels[1] != 10 || levels[2] != 13 {
		t.Fatalf("talents = %v", levels)
	}
	if !panel.Ranks[2].Active || panel.Ranks[3].Active {
		t.Fatal("constellations")
	}
	if _, err = showcase.Parse(t.Context(), app.Game, app.Catalog, []byte(`{"playerInfo":{"nickname":"x"},"ttl":60}`)); !errors.Is(err, gamekit.ErrShowcaseEmpty) {
		t.Fatal(err)
	}
	var failure *gamekit.ShowcaseFailure
	if _, err = showcase.Parse(t.Context(), app.Game, app.Catalog, []byte(`{"ttl":60}`)); !errors.As(err, &failure) {
		t.Fatal(err)
	}
}
