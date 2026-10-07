package images_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/artwork"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
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

// Yunzai's abyss pages take portraits from miao's Character: 空 and 荧 for
// the Traveler. miao keeps no side portrait of its newest characters (阿罗夏,
// 奥黛塔) and no pictures of characters newer than its data; the icon the
// official answer links stands in then.
func TestAbyssPortraitsFallBackToOfficialIcons(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("png"))
	}))
	defer server.Close()
	// The portraits miao has of these characters; it has no side portrait of
	// 阿罗夏 or 奥黛塔.
	root := t.TempDir()
	for _, name := range []string{"空/imgs/side", "荧/imgs/face", "胡桃/imgs/side", "胡桃/imgs/face", "阿罗夏/imgs/face", "奥黛塔/imgs/face"} {
		file := filepath.Join(root, "miao-plugin", "resources", "meta-gs", "character", filepath.FromSlash(name)+".webp")
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("webp"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := &artwork.Store{Root: root, Sources: []artwork.Source{{ID: "mihoyo", Mirrors: []string{server.URL + "/"}}}}
	context := app.ImageContext{Game: application.Game, Artwork: store}
	paths := func(image app.Image) map[string]string {
		out := map[string]string{}
		for _, resource := range image.Resources {
			out[resource.ID] = resource.Path
		}
		return out
	}
	rank := func(id string) []any {
		return []any{map[string]any{"avatar_id": json.Number(id), "value": json.Number("1"), "avatar_icon": "https://icons.test/side-" + id + ".png"}}
	}
	data := map[string]any{"start_time": "1788220800", "total_battle_times": json.Number("12"),
		"damage_rank": rank("10000148"), "take_damage_rank": rank("10000150"), "defeat_rank": rank("10000005"), "normal_skill_rank": rank("10000046"), "energy_skill_rank": rank("10000999"),
		"reveal_rank": []any{map[string]any{"avatar_id": json.Number("10000148"), "value": json.Number("8"), "avatar_icon": "https://icons.test/face-10000148.png"},
			map[string]any{"avatar_id": json.Number("10000999"), "value": json.Number("2"), "avatar_icon": "https://icons.test/face-10000999.png"}}}
	image, ok := images.Abyss(context, app.QueryResult{Data: data})
	if !ok {
		t.Fatal("abyss")
	}
	files := paths(image)
	ranks := image.Data["ranks"].(map[string]any)
	for key, want := range map[string]string{"damage": "assets/mihoyo/icons.test/side-10000148.png", "take_damage": "assets/mihoyo/icons.test/side-10000150.png",
		"defeat": "assets/miao-plugin/resources/meta-gs/character/空/imgs/side.webp", "normal_skill": "assets/miao-plugin/resources/meta-gs/character/胡桃/imgs/side.webp",
		"energy_skill": "assets/mihoyo/icons.test/side-10000999.png"} {
		if got := files[ranks[key].(map[string]any)["icon"].(string)]; got != want {
			t.Errorf("%s icon = %q, want %q", key, got, want)
		}
	}
	used := image.Data["list"].([]any)
	for index, want := range []string{"assets/miao-plugin/resources/meta-gs/character/阿罗夏/imgs/face.webp", "assets/mihoyo/icons.test/face-10000999.png"} {
		if got := files[used[index].(map[string]any)["icon"].(string)]; got != want {
			t.Errorf("used %d icon = %q, want %q", index, got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "mihoyo", "icons.test", "side-10000046.png")); err == nil {
		t.Error("an official icon was fetched for a character miao draws")
	}
	floor := decode(t, `{"floors":[{"index":12,"star":9,"max_star":9,"levels":[{"index":1,"star":3,"battles":[
		{"index":1,"timestamp":"1788300000","avatars":[{"id":10000007,"level":90,"rarity":5},{"id":10000999,"level":90,"rarity":5,"icon":"https://icons.test/face-10000999.png"}]},
		{"index":2,"timestamp":"1788300000","avatars":[{"id":10000046,"level":90,"rarity":5}]}]}]}]}`)
	context.Word = "深渊十二层"
	image, ok = images.AbyssFloor(context, app.QueryResult{Data: floor})
	if !ok {
		t.Fatal("abyss floor")
	}
	files = paths(image)
	avatars := image.Data["rooms"].([]any)[0].(map[string]any)["battles"].([]any)[0].(map[string]any)["avatars"].([]any)
	for index, want := range []string{"assets/miao-plugin/resources/meta-gs/character/荧/imgs/face.webp", "assets/mihoyo/icons.test/face-10000999.png"} {
		if got := files[avatars[index].(map[string]any)["icon"].(string)]; got != want {
			t.Errorf("floor avatar %d icon = %q, want %q", index, got, want)
		}
	}
}
