package images_test

import (
	"reflect"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestMiaoGachaPagesFollowGachaData(t *testing.T) {
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
		record("2", first.From[:10]+" 20:01:00", first.Characters5[0], "角色", "5"),
		record("3", second.From[:10]+" 20:00:00", "迪卢克", "角色", "5"),
		record("4", second.From[:10]+" 20:01:00", "香菱", "角色", "4"),
		record("5", second.From[:10]+" 20:02:00", "香菱", "角色", "4"),
	}}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog}
	stats := func(data map[string]any) []any {
		values := []any{}
		for _, stat := range data["stats"].([]any) {
			values = append(values, stat.(map[string]any)["value"])
		}
		return values
	}

	// Two pulls since the newest five-star; 迪卢克 missed the rate-up and is
	// left out of the pulls per rate-up.
	detail, ok := images.GachaDetail(context, app.GachaImage{Word: "喵喵角色记录", Archive: archive})
	if !ok {
		t.Fatal("detail not drawn")
	}
	if got, want := stats(detail.Data), []any{5, 2, 1, 3, "1.50", "320"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("detail stats = %v, want %v", got, want)
	}
	counts := []any{}
	for _, item := range detail.Data["items"].([]any) {
		entry := item.(map[string]any)
		counts = append(counts, []any{entry["count"], entry["up"], entry["drawn"]})
	}
	if want := []any{[]any{2, true, true}, []any{1, false, false}, []any{2, true, false}}; !reflect.DeepEqual(counts, want) {
		t.Fatalf("detail items = %v, want %v", counts, want)
	}
	if _, ok := images.GachaDetail(context, app.GachaImage{Word: "喵喵武器记录", Archive: archive}); ok {
		t.Fatal("喵喵武器记录 read the character wish")
	}

	stat, ok := images.GachaStat(context, app.GachaImage{Word: "喵喵角色统计", Archive: archive})
	if !ok {
		t.Fatal("stat not drawn")
	}
	if got, want := stats(stat.Data), []any{5, 2, 1, 3, "5.0"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("stat totals = %v, want %v", got, want)
	}
	versions := stat.Data["versions"].([]any)
	newest := versions[0].(map[string]any)
	items := newest["items"].([]any)
	if len(versions) != 2 || newest["version"] != second.Version || stats(newest)[0] != 3 || items[0].(map[string]any)["name"] != "迪卢克" || items[1].(map[string]any)["num"] != 2 {
		t.Fatalf("stat versions = %v", versions)
	}
}
