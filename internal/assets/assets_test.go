package assets

import (
	"slices"
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
)

// The plugin starts exactly this way; a manifest or data change that the shared
// library rejects must fail here rather than when the host launches it.
func TestPluginStartsFromEmbeddedAssets(t *testing.T) {
	application, err := app.New(Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	application.Close()
}

func TestXiaoyaoOverloadedBlessingAliases(t *testing.T) {
	application, err := app.New(Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	want := map[string]string{"超载祝佑·霆击": "超载霆击", "元素幻变：超载祝佑": "元素超载", "超载祝佑·追燃": "超载追燃"}
	for _, group := range application.Game.Data.Xiaoyao.Cards {
		for _, card := range group.Cards {
			if alias, ok := want[card.Name]; ok {
				if !slices.Contains(card.Aliases, alias) {
					t.Fatalf("%s is missing alias %s", card.Name, alias)
				}
				delete(want, card.Name)
			}
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing imported cards: %v", want)
	}
}
