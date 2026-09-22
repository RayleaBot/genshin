package app

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

func TestOfficialCloudRollRestorationAndPercentAliases(t *testing.T) {
	genshin := testGame(t)
	rolls, err := cloudRolls(genshin, 5, PanelStat{Key: "cpct", Value: "7.8%"}, 1)
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
	if _, err = cloudRolls(genshin, 5, PanelStat{Key: "cpct", Value: "1000%"}, 1); err == nil {
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
	rawPanel := asObject(asList(result.Data["list"])[0])
	id, raw, err := officialCloudAvatar(t.Context(), testGame(t), panel, rawPanel)
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
	_, raw, err := officialCloudAvatar(t.Context(), game, panel, asObject(asList(result.Data["list"])[0]))
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
