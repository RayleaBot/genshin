package images_test

import (
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestUIDListFollowsYunzai(t *testing.T) {
	image, ok := images.UIDList(app.ImageContext{Game: app.Game{Prefix: "#"}}, app.UIDListImage{Entries: []app.UIDListEntry{
		{UID: "100000001", Account: true, Nickname: "旅行者", Level: 60},
		{UID: "100000002", Active: true, Nickname: "只有昵称"},
	}})
	if !ok || image.Template != "uid-list" || image.Data["mark"] != "#" {
		t.Fatal(image)
	}
	uids := image.Data["uids"].([]any)
	first, second := uids[0].(map[string]any), uids[1].(map[string]any)
	// Upstream shows the player only with both name and level.
	if first["type"] != "ck" || first["name"] != "旅行者" || first["index"] != 1 || second["type"] != "reg" || second["active"] != true || second["name"] != nil {
		t.Fatal(uids)
	}
}
