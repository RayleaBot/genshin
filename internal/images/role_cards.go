package images

import (
	"slices"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// RoleCards draws 月谕圣牌 the way miao's stat/role-card does: the cards
// collected of the total, then every card in the official order with its
// picture once unlocked and the number held; 月谕圣牌交换 is
// stat/role-card-exchange: each member to trade with, most kinds first, with
// the cards they can give and those they lack that the requester can spare.
func RoleCards(context app.ImageContext, page app.RoleCardsImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range statArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	urls := []any{}
	for _, card := range page.Cards {
		urls = append(urls, card.Icon)
	}
	for _, match := range page.Matches {
		for _, card := range slices.Concat(match.Receive, match.Give) {
			urls = append(urls, card.Icon)
		}
	}
	resources.Prefetch("mihoyo", urls...)
	if page.Exchange {
		// A card offered shows how many the giver can spare.
		trade := func(cards []app.RoleCard) []any {
			list := []any{}
			for _, card := range cards {
				list = append(list, map[string]any{"name": card.Name, "icon": resources.URL("mihoyo", card.Icon), "num": card.Count - 1})
			}
			return list
		}
		matches := []any{}
		for index, match := range page.Matches {
			name := []rune(match.Nickname)
			// miao keeps the sender's name to twelve characters, as lodash's
			// truncate.
			if len(name) > 12 {
				name = append(name[:9], []rune("...")...)
			}
			if len(name) == 0 {
				name = []rune(match.ActorID)
			}
			matches = append(matches, map[string]any{"rank": index + 1, "name": string(name), "qq": match.ActorID, "uid": match.UID, "score": match.Score,
				"receive": trade(match.Receive), "give": trade(match.Give)})
		}
		return app.Image{Template: "stat-role-card-exchange", Data: map[string]any{
			"uid": page.UID, "group": page.Group, "matches": matches,
		}, Resources: resources.List}, true
	}
	cards := []any{}
	for _, card := range page.Cards {
		entry := map[string]any{"name": card.Name, "count": card.Count}
		if card.Unlocked {
			entry["unlocked"], entry["icon"] = true, resources.URL("mihoyo", card.Icon)
		}
		cards = append(cards, entry)
	}
	return app.Image{Template: "stat-role-card", Data: map[string]any{
		"uid": page.UID, "collected": page.Collected, "total": page.Total, "cards": cards,
	}, Resources: resources.List}, true
}
