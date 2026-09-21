package images

import (
	"math"
	"strconv"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// ledgerColors are Yunzai's colours for the primogem sources, by action ID.
var ledgerColors = []string{"#73a9c6", "#d56565", "#70b2b4", "#bd9a5a", "#739970", "#7a6da7", "#597ea0", "#ffb6c1"}

// ledgerColor is the colour of a primogem source by its action ID.
func ledgerColor(id int) string {
	return ledgerColors[((id%len(ledgerColors))+len(ledgerColors))%len(ledgerColors)]
}

// ledgerArtwork maps the images the ledger page names to Yunzai's paths.
var ledgerArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"img-other-bg", "resources/img/other/bg.webp"},
	{"img-other-chart", "resources/img/other/chart.png"},
	{"genshin-logo", "resources/img/other/原神.png"},
}

// Ledger draws 札记 the way Yunzai's html/ledger/ledger-gs does: the month,
// this and last month's primogems (with the pulls they buy) and mora, and the
// primogem sources as a ring chart with a legend. Upstream draws the ring with
// G2Plot; this draws the same ring as SVG.
func Ledger(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	month, _ := result.Data["month_data"].(map[string]any)
	if month == nil {
		return gamekit.Image{}, false
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range ledgerArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	// Upstream writes large numbers in ten thousands.
	amount := func(value any, digits int) string {
		number := gamekit.Int(value)
		if number > 10000 {
			return strconv.FormatFloat(float64(number)/10000, 'f', digits, 64) + " w"
		}
		return strconv.Itoa(number)
	}
	pulls := func(value any) string { return strconv.Itoa(int(math.Round(float64(gamekit.Int(value)) / 160))) }

	groups, _ := month["group_by"].([]any)
	total := 0
	for _, raw := range groups {
		group, _ := raw.(map[string]any)
		total += gamekit.Int(group["num"])
	}
	legend, values, colors := []any{}, []float64{}, []string{}
	for _, raw := range groups {
		group, _ := raw.(map[string]any)
		color := ledgerColor(gamekit.Int(group["action_id"]))
		legend = append(legend, map[string]any{"color": color, "action": gamekit.Text(group["action"]), "percent": gamekit.Text(group["percent"]), "num": gamekit.Int(group["num"])})
		values, colors = append(values, float64(gamekit.Int(group["num"]))), append(colors, color)
	}
	// G2Plot starts at the top and runs clockwise; the ring spans radius 0.7
	// to 1 of the 240px chart less its 10px padding.
	slices := gamekit.G2Ring(values, colors, 120, 120, 110, 77)

	day := gamekit.Text(result.Data["data_month"]) + "月"
	if now := context.Now.In(chinaTime); gamekit.Int(result.Data["data_month"]) == int(now.Month()) {
		day += strconv.Itoa(now.Day()) + "号"
	}
	return gamekit.Image{Template: "ledger", Data: map[string]any{
		"uid": result.Role.UID, "day": day,
		"current_primogems": amount(month["current_primogems"], 2), "gacha": pulls(month["current_primogems"]), "current_mora": amount(month["current_mora"], 1),
		"last_primogems": amount(month["last_primogems"], 2), "last_gacha": pulls(month["last_primogems"]), "last_mora": amount(month["last_mora"], 1),
		"legend": legend, "slices": slices, "total": strconv.Itoa(total),
	}, Resources: resources.List}, true
}
