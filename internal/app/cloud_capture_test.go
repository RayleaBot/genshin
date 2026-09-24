package app

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

func TestOfficialCloudRollRestorationAndPercentAliases(t *testing.T) {
	genshin := testGame(t)
	rolls, err := cloudRolls(genshin, 5, PanelStat{Key: "cpct", Value: "7.8%", Times: 1})
	if err != nil || len(rolls) != 2 {
		t.Fatal(rolls, err)
	}
	total := 0.0
	for _, v := range rolls {
		total += genshin.Data.CloudGear.AttrIDMap[asText(v)].Value * 100
	}
	if math.Abs(total-7.78) > 0.001 {
		t.Fatal(total)
	}
	if _, err = cloudRolls(genshin, 5, PanelStat{Key: "cpct", Value: "1000%", Times: 1}); err == nil {
		t.Fatal("unrepresentable stat exported")
	}
}
func TestOfficialCloudCaptureKeepsOriginalTalentAndWeaponIdentity(t *testing.T) {
	caller := &ocrCaller{}
	client := AccountsClient{Game: "genshin", Caller: caller}
	result, err := client.Execute(t.Context(), Selection{"account", "role"}, "genshin.character", map[string]any{"character_ids": []any{"10000046"}})
	if err != nil {
		t.Fatal(err)
	}
	panel := NormalizePanels(result, Catalog{})[0]
	panel.Rank = 6
	id, raw, err := panelCloudAvatar(t.Context(), testGame(t), panel, miaoSource(panel))
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	json.Unmarshal(raw, &exported)
	if id != "10000046" || asText(fieldAt(exported, "weapon.name")) != "护摩之杖" || number(fieldAt(exported, "talent.e")) != 7 || number(fieldAt(exported, "talent.q")) != 7 || asText(exported["_time"]) == "" {
		t.Fatal(exported)
	}
}
func TestCloudCapturePromotionResolutionUsesPinnedBaseAttributes(t *testing.T) {
	for _, spec := range []struct{ game, key string }{{"genshin", "gs_10000046"}} {
		key, engine := spec.key, calcEngine(t)
		metadata := engine.Metadata()
		var cases []struct {
			Key   string
			Input BuildProfile
		}
		if err := json.Unmarshal(pluginFile(t, "internal/assets/testdata/calc-vectors.json"), &cases); err != nil {
			t.Fatal(err)
		}
		var profile BuildProfile
		var record reference.Character
		for _, c := range cases {
			if c.Key == key && c.Input.Rank == 0 {
				profile = c.Input
				break
			}
		}
		for _, r := range metadata.Characters {
			if r.Key == key {
				record = r
				break
			}
		}
		profile.Promote = nil
		profile.Weapon.Promote = nil
		cp, wp, err := resolveCloudPromotions(t.Context(), engine, record, profile)
		if err != nil || cp != 6 || wp != 6 {
			t.Fatal(key, cp, wp, err)
		}
	}
}

