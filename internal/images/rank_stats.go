package images

import (
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// rankStatsArtwork maps the images named in the converted stylesheets (miao's
// common for the layout, ark-plugin's graph/stats) to their sources and
// paths.
var rankStatsArtwork = [][3]string{
	{"Number", "miao-plugin", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "miao-plugin", "resources/common/font/NZBZ.woff"},
	{"YS", "miao-plugin", "resources/common/font/HYWH-65W.woff"},
	{"graph-background", "ark-plugin", "resources/graph/background.png"},
}

// rankStatsPercents are the ranking percentiles of ark's scores.
var rankStatsPercents = []float64{1, 5, 10, 20, 30, 50, 70, 90, 95, 99}

// RankStats draws 排名统计 the way ark-plugin's graph/stats does: the
// character, the calculation and sample size, the ECharts line of the score
// at each ranking percentile with its value above each point and a guide at
// each percentile, and a card per percentile with the first three marked.
func RankStats(context app.ImageContext, image app.RankStatsImage) (app.Image, bool) {
	scores := image.Stats.Scores
	if len(scores) != len(rankStatsPercents) {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range rankStatsArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	// ark's chart: a 1120 x 520 box, grid 30/40/50/80, y up to 1.15 times the
	// top score in six splits, labels in 万 from 10000.
	chart := app.EChartsLine(rankStatsPercents, scores, 1120, 520, [4]float64{30, 40, 50, 80}, math.Ceil(slices.Max(scores)*1.15), 6, 0.4, func(value float64) string {
		if value >= 10000 {
			return strconv.FormatFloat(value/10000, 'f', 1, 64) + "万"
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	})
	cards := []any{}
	for index, point := range chart["points"].([]any) {
		label := strconv.FormatFloat(rankStatsPercents[index], 'f', -1, 64) + "%"
		item := point.(map[string]any)
		x, _ := strconv.ParseFloat(item["x"].(string), 64)
		y, _ := strconv.ParseFloat(item["y"].(string), 64)
		// Labels sit 12px above the 10px symbol; the first is padded 40px on
		// its left.
		item["mark"], item["label_x"], item["label_y"] = label, x, y-17
		if index == 0 {
			item["label_x"] = x + 20
		}
		cards = append(cards, map[string]any{"label": label, "value": item["value"], "top": index < 3})
	}
	bottom, _ := strconv.ParseFloat(chart["bottom"].(string), 64)
	left, _ := strconv.ParseFloat(chart["left"].(string), 64)
	right, _ := strconv.ParseFloat(chart["right"].(string), 64)
	top, _ := strconv.ParseFloat(chart["top"].(string), 64)
	chart["label_x"], chart["mark_y"] = left-8, bottom+5
	chart["x_name_x"], chart["x_name_y"] = (left+right)/2, bottom+32
	chart["y_name_x"], chart["y_name_y"] = left-20, top-15
	now := time.Now().In(time.FixedZone("UTC+8", 8*3600))
	return app.Image{Template: "rank-stats", Data: map[string]any{"character": image.Character, "title": image.Stats.Title, "total": image.Stats.Total,
		"time": now.Format("2006-01-02 15:04"), "chart": chart, "cards": cards}, Resources: resources.List}, true
}
