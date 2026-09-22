package images_test

import (
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestAbyssFloorFollowsYunzai(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	half := func(index string) string {
		return `{"index":` + index + `,"timestamp":"1788300000","avatars":[{"id":10000046,"level":90,"rarity":5},{"id":10000089,"level":90,"rarity":5}]}`
	}
	data := decode(t, `{"floors":[{"index":11,"star":9,"max_star":9,"levels":[]},{"index":12,"star":7,"max_star":9,"levels":[
		{"index":1,"star":3,"battles":[`+half("1")+`,`+half("2")+`]},{"index":2,"star":2,"battles":[`+half("1")+`,`+half("2")+`]},{"index":3,"star":0,"battles":[`+half("1")+`]}]}]}`)
	context := app.ImageContext{Game: application.Game, Word: "上期深渊十二层", Query: func(string, map[string]any) (app.QueryResult, error) {
		return app.QueryResult{Data: decode(t, `{"avatars":[{"id":10000046,"actived_constellation_num":1}]}`)}, nil
	}}
	image, ok := images.AbyssFloor(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok || image.Data["floor"] != 12 || image.Data["star"] != "7" {
		t.Fatalf("image = %v", image.Data)
	}
	// A chamber with a single half is left out, as upstream.
	rooms := image.Data["rooms"].([]any)
	second := rooms[1].(map[string]any)
	if len(rooms) != 2 || second["time"] != "2026-09-02 06:00:00" || second["stars"].([]any)[1] != true || second["stars"].([]any)[2] != false {
		t.Errorf("rooms = %v", rooms)
	}
	avatars := rooms[0].(map[string]any)["battles"].([]any)[0].(map[string]any)["avatars"].([]any)
	if hutao := avatars[0].(map[string]any); hutao["life"] != 1 || hutao["name"] != "胡桃" || avatars[1].(map[string]any)["life"] != 0 {
		t.Errorf("avatars = %v", avatars)
	}
	context.Word = "深渊十层"
	if _, ok := images.AbyssFloor(context, app.QueryResult{Data: data}); ok {
		t.Error("a floor without record answers in text like upstream")
	}
	if images.Queries()["genshin.abyss_floor"](time.Now()) != "genshin.abyss" {
		t.Error("a floor draws on the abyss record")
	}
}
