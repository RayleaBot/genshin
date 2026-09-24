package images_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/artwork"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

// miao's Character draws the Traveler's face and side from 空's or 荧's
// folder, not 旅行者's, on 面板列表 and 今日素材 alike; 空 shares 荧's
// records.
func TestTravelerPortraitsFollowMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	files := map[string]string{"resources/meta-gs/material/data.json": `{"「诗文」的哲学":{"type":"talent","items":{"「诗文」的教导":{"star":2}}},"狮牙斗士的理想":{"type":"weapon","items":{"狮牙斗士的铁链":{"star":2}}}}`}
	for _, name := range []string{"空", "荧", "旅行者"} {
		for _, kind := range []string{"face", "face-q", "side"} {
			files["resources/meta-gs/character/"+name+"/imgs/"+kind+".webp"] = "webp"
		}
	}
	for name, content := range files {
		file := filepath.Join(root, "miao-plugin", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Artwork: &artwork.Store{Root: root}}
	promote := 6
	traveler := func(id, element string) app.SavedPanel {
		return app.SavedPanel{Panel: app.CharacterPanel{ID: id, Level: 90, Promote: &promote, Rank: 6, RankKnown: true, Element: element,
			Weapon: &app.PanelEquipment{ID: "11303", Name: "旅行剑", Level: 90, Promote: &promote, Refinement: 1, Rarity: "3"}}}
	}
	panels := []app.SavedPanel{traveler("10000005", "anemo"), traveler("10000007", "geo")}
	paths := func(image app.Image) []string {
		out := []string{}
		for _, resource := range image.Resources {
			if strings.Contains(resource.Path, "/meta-gs/character/") {
				out = append(out, strings.TrimPrefix(resource.Path, "assets/miao-plugin/resources/meta-gs/character/"))
			}
		}
		slices.Sort(out)
		return out
	}
	list, ok := images.PanelList(context, app.PanelListImage{UID: "100000001", Panels: panels})
	if got, want := paths(list), []string{"空/imgs/face-q.webp", "荧/imgs/face-q.webp"}; !ok || !slices.Equal(got, want) {
		t.Errorf("面板列表 portraits = %v, want %v", got, want)
	}
	// 「诗文」 books and 狮牙斗士 materials are the third week group's.
	daily, ok := images.DailyMaterial(context, app.DailyMaterialImage{UID: "100000001", Week: 3, Panels: panels})
	if got, want := paths(daily), []string{"空/imgs/face.webp", "空/imgs/side.webp", "荧/imgs/face.webp", "荧/imgs/side.webp"}; !ok || !slices.Equal(got, want) {
		t.Errorf("今日素材 portraits = %v, want %v", got, want)
	}
}
