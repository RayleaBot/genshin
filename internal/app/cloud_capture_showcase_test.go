package app_test

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/showcase"
)

// miao's EnkaData saves a showcase artifact's mainPropId and appendPropIdList
// as they are, and the player data written for exports and the local panel
// rank does the same; a showcase panel kept before the IDs were saved
// restores as many rolls of each substat from its values.
func TestShowcasePanelWritesTheShowcaseRollIDs(t *testing.T) {
	raw, err := os.ReadFile("../showcase/testdata/enka-sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var answer struct {
		AvatarInfoList []struct {
			AvatarID  int `json:"avatarId"`
			EquipList []struct {
				Reliquary *struct {
					Level            int   `json:"level"`
					MainPropID       int   `json:"mainPropId"`
					AppendPropIDList []int `json:"appendPropIdList"`
				} `json:"reliquary"`
				Flat struct {
					EquipType string `json:"equipType"`
					RankLevel int    `json:"rankLevel"`
				} `json:"flat"`
			} `json:"equipList"`
		} `json:"avatarInfoList"`
	}
	if err := json.Unmarshal(raw, &answer); err != nil {
		t.Fatal(err)
	}
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := showcase.Parse(t.Context(), application.Game, application.Catalog, raw)
	if err != nil || len(profile.Panels) != len(answer.AvatarInfoList) {
		t.Fatalf("parsed %d panels, %v", len(profile.Panels), err)
	}
	slots := map[string]string{"EQUIP_BRACER": "1", "EQUIP_NECKLACE": "2", "EQUIP_SHOES": "3", "EQUIP_RING": "4", "EQUIP_DRESS": "5"}
	type written struct {
		Source string `json:"_source"`
		Artis  map[string]struct {
			Level   int   `json:"level"`
			Star    int   `json:"star"`
			MainID  int   `json:"mainId"`
			AttrIDs []any `json:"attrIds"`
		} `json:"artis"`
	}
	write := func(panel app.CharacterPanel) written {
		_, raw, err := app.PanelCloudAvatar(t.Context(), application.Game, panel, app.MiaoSource(panel))
		if err != nil {
			t.Fatal(err)
		}
		var avatar written
		if err := json.Unmarshal(raw, &avatar); err != nil {
			t.Fatal(err)
		}
		return avatar
	}
	for index, avatar := range answer.AvatarInfoList {
		panel := profile.Panels[index]
		if panel.ID != strconv.Itoa(avatar.AvatarID) {
			t.Fatalf("panel %d is %s, want %d", index, panel.ID, avatar.AvatarID)
		}
		got := write(panel)
		if got.Source != "enka" || len(got.Artis) != 5 {
			t.Fatalf("%s: _source %q with %d artifacts", panel.ID, got.Source, len(got.Artis))
		}
		for _, equip := range avatar.EquipList {
			if equip.Reliquary == nil {
				continue
			}
			arti := got.Artis[slots[equip.Flat.EquipType]]
			ids := []int{}
			for _, id := range arti.AttrIDs {
				// Numbers, as EnkaData saves them.
				number, ok := id.(float64)
				if !ok {
					t.Fatalf("%s: roll ID %#v is not a number", panel.ID, id)
				}
				ids = append(ids, int(number))
			}
			if !slices.Equal(ids, equip.Reliquary.AppendPropIDList) || arti.MainID != equip.Reliquary.MainPropID || arti.Level != min(20, equip.Reliquary.Level-1) || arti.Star != equip.Flat.RankLevel {
				t.Errorf("%s %s = %+v, want %v main %d", panel.ID, equip.Flat.EquipType, arti, equip.Reliquary.AppendPropIDList, equip.Reliquary.MainPropID)
			}
		}
		// Without the IDs each substat restores as many rolls as it had.
		for piece := range panel.Equipment {
			panel.Equipment[piece].MainID, panel.Equipment[piece].AttrIDs = 0, nil
		}
		restored := write(panel)
		for _, equip := range avatar.EquipList {
			if equip.Reliquary == nil {
				continue
			}
			counts := func(ids []any) map[int]int {
				out := map[int]int{}
				for _, id := range ids {
					value, _ := strconv.Atoi(app.Text(id))
					out[value/10%1000]++
				}
				return out
			}
			want := []any{}
			for _, id := range equip.Reliquary.AppendPropIDList {
				want = append(want, float64(id))
			}
			if got, want := counts(restored.Artis[slots[equip.Flat.EquipType]].AttrIDs), counts(want); !maps.Equal(got, want) {
				t.Errorf("%s %s restored rolls %v, want %v", panel.ID, equip.Flat.EquipType, got, want)
			}
		}
	}
}
