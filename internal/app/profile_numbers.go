package app

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/RayleaBot/genshin/internal/reference"
)

// ProfileNumbers holds the formatted static attributes used by the character card.
type ProfileNumbers struct {
	Stats  map[string]PanelStat `json:"stats"`
	Weapon []PanelStat          `json:"weapon"`
}

func (g Game) profileNumbers(ctx context.Context, panel CharacterPanel, record reference.Character) ProfileNumbers {
	if g.Calc == nil || record.ID == "" || panel.Level < 1 || (panel.Weapon == nil && !panel.WeaponKnown) {
		return ProfileNumbers{}
	}
	weapon := BuildWeapon{Refinement: 1}
	if w := panel.Weapon; w != nil {
		weapon = BuildWeapon{ID: w.ID, Level: w.Level, Promote: w.Promote, Refinement: w.Refinement}
	}
	if g.Data != nil && g.Data.CloudGear != nil && (panel.Source == "mihoyo" || panel.Source == "enka") {
		pieces := slices.Clone(panel.Equipment)
		for i, piece := range pieces {
			if !piece.Complete {
				continue
			}
			if raw, ok := panelCloudGear(g, piece, panel.Source); ok {
				if _, exact := decodeCloudGear(g, piece.Slot, raw); exact != nil && len(exact.Sub) == len(piece.Sub) {
					pieces[i] = *exact
				}
			}
		}
		panel.Equipment = pieces
	}
	gear, err := buildGearSet(panel)
	if err != nil {
		gear = []BuildGear{}
	}
	input := map[string]any{"level": panel.Level, "promote": panel.Promote, "rank": panel.Rank, "weapon": weapon,
		"equipment": gear, "observed": observedAttributes(panel), "complete": err == nil && panel.RankKnown}
	raw, err := g.Calc.Change(ctx, record, input)
	var result ProfileNumbers
	if err != nil || json.Unmarshal(raw, &result) != nil {
		return ProfileNumbers{}
	}
	return result
}
