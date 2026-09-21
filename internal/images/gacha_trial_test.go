package images

import (
	"os"
	"path/filepath"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/artwork"
)

func TestGachaTrialOrdersDrawsLikeYunzai(t *testing.T) {
	root := t.TempDir()
	files := []string{"miao-plugin/resources/meta-gs/weapon/bow/弹弓/gacha.webp", "miao-plugin/resources/meta-gs/weapon/catalyst/流浪的晚星/gacha.webp"}
	for _, icon := range []string{"雷", "冰", "法器", "弓"} {
		files = append(files, "yunzai-genshin/resources/img/gacha/items/"+icon+".png")
	}
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("webp"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	catalog := gamekit.Catalog{Entries: []gamekit.Entry{
		{ID: "10000052", Name: "雷电将军", Abbr: "雷神", Kind: "character", Element: "雷"},
		{ID: "10000074", Name: "莱依拉", Aliases: []string{"莱伊拉"}, Kind: "character", Element: "冰"},
		{ID: "14414", Name: "流浪的晚星", Aliases: []string{"流浪晚星"}, Kind: "weapon"},
	}}
	context := gamekit.ImageContext{Artwork: &artwork.Store{Root: root}, Catalog: catalog}
	draws := []gamekit.SimulationDraw{
		{Name: "弹弓", Rarity: 3, Item: "weapon"},
		{Name: "流浪晚星", Rarity: 4, Item: "weapon"},
		{Name: "莱伊拉", Rarity: 4, Item: "character"},
		{Name: "莱依拉", Rarity: 4, Item: "character"},
		{Name: "雷电将军", Rarity: 5, Item: "character", Interval: 73, Guaranteed: true},
	}
	image, ok := GachaTrial(context, gamekit.SimulationImage{Selection: gamekit.SimulationSelection{Kind: "character", Featured: "雷电将军"}, Draws: draws, Pity: gamekit.SimulationPity{Five: 0}})
	if !ok {
		t.Fatal("no image")
	}
	// Five-stars first, then characters before weapons and first copies
	// before repeats; the alias names the same character, so it repeats.
	list := image.Data["list"].([]any)
	want := []struct {
		element string
		have    bool
		times   any
	}{{"electro", false, 73}, {"cryo", false, nil}, {"", true, nil}, {"catalyst", false, nil}, {"bow", false, nil}}
	for index, item := range list {
		entry := item.(map[string]any)
		element, _ := entry["element"].(string)
		if entry["have"] != want[index].have || entry["times"] != want[index].times || element != want[index].element {
			t.Errorf("list[%d] = %v", index, entry)
		}
	}
	if image.Data["info"] != "雷神「73抽」大保底" || image.Data["pool"] != "角色池：雷神" || image.Data["header"] != true {
		t.Errorf("header = %v %v %v", image.Data["info"], image.Data["pool"], image.Data["header"])
	}
	// Without a five-star the corner shows the pity count; the weapon pool
	// shows the Epitomized Path instead of the pool.
	image, _ = GachaTrial(context, gamekit.SimulationImage{Selection: gamekit.SimulationSelection{Kind: "weapon", FateTarget: "流浪晚星"}, Draws: draws[:1], Pity: gamekit.SimulationPity{Five: 12, Fate: 1}})
	if image.Data["info"] != "累计「12抽」" || image.Data["bing"] != "流浪的晚星" || image.Data["life"] != 1 || image.Data["pool"] != nil {
		t.Errorf("weapon header = %v", image.Data)
	}
}