func TestImportedCloudAvatarBecomesAPanel(t *testing.T) {
	caller := &ocrCaller{}
	client := AccountsClient{Game: "genshin", Caller: caller}
	result, err := client.Execute(t.Context(), Selection{"account", "role"}, "genshin.character", map[string]any{"character_ids": []any{"10000046"}})
	if err != nil {
		t.Fatal(err)
	}
	game := testGame(t)
	panel := NormalizePanels(result, Catalog{})[0]
	panel.Rank = 6
	_, raw, err := panelCloudAvatar(t.Context(), game, panel, miaoSource(panel))
	if err != nil {
		t.Fatal(err)
	}
	// 导入面板数据 reads the exported player data back into a panel whose
	// properties come from its parts.
	a := App{Game: game}
	imported, err := a.cloudAvatarPanel(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if imported.ID != "10000046" || imported.Level != 90 || imported.Rank != 6 || imported.Source != "share" || imported.Weapon == nil || imported.Weapon.Name != "护摩之杖" || imported.Weapon.Level != 90 || imported.Weapon.Refinement != 1 || len(imported.Stats) == 0 || len(imported.Weapon.Main) == 0 {
		t.Fatalf("%+v", imported)
	}
}

// ark's 导出面板 and 导出面板数据 send miao's whole player data, whatever each
// character was read from. The showcase keeps rounded substats with how
// often each rolled; a panel read back from player data keeps their sums.
func TestExportWritesEveryKeptPanel(t *testing.T) {
	game := testGame(t)
	a := App{Game: game}
	source := `{"id":10000030,"elem":"geo","level":90,"promote":6,"cons":0,"talent":{"a":9,"e":10,"q":10},"weapon":{"name":"护摩之杖","level":90,"promote":6,"affix":1},"artis":{
		"1":{"name":"角斗士的留恋","star":5,"level":20,"mainId":14001,"attrIds":["501204","501203","501204","501224","501223","501222","501061","501064","501241"]},
		"2":{"name":"角斗士的归宿","star":5,"level":20,"mainId":12001,"attrIds":["501202","501204","501224","501224","501223","501221","501031","501094"]},
		"3":{"name":"角斗士的希冀","star":5,"level":20,"mainId":10004,"attrIds":["501201","501222","501224","501241","501243","501244","501231","501234","501233"]},
		"4":{"name":"角斗士的酣醉","star":5,"level":20,"mainId":15013,"attrIds":["501203","501203","501203","501203","501221","501062","501081","501022"]},
		"5":{"name":"角斗士的凯旋","star":5,"level":20,"mainId":13007,"attrIds":["501224","501221","501222","501223","501061","501063","501021","501024","501242"]}}}`
	rolls := func(raw json.RawMessage) map[string]map[string][]float64 {
		var avatar map[string]any
		if err := json.Unmarshal(raw, &avatar); err != nil {
			t.Fatal(err)
		}
		out := map[string]map[string][]float64{}
		for slot, gear := range asObject(avatar["artis"]) {
			out[slot] = map[string][]float64{}
			for _, id := range asList(asObject(gear)["attrIds"]) {
				attr := game.Data.CloudGear.AttrIDMap[asText(id)]
				out[slot][attr.Key] = append(out[slot][attr.Key], attr.Value)
			}
		}
		return out
	}
	want := rolls(json.RawMessage(source))
	imported, err := a.cloudAvatarPanel(t.Context(), json.RawMessage(source))
	if err != nil {
		t.Fatal(err)
	}
	_, raw, err := panelCloudAvatar(t.Context(), game, imported, miaoSource(imported))
	if err != nil {
		t.Fatal(err)
	}
	sum := func(list []float64) (total float64) {
		for _, v := range list {
			total += v
		}
		return total
	}
	got := rolls(raw)
	if len(got) != len(want) {
		t.Fatalf("imported panel wrote %d of %d artifacts", len(got), len(want))
	}
	for slot, keys := range got {
		for key, values := range keys {
			if math.Abs(sum(values)-sum(want[slot][key])) > 1e-6*sum(want[slot][key]) {
				t.Errorf("imported slot %s %s = %v, want the sum of %v", slot, key, values, want[slot][key])
			}
		}
	}
	showcase, err := a.cloudAvatarPanel(t.Context(), json.RawMessage(source))
	if err != nil {
		t.Fatal(err)
	}
	showcase.Source = "enka"
	for _, gear := range showcase.Equipment {
		for i, stat := range gear.Sub {
			value, _ := buildStatNumber(stat.Value)
			if strings.HasSuffix(stat.Value, "%") {
				stat.Value = strconv.FormatFloat(value, 'f', 1, 64) + "%"
			} else {
				stat.Value = strconv.FormatFloat(math.Round(value), 'f', 0, 64)
			}
			stat.Times = len(want[strconv.Itoa(gear.Slot)][stat.Key]) - 1
			gear.Sub[i] = stat
		}
	}
	result, err := AccountsClient{Game: "genshin", Caller: &ocrCaller{}}.Execute(t.Context(), Selection{"account", "role"}, "genshin.character", map[string]any{"character_ids": []any{"10000046"}})
	if err != nil {
		t.Fatal(err)
	}
	official := NormalizePanels(result, Catalog{})[0]
	saved := SavedProfiles{Panels: map[string]SavedPanel{"10000046": {Panel: official, Official: official.Official}, "10000030": {Panel: showcase}}}
	avatars := a.exportAvatars(t.Context(), saved)
	if len(avatars) != 2 || avatars["10000046"] == nil {
		t.Fatalf("exported %d of 2 panels", len(avatars))
	}
	// Each keeps the _source miao saves its source under.
	for id, want := range map[string]string{"10000046": "mysPanel", "10000030": "enka"} {
		var avatar map[string]any
		if err := json.Unmarshal(avatars[id], &avatar); err != nil || avatar["_source"] != want {
			t.Errorf("%s _source = %v, want %s", id, avatar["_source"], want)
		}
	}
	got = rolls(avatars["10000030"])
	if len(got) != len(want) {
		t.Fatalf("showcase panel wrote %d of %d artifacts", len(got), len(want))
	}
	for slot, keys := range got {
		for key, values := range keys {
			if len(values) != len(want[slot][key]) {
				t.Errorf("showcase slot %s %s rolled %d times, want %d", slot, key, len(values), len(want[slot][key]))
			}
		}
	}
}
