package images

import (
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/reference"
)

func TestTheaterRosterFollowsMiao(t *testing.T) {
	cards := &avatarCards{resources: &app.ImageResources{}, records: map[string]reference.Character{
		"1": {ID: "1", Name: "甲", Element: "pyro"}, "2": {ID: "2", Name: "乙", Element: "hydro"},
		"3": {ID: "3", Name: "丙", Element: "anemo"}, "4": {ID: "4", Name: "丁", Element: "pyro"},
	}}
	card := func(id string) map[string]any { return map[string]any{"elem": cards.records[id].Element} }
	entries := []rosterEntry{
		{id: "1", card: card("1"), level: 80},
		{id: "2", card: card("2"), level: 90},
		{id: "3", card: card("3"), level: 90},
		{id: "10000117", card: map[string]any{"elem": "pyro"}, level: 90},
	}
	roster := theaterRoster(entries, cards, map[string]any{"initial": []string{"1", "4"}, "invite": []string{"3"}, "elements": []string{"pyro"}})
	ids := []string{}
	for _, entry := range roster {
		ids = append(ids, entry.id)
	}
	// 甲 is replaced by its level-90 opening copy and 丁 joins, the two tied
	// in reverse order as miao's; 乙's element is not allowed, 丙 is invited,
	// and the Manekin is left out.
	if len(ids) != 3 || ids[0] != "4" || ids[1] != "1" || ids[2] != "3" || roster[1].level != 90 || roster[1].aeq != 24 {
		t.Fatalf("roster = %v", ids)
	}
}
