package app

import (
	"testing"
	"time"
)

func TestRoleCardMatchingRequiresMutualNeedSameRegionAndFreshOptIn(t *testing.T) {
	now := time.Now().UnixMilli()
	card := func(name string, count int, unlocked bool) RoleCard {
		return RoleCard{Name: name, Count: count, Unlocked: unlocked, Icon: "https://example.com/" + name + ".png"}
	}
	self := RoleCardOffer{ActorID: "self", UID: "1", Region: "cn_gf01", Cards: []RoleCard{card("A", 2, true), card("C", 0, false), card("B", 0, false)}}
	other := RoleCardOffer{ActorID: "other", UID: "2", Region: "cn_gf01", Cards: []RoleCard{card("A", 0, false), card("B", 3, true), card("C", 2, true)}, SavedMS: now}
	rows := []RoleCardOffer{other}
	foreign := other
	foreign.Region = "os_asia"
	rows = append(rows, foreign)
	stale := other
	stale.SavedMS = now - int64(31*24*time.Hour/time.Millisecond)
	rows = append(rows, stale)
	oneway := other
	oneway.ActorID = "oneway"
	oneway.Cards = []RoleCard{card("A", 1, true), card("B", 3, true)}
	rows = append(rows, oneway)
	matches := roleCardMatches(self, rows, now)
	if len(matches) != 1 || matches[0].Score != 3 || len(matches[0].Give) != 1 || matches[0].Give[0].Name != "A" || matches[0].Give[0].Count != 2 {
		t.Fatal(matches)
	}
	// What the member gives follows the requester's cards, as miao picks
	// them by the requester's needs, and carries the member's count.
	if receive := matches[0].Receive; len(receive) != 2 || receive[0].Name != "C" || receive[1].Name != "B" || receive[1].Count != 3 {
		t.Fatal(receive)
	}
	if _, err := parseRoleCards(map[string]any{"tarot_card_state": map[string]any{"list": []any{map[string]any{"name": "A", "is_unlock": true}}}}); err == nil {
		t.Fatal("missing count inferred")
	}
	if _, err := parseRoleCards(map[string]any{}); PublicError(err).Message != "暂未获得「月谕圣牌」收藏数据..." {
		t.Fatal(err)
	}
}
