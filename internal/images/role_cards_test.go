package images_test

import (
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestRoleCardsFollowMiao(t *testing.T) {
	cards := []app.RoleCard{{Name: "月神", Count: 2, Unlocked: true, Icon: "https://example.com/1.png"}, {Name: "旅人", Count: 0, Unlocked: false}}
	image, ok := images.RoleCards(app.ImageContext{}, app.RoleCardsImage{UID: "100000001", Collected: 1, Total: 2, Cards: cards})
	if !ok || image.Template != "stat-role-card" || image.Data["collected"] != 1 {
		t.Fatalf("image = %v", image)
	}
	// A locked card shows no picture.
	list := image.Data["cards"].([]any)
	if list[0].(map[string]any)["unlocked"] != true || list[0].(map[string]any)["count"] != 2 || list[1].(map[string]any)["unlocked"] != nil {
		t.Errorf("cards = %v", list)
	}

	matches := []app.RoleCardMatch{{ActorID: "10001", Nickname: "一二三四五六七八九十十一十二", UID: "100000002", Score: 2,
		Receive: []app.RoleCard{{Name: "旅人", Count: 3, Unlocked: true}}, Give: []app.RoleCard{{Name: "月神", Count: 2, Unlocked: true}}},
		{ActorID: "10002", UID: "100000003", Score: 2}}
	image, ok = images.RoleCards(app.ImageContext{}, app.RoleCardsImage{UID: "100000001", Exchange: true, Group: "20001", Matches: matches})
	if !ok || image.Template != "stat-role-card-exchange" || image.Data["group"] != "20001" {
		t.Fatalf("image = %v", image)
	}
	// Names keep twelve characters as lodash's truncate, a member without
	// one shows the ID, and each card shows how many its giver can spare.
	first, second := image.Data["matches"].([]any)[0].(map[string]any), image.Data["matches"].([]any)[1].(map[string]any)
	if first["name"] != "一二三四五六七八九..." || first["rank"] != 1 || second["name"] != "10002" {
		t.Errorf("matches = %v %v", first, second)
	}
	if receive := first["receive"].([]any)[0].(map[string]any); receive["num"] != 2 || receive["name"] != "旅人" {
		t.Errorf("receive = %v", receive)
	}
	if give := first["give"].([]any)[0].(map[string]any); give["num"] != 1 {
		t.Errorf("give = %v", give)
	}
}
