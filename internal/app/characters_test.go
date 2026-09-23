package app

import "testing"

// As miao's player data: the list stands in for the index's entry, a kept
// panel adds its character, and without a level above 1 or a weapon a
// character is left out.
func TestPlayerCharactersMergeAsMiao(t *testing.T) {
	index := decoded(t, `{"avatars":[{"id":10000046,"level":80},{"id":10000025,"level":1},{"id":10000032,"level":1}]}`)
	list := decoded(t, `{"list":[{"id":10000046,"level":90},{"id":10000021,"level":1,"weapon":{"name":"猎弓"}}]}`)["list"].([]any)
	kept := map[string]SavedPanel{"10000032": {Panel: CharacterPanel{ID: "10000032", Level: 1, Weapon: &PanelEquipment{Name: "无锋剑"}}}, "10000089": {Panel: CharacterPanel{ID: "10000089", Level: 90, Rank: 2}}}
	levels := map[string]int{}
	for _, raw := range playerCharacters(index, list, kept) {
		avatar := asObject(raw)
		levels[asText(avatar["id"])] = Int(avatar["level"])
	}
	want := map[string]int{"10000046": 90, "10000032": 1, "10000021": 1, "10000089": 90}
	if len(levels) != len(want) {
		t.Fatalf("characters = %v", levels)
	}
	for id, level := range want {
		if levels[id] != level {
			t.Errorf("characters = %v", levels)
		}
	}
}
