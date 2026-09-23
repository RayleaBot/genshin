package images_test

import (
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestGachaCountFollowsYunzai(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	events := []app.PoolInfo{}
	for _, pool := range application.Game.Data.Resources.Pools {
		if pool.Kind == "event" && len(pool.Characters5) > 0 {
			events = append(events, pool)
		}
	}
	first, second := events[0], events[1]
	record := func(id, at, name, kind, rank string) gacha.Record {
		return gacha.Record{ID: id, GachaType: "301", Time: at, Name: name, ItemType: kind, Rank: rank}
	}
	archive := gacha.Archive{Records: []gacha.Record{
		record("1", first.From[:10]+" 20:00:00", "行秋", "角色", "4"),
		record("2", first.From[:10]+" 20:01:00", "行秋", "角色", "4"),
		record("3", first.From[:10]+" 20:02:00", "黎明神剑", "武器", "3"),
		record("4", second.From, "胡桃", "角色", "5"),
	}}
	image, ok := images.GachaCount(app.ImageContext{Game: application.Game, Catalog: application.Catalog}, app.GachaImage{Word: "角色统计", Archive: archive})
	if !ok {
		t.Fatal("not drawn")
	}
	pools := image.Data["pools"].([]any)
	newest, oldest := pools[0].(map[string]any), pools[1].(map[string]any)
	if len(pools) != 2 || newest["count"] != 1 || oldest["count"] != 3 || len(oldest["items"].([]any)) != 1 || oldest["items"].([]any)[0].(map[string]any)["count"] != 2 {
		t.Fatalf("pools = %v", pools)
	}
}
