package images

import (
	"regexp"
	"strings"

	"github.com/RayleaBot/genshin/internal/app"
)

// tcgArtwork maps the images the Genius Invokation TCG pages name to
// Yunzai's paths; the digits serve both character HP and dice costs.
var tcgArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"genshin-logo", "resources/img/other/原神.png"},
	{"deck-frame", "resources/img/deck/边框.png"},
	{"deck-container", "resources/img/deck/容器.png"},
}

var tcgDeckWord = regexp.MustCompile(`([0-9]{1,2})$`)

// tcgResources adds a Yunzai deck image the first time a page names it.
type tcgResources struct{ app.ImageResources }

func newTCGResources(context app.ImageContext) *tcgResources {
	resources := &tcgResources{app.ImageResources{Context: context}}
	for _, item := range tcgArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	return resources
}

// deck is one of Yunzai's img/deck images, a digit or a dice cost type.
func (r *tcgResources) deck(name string) string {
	return r.Artwork("deck-"+name, "yunzai-genshin", "resources/img/deck/"+name+".png")
}

// characters are a list's character cards with their HP.
func (r *tcgResources) characters(list []any) []any {
	cards := []any{}
	for _, raw := range list {
		card, _ := raw.(map[string]any)
		cards = append(cards, map[string]any{"hp": r.deck(app.Text(card["hp"])), "image": r.URL("mihoyo", card["image"])})
	}
	return cards
}

// actions are a list's action cards with their dice costs.
func (r *tcgResources) actions(list []any) []any {
	cards := []any{}
	for _, raw := range list {
		card, _ := raw.(map[string]any)
		costs := []any{}
		costList, _ := card["action_cost"].([]any)
		for _, rawCost := range costList {
			cost, _ := rawCost.(map[string]any)
			costs = append(costs, map[string]any{"type": r.deck(app.Text(cost["cost_type"])), "value": r.deck(app.Text(cost["cost_value"]))})
		}
		cards = append(cards, map[string]any{"num": app.Text(card["num"]), "costs": costs, "image": r.URL("mihoyo", card["image"])})
	}
	return cards
}

// TCGDecks draws 七圣召唤查询牌组 the way Yunzai's deckList and deck pages
// do: every deck's character cards with a note naming the command for it, or,
// when the command ends in a deck number, that deck's character cards and
// action cards with their costs and copies. Upstream answers in text when the
// number names no deck.
func TCGDecks(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	decks, _ := result.Data["deck_list"].([]any)
	resources := newTCGResources(context)
	data := map[string]any{"uid": result.Role.UID, "nickname": app.Text(result.Data["nickname"]), "level": app.Text(result.Data["level"])}
	if match := tcgDeckWord.FindStringSubmatch(context.Word); match != nil {
		for _, raw := range decks {
			deck, _ := raw.(map[string]any)
			if app.Text(deck["id"]) != match[1] {
				continue
			}
			characters, _ := deck["avatar_cards"].([]any)
			actions, _ := deck["action_cards"].([]any)
			data["deck"] = map[string]any{"characters": resources.characters(characters), "actions": resources.actions(actions)}
			return app.Image{Template: "tcg-decks", Data: data, Resources: resources.List}, true
		}
		return app.Image{}, false
	}
	list := []any{}
	for _, raw := range decks {
		deck, _ := raw.(map[string]any)
		characters, _ := deck["avatar_cards"].([]any)
		id := app.Text(deck["id"])
		list = append(list, map[string]any{"id": id, "command": context.Game.Prefix + "七圣查询卡组" + id, "characters": resources.characters(characters)})
	}
	data["decks"] = list
	return app.Image{Template: "tcg-decks", Data: data, Resources: resources.List}, true
}

// TCGCards draws 七圣召唤查询牌 the way Yunzai's deckCard page does: the TCG
// level and the cards collected of each kind, then every character card owned
// with its wins, uses and HP and every action card owned with its uses,
// copies and dice costs. 角色 or 行动 in the command keeps one kind, as
// upstream queries only that list.
func TCGCards(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	cards, ok := result.Data["card_list"].([]any)
	if !ok {
		return app.Image{}, false
	}
	resources := newTCGResources(context)
	resources.Artwork("deck-tcg", "yunzai-genshin", "resources/img/deck/七圣召唤.png")
	data := map[string]any{"uid": result.Role.UID, "nickname": result.Role.Nickname, "show_characters": !strings.Contains(context.Word, "行动"), "show_actions": !strings.Contains(context.Word, "角色")}
	if context.Query != nil {
		if basic, err := context.Query("genshin.tcg", nil); err == nil {
			info := basic.Data
			data["nickname"], data["level"] = app.Text(info["nickname"]), app.Int(info["level"])
			data["avatar_gained"], data["avatar_total"] = app.Int(info["avatar_card_num_gained"]), app.Int(info["avatar_card_num_total"])
			data["action_gained"], data["action_total"] = app.Int(info["action_card_num_gained"]), app.Int(info["action_card_num_total"])
		}
	}
	// One list holds both kinds; character cards are the ones with HP.
	owned := []map[string]any{}
	urls := []any{}
	for _, raw := range cards {
		card, _ := raw.(map[string]any)
		if app.Int(card["num"]) > 0 {
			owned = append(owned, card)
			urls = append(urls, card["image"])
		}
	}
	resources.Prefetch("mihoyo", urls...)
	characters, actions := []any{}, []any{}
	for _, card := range owned {
		image := resources.URL("mihoyo", card["image"])
		if app.Int(card["hp"]) > 0 {
			if data["show_characters"] == true {
				characters = append(characters, map[string]any{"image": image, "hp": resources.deck(app.Text(card["hp"])),
					"wins": app.Int(card["proficiency"]), "uses": app.Int(card["use_count"])})
			}
			continue
		}
		if data["show_actions"] != true {
			continue
		}
		costs := []any{}
		list, _ := card["action_cost"].([]any)
		for _, raw := range list {
			cost, _ := raw.(map[string]any)
			costs = append(costs, map[string]any{"type": resources.deck(app.Text(cost["cost_type"])), "value": resources.deck(app.Text(cost["cost_value"]))})
		}
		actions = append(actions, map[string]any{"image": image, "uses": app.Int(card["use_count"]), "num": app.Int(card["num"]), "costs": costs})
	}
	data["characters"], data["actions"] = characters, actions
	return app.Image{Template: "tcg-cards", Data: data, Resources: resources.List}, true
}
