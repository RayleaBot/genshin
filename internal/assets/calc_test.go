package assets

import (
	"context"
	"encoding/json"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
	"math"
	"os"
	"testing"
)

type vector struct {
	Key     string         `json:"key"`
	Input   map[string]any `json:"input"`
	Results []struct {
		Expected *float64 `json:"expected"`
		Text     string   `json:"text"`
		Critical *float64 `json:"critical"`
	} `json:"results"`
}

func vectors(t *testing.T) []vector {
	t.Helper()
	raw, err := os.ReadFile("testdata/calc-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []vector
	if json.Unmarshal(raw, &cases) != nil {
		t.Fatal("invalid reference fixture")
	}
	return cases
}
func TestReferenceProfiles(t *testing.T) {
	metadata := calcEngine(t).Metadata()
	records := map[string]reference.Character{}
	for _, c := range metadata.Characters {
		records[c.Key] = c
	}
	for _, v := range vectors(t) {
		t.Run(v.Key+"/"+string(rune('0'+int(v.Input["rank"].(float64)))), func(t *testing.T) {
			result, err := calcEngine(t).Run(context.Background(), records[v.Key], v.Input)
			if err != nil {
				t.Fatal(err)
			}
			var output struct {
				Baseline struct {
					Results []struct {
						ID       string   `json:"id"`
						Expected *float64 `json:"expected"`
						Text     string   `json:"text"`
						Critical *float64 `json:"critical"`
						Default  bool     `json:"default"`
					} `json:"results"`
				} `json:"baseline"`
			}
			if json.Unmarshal(result, &output) != nil || len(output.Baseline.Results) != len(v.Results) {
				t.Fatal("result shape changed")
			}
			// Group ranks use one detail: 神里绫华 names the third with defDmgIdx.
			defaults := []string{}
			for _, result := range output.Baseline.Results {
				if result.Default {
					defaults = append(defaults, result.ID)
				}
			}
			if len(defaults) > 1 || records[v.Key].Name == "神里绫华" && (len(defaults) != 1 || defaults[0] != "2") {
				t.Fatalf("default details = %v", defaults)
			}
			equal := func(a, b float64) bool { return math.Abs(a-b) <= 1e-8*math.Max(1, math.Abs(b)) }
			for i, expected := range v.Results {
				actual := output.Baseline.Results[i]
				if actual.Text != expected.Text || (actual.Expected == nil) != (expected.Expected == nil) || actual.Expected != nil && !equal(*actual.Expected, *expected.Expected) || (actual.Critical == nil) != (expected.Critical == nil) || actual.Critical != nil && !equal(*actual.Critical, *expected.Critical) {
					t.Fatalf("result %d differs: got %+v want %+v", i, actual, expected)
				}
			}
		})
	}
}
func TestEveryWeaponAndRequestIsolation(t *testing.T) {
	metadata := calcEngine(t).Metadata()
	fixtures := vectors(t)
	records := map[string]reference.Character{}
	for _, c := range metadata.Characters {
		records[c.Key] = c
	}
	byType := map[string]vector{}
	for _, fixture := range fixtures {
		c := records[fixture.Key]
		key := c.Game + "/" + c.WeaponType
		if _, exists := byType[key]; !exists {
			byType[key] = fixture
		}
	}
	for _, weapon := range metadata.Weapons {
		if weapon.Game == "zzz" {
			continue
		}
		t.Run(weapon.Game+"/"+weapon.ID, func(t *testing.T) {
			fixture, ok := byType[weapon.Game+"/"+weapon.Type]
			if !ok {
				t.Fatal("missing representative character")
			}
			encoded, _ := json.Marshal(fixture.Input)
			var input map[string]any
			_ = json.Unmarshal(encoded, &input)
			level := 90
			if weapon.Game == "sr" {
				level = 80
			}
			input["candidate_weapon"] = map[string]any{"id": weapon.ID, "level": level, "promote": 6, "refinement": 5}
			result, err := calcEngine(t).Run(context.Background(), records[fixture.Key], input)
			if err != nil {
				t.Fatal(err)
			}
			var out struct {
				Candidate any `json:"candidate"`
			}
			if json.Unmarshal(result, &out) != nil || out.Candidate == nil {
				t.Fatal("candidate omitted")
			}
		})
	}
	fixture := fixtures[0]
	for i := 0; i < 4; i++ {
		t.Run("isolated", func(t *testing.T) {
			t.Parallel()
			for repeat := 0; repeat < 3; repeat++ {
				result, err := calcEngine(t).Run(context.Background(), records[fixture.Key], fixture.Input)
				if err != nil {
					t.Fatal(err)
				}
				var out struct {
					Candidate any `json:"candidate"`
				}
				_ = json.Unmarshal(result, &out)
				if out.Candidate != nil {
					t.Fatal("candidate leaked between executions")
				}
			}
		})
	}
}

// calcEngine runs this plugin's embedded scripts exactly as the plugin does.
func calcEngine(t *testing.T) *reference.Engine {
	t.Helper()
	engine, err := reference.New(Load().Calc)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}
