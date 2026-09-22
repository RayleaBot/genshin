package images_test

import (
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestTCGDecksFollowYunzai(t *testing.T) {
	data := decode(t, `{"nickname":"旅行者","level":12,"deck_list":[
		{"id":1,"avatar_cards":[{"hp":10,"image":"https://example.invalid/a.png"}],"action_cards":[]},
		{"id":3,"avatar_cards":[{"hp":12}],"action_cards":[{"num":2,"action_cost":[{"cost_type":"CostTypeSame","cost_value":3}]}]}]}`)
	result := app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data}
	image, ok := images.TCGDecks(app.ImageContext{Game: app.Game{Prefix: "#"}, Word: "七圣查询牌组"}, result)
	decks := image.Data["decks"].([]any)
	if !ok || len(decks) != 2 || decks[1].(map[string]any)["command"] != "#七圣查询卡组3" || image.Data["deck"] != nil {
		t.Fatalf("list = %v", image.Data)
	}
	image, ok = images.TCGDecks(app.ImageContext{Word: "七圣召唤查询卡组3"}, result)
	deck := image.Data["deck"].(map[string]any)
	action := deck["actions"].([]any)[0].(map[string]any)
	if !ok || len(deck["characters"].([]any)) != 1 || action["num"] != "2" || len(action["costs"].([]any)) != 1 {
		t.Errorf("deck 3 = %v", deck)
	}
	// Upstream answers in text when the number names no deck.
	if _, ok := images.TCGDecks(app.ImageContext{Word: "七圣查询牌组5"}, result); ok {
		t.Error("a missing deck drew a page")
	}
}

func TestTCGCardsKeepsOwnedCardsOfTheKindAsked(t *testing.T) {
	cards := []any{
		map[string]any{"num": 1, "hp": 10, "proficiency": 3, "use_count": 5, "image": "https://example.com/a.png"},
		map[string]any{"num": 0, "hp": 10, "image": "https://example.com/b.png"},
		map[string]any{"num": 2, "use_count": 7, "image": "https://example.com/c.png", "action_cost": []any{map[string]any{"cost_type": "CostTypeSame", "cost_value": "2"}}},
	}
	result := app.QueryResult{Role: app.Role{UID: "100000001"}, Data: map[string]any{"card_list": cards}}
	image, ok := images.TCGCards(app.ImageContext{Word: "七圣查询牌"}, result)
	characters, actions := image.Data["characters"].([]any), image.Data["actions"].([]any)
	if !ok || len(characters) != 1 || len(actions) != 1 || characters[0].(map[string]any)["wins"] != 3 || actions[0].(map[string]any)["num"] != 2 {
		t.Fatalf("cards = %v / %v", characters, actions)
	}
	// 角色 keeps only the character cards, as upstream then skips the action list.
	image, _ = images.TCGCards(app.ImageContext{Word: "七圣查询角色牌"}, result)
	if len(image.Data["actions"].([]any)) != 0 || image.Data["show_actions"] != false || len(image.Data["characters"].([]any)) != 1 {
		t.Fatalf("character cards = %v", image.Data)
	}
}
