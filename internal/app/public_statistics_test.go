package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPublicStatisticsPreserveZeroAndSeparateDenominators(t *testing.T) {
	a := App{Game: Game{ID: "genshin"}, Content: PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Fatal("public stats carried credentials")
		}
		data := map[string]any{"code": 200, "result": []any{map[string]any{"role": "测试角色", "c0": 0, "c6": 100, "role_sum": 10}}}
		if r.URL.Host == "api.yshelper.com" {
			data = map[string]any{"code": 200, "result": []any{}, "has_list": []any{map[string]any{"name": "测试角色", "own_rate": 0}}}
		}
		raw, _ := json.Marshal(data)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	out, err := a.publicStatistics(t.Context(), ContentQuery{Source: "ownership"})
	if err != nil {
		t.Fatal(err)
	}
	row := out["rows"].([]map[string]any)[0]
	if *row["holding_rate"].(*float64) != 0 || row["constellations"].([]*float64)[1] != nil || *row["constellations"].([]*float64)[0] != 0 {
		t.Fatal("missing values changed to zero")
	}
}
func TestAbyssTeamPairsFollowMiao(t *testing.T) {
	catalog := Catalog{Entries: []Entry{{ID: "1", Name: "甲", Kind: "character"}, {ID: "2", Name: "乙", Kind: "character"}, {ID: "3", Name: "丙", Kind: "character"}, {ID: "4", Name: "丁", Kind: "character"}}}
	data := map[string]any{
		"has_list": []any{map[string]any{"avatar": "a", "name": "甲"}, map[string]any{"avatar": "b", "name": "乙"}, map[string]any{"avatar": "c", "name": "丙"}, map[string]any{"avatar": "d", "name": "丁"}},
		"result": []any{[]any{map[string]any{"rank_name": "S"}}, []any{
			map[string]any{"role": []any{map[string]any{"avatar": "b"}, map[string]any{"avatar": "a"}}, "up_use_num": 10, "down_use_num": 0},
			map[string]any{"role": []any{map[string]any{"avatar": "a"}, map[string]any{"avatar": "c"}}, "up_use_num": 0, "down_use_num": 50},
			map[string]any{"role": []any{map[string]any{"avatar": "c"}, map[string]any{"avatar": "d"}}, "up_use_num": 0, "down_use_num": 5},
		}},
	}
	// 丁 is missing, so its team counts only its uses.
	pairs, missing := abyssTeamPairs(data, catalog, map[string]float64{"1": 1000, "2": 1000, "3": 1000})
	if !missing["4"] || len(missing) != 1 {
		t.Fatalf("missing = %v", missing)
	}
	// 甲乙 cannot pair with 甲丙, which shares 甲, so it pairs with 丙丁.
	if len(pairs) != 1 || pairs[0].Up.IDs[0] != "1" || pairs[0].Up.IDs[1] != "2" || pairs[0].Down.IDs[0] != "3" || pairs[0].Down.Owned || pairs[0].Mark != 15 || pairs[0].Count != 5 {
		t.Fatalf("pairs = %+v", pairs)
	}
}
