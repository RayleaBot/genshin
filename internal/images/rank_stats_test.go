package images_test

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestRankStatsFollowsArk(t *testing.T) {
	stats := gamekit.CloudStats{Title: "重击伤害", Total: "100", Scores: []float64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}}
	image, ok := images.RankStats(gamekit.ImageContext{}, gamekit.RankStatsImage{Character: "胡桃", Stats: stats})
	if !ok || image.Template != "rank-stats" {
		t.Fatal(image)
	}
	cards := image.Data["cards"].([]any)
	first, fourth := cards[0].(map[string]any), cards[3].(map[string]any)
	if len(cards) != 10 || first["label"] != "1%" || first["top"] != true || fourth["top"] != false || fourth["value"] != "7" {
		t.Fatal(cards)
	}
	// Scores other than ten percentiles keep the text reply.
	if _, ok := images.RankStats(gamekit.ImageContext{}, gamekit.RankStatsImage{Stats: gamekit.CloudStats{Scores: []float64{1}}}); ok {
		t.Fatal("drew a partial distribution")
	}
}
