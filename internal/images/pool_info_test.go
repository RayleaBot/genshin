package images_test

import (
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestPoolInfoKeepsTheItemsRowUnlessDetailed(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hutao, _ := application.Catalog.Resolve("胡桃", "character", nil)
	pool := app.PoolInfo{Version: "3.4", Half: "下半", Kind: "event", Characters5: []string{"胡桃", "夜兰"}, Characters4: []string{"行秋", "北斗", "烟绯"}, Weapons5: []string{"护摩之杖", "若水"}}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog}
	groups := func(detail bool) []any {
		image, ok := images.PoolInfo(context, app.PoolImage{Pools: []app.PoolInfo{pool}, Item: hutao, Detail: detail})
		if !ok {
			t.Fatal("not drawn")
		}
		return image.Data["pools"].([]any)[0].(map[string]any)["groups"].([]any)
	}
	simple := groups(false)
	if len(simple) != 1 || simple[0].(map[string]any)["title"] != "五星角色" || len(simple[0].(map[string]any)["rows"].([]any)[0].([]any)) != 2 {
		t.Fatalf("simple = %v", simple)
	}
	if detailed := groups(true); len(detailed) != 3 {
		t.Fatalf("detailed = %v", detailed)
	}
}
