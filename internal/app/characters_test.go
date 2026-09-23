package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

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

// publicPool answers the accounts plugin's public query with an index and a
// character list, failing the operation named. As service calls, it decodes
// numbers as json.Number.
type publicPool struct {
	asked  []string
	failed string
}

func (c *publicPool) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	operation := asText(req.Params["operation"])
	c.asked = append(c.asked, req.Method+" "+operation+" "+asText(req.Params["region"]))
	if req.Method != "public.execute" || operation == c.failed {
		return &rayleabot.ActionError{Code: "plugin.account_public_unavailable"}
	}
	data := `{"data":{"avatars":[{"id":10000046,"level":80},{"id":10000025,"level":1}]}}`
	if operation == "characters" {
		data = `{"data":{"list":[{"id":10000046,"level":90,"weapon":{"id":13501,"name":"护摩之杖"}}]}}`
	}
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(out)
}

// A UID not on the requester's account is read as miao reads it with the
// public cookie pool: the index, then the character list whose entries stand
// in for the index's, with the kept panels. A failed list read leaves the
// index and the panels, and is the failure answered.
func TestReadPlayerThroughPublicPool(t *testing.T) {
	pool := &publicPool{}
	client := AccountsClient{Caller: pool, Game: "genshin"}
	a := App{Game: Game{ID: "genshin"}}
	saved := SavedProfiles{Panels: map[string]SavedPanel{"10000089": {Panel: CharacterPanel{ID: "10000089", Level: 90}}}}
	levels := func(image CharactersImage) map[string]int {
		out := map[string]int{}
		for _, raw := range image.Characters {
			avatar := asObject(raw)
			out[asText(avatar["id"])] = Int(avatar["level"])
		}
		return out
	}
	image, failed := a.readPlayer(t.Context(), client, panelOwner{UID: "500000001"}, saved)
	if failed != nil || !image.Public || strings.Join(pool.asked, ",") != "public.execute profile cn_qd01,public.execute characters cn_qd01" {
		t.Fatalf("asked = %v, failed %v", pool.asked, failed)
	}
	if got := levels(image); len(got) != 2 || got["10000046"] != 90 || got["10000089"] != 90 || asText(asObject(asObject(image.Characters[0])["weapon"])["name"]) != "护摩之杖" {
		t.Errorf("characters = %v", image.Characters)
	}
	pool.asked, pool.failed = nil, "characters"
	image, failed = a.readPlayer(t.Context(), client, panelOwner{UID: "100000001"}, saved)
	if PublicError(failed).Code != "plugin.account_public_unavailable" || len(pool.asked) != 2 || image.Index == nil {
		t.Fatalf("asked = %v, failed %v", pool.asked, failed)
	}
	if got := levels(image); len(got) != 2 || got["10000046"] != 80 || got["10000089"] != 90 {
		t.Errorf("characters = %v", image.Characters)
	}
}
