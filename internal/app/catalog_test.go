package app

import "testing"

func TestResolvePrefersTheExactName(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"1002","name":"丹恒","kind":"character"},
		{"id":"1213","name":"丹恒•饮月","aliases":["饮月"],"kind":"character"},
		{"id":"8006","name":"星·同谐","kind":"character"},
		{"id":"8010","name":"星·同谐","kind":"character"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]string{"丹恒": "1002", "饮月": "1213", "丹恒•": "1213", "8010": "8010", "龙尊": "1213"} {
		aliases := map[string]string{"龙尊": "1213"}
		entry, ok := catalog.Resolve(query, "character", aliases)
		if !ok || entry.ID != want {
			t.Fatalf("%s resolved to %q %v, want %s", query, entry.ID, ok, want)
		}
	}
	// Two entries share the name exactly; only the ID can pick one.
	if _, ok := catalog.Resolve("星·同谐", "character", nil); ok {
		t.Fatal("an ambiguous name resolved")
	}
}

func TestResolveReadsTravelersAsMiao(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"10000005","name":"空","aliases":["男主"],"kind":"character"},
		{"id":"10000007","name":"荧","aliases":["女主"],"kind":"character"},
		{"id":"20000000","name":"旅行者","aliases":["主角","主"],"kind":"character"},
		{"id":"10000052","name":"雷电将军","aliases":["雷神"],"kind":"character"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]string{"风主": "20000000", "雷荧": "20000000", "草主角": "20000000", "稻妻旅行者": "20000000", "雷女主": "20000000", "空": "10000005", "旅行者": "20000000", "雷电将军": "10000052"} {
		entry, ok := catalog.Resolve(query, "character", nil)
		if !ok || entry.ID != want {
			t.Errorf("%s resolved to %q %v, want %s", query, entry.ID, ok, want)
		}
	}
	// Only an element makes the rest a traveler.
	for _, query := range []string{"神主", "雷神主"} {
		if entry, ok := catalog.Resolve(query, "character", nil); ok {
			t.Errorf("%s resolved to %s", query, entry.ID)
		}
	}
}
