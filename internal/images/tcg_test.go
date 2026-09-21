package images_test

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestTCGDecksFollowYunzai(t *testing.T) {
	data := decode(t, `{"nickname":"旅行者","level":12,"deck_list":[
		{"id":1,"avatar_cards":[{"hp":10,"image":"https://example.invalid/a.png"}],"action_cards":[]},
		{"id":3,"avatar_cards":[{"hp":12}],"action_cards":[{"num":2,"action_cost":[{"cost_type":"CostTypeSame","cost_value":3}]}]}]}`)
	result := gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: data}
	image, ok := images.TCGDecks(gamekit.ImageContext{Game: gamekit.Game{Prefix: "#"}, Word: "七圣查询牌组"}, result)
	decks := image.Data["decks"].([]any)
	if !ok || len(decks) != 2 || decks[1].(map[string]any)["command"] != "#七圣查询卡组3" || image.Data["deck"] != nil {
		t.Fatalf("list = %v", image.Data)
	}
	image, ok = images.TCGDecks(gamekit.ImageContext{Word: "七圣召唤查询卡组3"}, result)
	deck := image.Data["deck"].(map[string]any)
	action := deck["actions"].([]any)[0].(map[string]any)
	if !ok || len(deck["characters"].([]any)) != 1 || action["num"] != "2" || len(action["costs"].([]any)) != 1 {
		t.Errorf("deck 3 = %v", deck)
	}
	// Upstream answers in text when the number names no deck.
	if _, ok := images.TCGDecks(gamekit.ImageContext{Word: "七圣查询牌组5"}, result); ok {
		t.Error("a missing deck drew a page")
	}
}
