package images_test

import (
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestAbyssSummaryFollowsMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Floor 12 comes first and its chambers out of order; Hu Tao holds the
	// strongest hit of exactly 1.25 W, which miao's toFixed rounds up.
	data := decode(t, `{"start_time":"1788220800","total_battle_times":12,"floors":[
		{"index":12,"star":8,"levels":[
			{"index":3,"star":2,"battles":[{"index":1,"timestamp":"1788300000","avatars":[{"id":10000046},{"id":10000089}]},{"index":2,"timestamp":"1788300100","avatars":[{"id":10000025}]}]},
			{"index":1,"star":3,"battles":[{"index":1,"timestamp":"1788200000","avatars":[{"id":10000089}]}]}]},
		{"index":11,"star":9,"levels":[{"index":1,"star":3,"battles":[]}]}],
		"damage_rank":[{"avatar_id":10000046,"value":12500}],"take_damage_rank":[],
		"defeat_rank":[{"avatar_id":10000089,"value":25}],"normal_skill_rank":[{"avatar_id":10000046,"value":30}],"energy_skill_rank":[]}`)
	asked := [][]any{}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Date(2026, 9, 23, 4, 0, 0, 0, time.UTC),
		Query: func(operation string, input map[string]any) (app.QueryResult, error) {
			asked = append(asked, input["character_ids"].([]any))
			return app.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000046,"level":90,"actived_constellation_num":1},
				"skills":[{"skill_id":10461,"skill_type":1,"level":10},{"skill_id":10462,"skill_type":1,"level":10},{"skill_id":10463,"skill_type":1,"level":10}]},
				{"base":{"id":10000089,"level":90,"actived_constellation_num":2}}]}`)}, nil
		}}
	image, ok := images.AbyssSummary(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok {
		t.Fatal("abyss summary")
	}
	if len(asked) != 1 || len(asked[0]) != 3 {
		t.Errorf("asked = %v", asked)
	}
	if image.Data["month"] != "9月" || image.Data["total"] != 12 || image.Data["updated"] != "09-23 12:00:00" {
		t.Errorf("head = %v %v %v", image.Data["month"], image.Data["total"], image.Data["updated"])
	}
	// The damage taken record has no character and is left out; the empty
	// burst record keeps its box.
	stats := image.Data["stats"].([]any)
	strongest, defeat, burst := stats[0].(map[string]any), stats[1].(map[string]any), stats[3].(map[string]any)
	if len(stats) != 4 || strongest["value"] != "1.3 W" || strongest["name"] != "胡桃" || strongest["shown"] != true || strongest["notes"] == nil {
		t.Errorf("strongest = %v", strongest)
	}
	if defeat["title"] != "最多击破" || defeat["value"] != "25次" || defeat["shown"] != true || defeat["notes"] != nil {
		t.Errorf("defeat = %v", defeat)
	}
	if burst["title"] != "元素爆发" || burst["shown"] != nil {
		t.Errorf("burst = %v", burst)
	}
	// Floors and chambers in order; a floor shows its last chamber's teams,
	// and characters the query does not return get an empty card.
	floors := image.Data["floors"].([]any)
	eleven, twelve := floors[0].(map[string]any), floors[1].(map[string]any)
	if eleven["index"] != 11 || len(eleven["up"].([]any)) != 0 || twelve["index"] != 12 {
		t.Errorf("floors = %v", floors)
	}
	up, down := twelve["up"].([]any), twelve["down"].([]any)
	if len(up) != 2 || up[0].(map[string]any)["name"] != "胡桃" || up[0].(map[string]any)["type"] != "wide" || len(down) != 1 || down[0].(map[string]any)["known"] != nil {
		t.Errorf("teams = %v %v", up, down)
	}
	chambers := twelve["chambers"].([]any)
	first, third := chambers[0].(map[string]any), chambers[1].(map[string]any)
	if first["index"] != 1 || first["time"] != "09-01 02:13:20" || third["index"] != 3 || third["star"] != 2 || len(third["down"].([]any)) != 1 || third["down"].([]any)[0].(map[string]any)["star"] != 5 {
		t.Errorf("chambers = %v", chambers)
	}
	// Upstream answers in text until the first chamber has battles.
	delete(data["floors"].([]any)[0].(map[string]any)["levels"].([]any)[0].(map[string]any), "battles")
	if _, ok := images.AbyssSummary(context, app.QueryResult{Data: data}); ok {
		t.Error("a first chamber without battles should answer in text")
	}
}
