package assets

import (
	"context"
	"encoding/json"
	"testing"
)

// TestScoreDetailFollowsMiaoFormatting checks the panel detail against miao's
// getMarkDetail rules: rolls are the official upgrade count plus one,
// efficiency is the value in maximum rolls divided by 0.85, and values use
// the attribute's display format.
func TestScoreDetailFollowsMiaoFormatting(t *testing.T) {
	engine := calcEngine(t)
	for _, record := range engine.Metadata().Characters {
		if record.Key != "gs_10000046" {
			continue
		}
		input := map[string]any{
			"rank": 1, "weapon": map[string]any{"refinement": 1},
			"attributes": map[string]any{"atk": 1200, "cdmg": 220, "cpct": 70, "def": 900, "hp": 35000, "mastery": 180, "recharge": 110},
			"equipment": []any{map[string]any{"slot": 1, "set_name": "炽烈的炎之魔女", "main": map[string]any{"key": "hpPlus", "value": 4780}, "sub": []any{
				map[string]any{"key": "cpct", "value": 10.5, "times": 2}, map[string]any{"key": "cdmg", "value": 21, "times": 2},
				map[string]any{"key": "atk", "value": 5.8}, map[string]any{"key": "mastery", "value": 23},
			}}},
		}
		raw, err := engine.Score(context.Background(), record, input)
		if err != nil {
			t.Fatal(err)
		}
		type attr struct {
			Key, Value, Mark, Eff string
			UpNum                 int
		}
		var detail struct {
			Mark     string `json:"mark"`
			AllAttrs []attr `json:"all_attrs"`
			Pieces   []struct {
				Main  attr   `json:"main"`
				Attrs []attr `json:"attrs"`
			} `json:"pieces"`
			Titles map[string]string `json:"titles"`
		}
		if err := json.Unmarshal(raw, &detail); err != nil || len(detail.Pieces) != 1 {
			t.Fatalf("detail = %s (%v)", raw, err)
		}
		piece := detail.Pieces[0]
		if piece.Main.Value != "4,780.0" || piece.Main.Eff != "-" {
			t.Errorf("main = %+v", piece.Main)
		}
		if got := piece.Attrs[0]; got != (attr{Key: "cpct", Value: "10.5%", Mark: "270", Eff: "3.2", UpNum: 3}) {
			t.Errorf("crit rate = %+v", got)
		}
		if got := piece.Attrs[3]; got != (attr{Key: "mastery", Value: "23.0", Mark: "74", Eff: "1.2", UpNum: 1}) {
			t.Errorf("mastery = %+v", got)
		}
		if detail.Mark != "51.3" || len(detail.AllAttrs) != 4 || detail.AllAttrs[0].Eff != "3.2" || detail.Titles["cpct"] != "暴击率" {
			t.Errorf("detail = %+v", detail)
		}
		return
	}
	t.Fatal("胡桃 left the catalog")
}
