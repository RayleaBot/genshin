package images

import (
	"strconv"
	"unicode"

	"github.com/RayleaBot/genshin/internal/app"
)

// payLogArtwork maps the images the payLog page names to Yunzai's paths.
var payLogArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"genshin-logo", "resources/img/other/原神.png"},
}

// payLogPrices are the yuan of each pack in PayMonth's order.
var payLogPrices = [8]float64{68, 30, 648, 328, 198, 98, 30, 6}

// PayLog draws 充值记录 the way Yunzai's html/payLog does: the yuan spent,
// the crystals bought, the month spent most and its yuan, and how many of
// each pack, four to a row; each month's yuan as a bar chart and the yuan by
// pack as a pie. Upstream draws both charts with ECharts; this draws them as
// SVG.
func PayLog(context app.ImageContext, log app.PayLog) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range payLogArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	labels, sales := []string{}, []float64{}
	best, total := -1, 0.0
	counts := [8]int{}
	for index, month := range log.Months {
		spent := 0.0
		for pack, count := range month.Counts {
			spent += float64(count) * payLogPrices[pack]
			counts[pack] += count
		}
		labels, sales = append(labels, month.Month), append(sales, spent)
		total += spent
		// Upstream sorts by yuan with a comparator that puts the later of two
		// equal months first.
		if best < 0 || spent >= sales[best] {
			best = index
		}
	}
	yuan := func(value float64) string { return "￥" + strconv.FormatFloat(value, 'f', -1, 64) }
	top := []any{
		map[string]any{"title": "总消费", "value": yuan(total)},
		map[string]any{"title": "总结晶", "value": log.Crystal},
	}
	if best >= 0 {
		top = append(top, map[string]any{"title": "消费最多", "value": labels[best]}, map[string]any{"title": labels[best] + "消费", "value": yuan(sales[best])})
	}
	pieNames, pieValues := []string{}, []float64{}
	for pack, count := range counts {
		top = append(top, map[string]any{"title": app.PayPackNames[pack], "value": count})
		if value := float64(count) * payLogPrices[pack]; value > 0 {
			pieNames, pieValues = append(pieNames, app.PayPackNames[pack]), append(pieValues, value)
		}
	}
	lines := []any{}
	for start := 0; start < len(top); start += 4 {
		lines = append(lines, top[start:min(start+4, len(top))])
	}
	pie := app.EChartsPie(pieNames, pieValues, 235, 150, 75, 300)
	// The pie's legend sits at the bottom, centred as a box: 25 by 14
	// icons, 5 to the text, 10 between items and lines, wrapping at the
	// chart's width less its padding.
	rows, widths := [][]int{{}}, []float64{0}
	for index, name := range pieNames {
		width := 30 + payLogTextWidth(name)
		last := len(rows) - 1
		if len(rows[last]) > 0 && widths[last]+10+width > 460 {
			rows, widths = append(rows, []int{}), append(widths, 0)
			last++
		}
		if len(rows[last]) > 0 {
			widths[last] += 10
		}
		rows[last], widths[last] = append(rows[last], index), widths[last]+width
	}
	boxWidth := 0.0
	for _, width := range widths {
		boxWidth = max(boxWidth, width)
	}
	legend := []any{}
	for line, row := range rows {
		x, y := (470-boxWidth)/2, 281-float64(len(rows)-1-line)*24
		for _, index := range row {
			legend = append(legend, map[string]any{"name": pieNames[index], "color": pie[index].(map[string]any)["color"], "x": x, "y": y, "text_x": x + 30, "text_y": y + 7})
			x += 30 + payLogTextWidth(pieNames[index]) + 10
		}
	}
	return app.Image{Template: "pay-log", Data: map[string]any{
		"uid": log.UID, "lines": lines, "total": yuan(total),
		"bar":    app.EChartsBar(labels, sales, 470, 300, [4]float64{60, 47, 60, 47}, 5, payLogTextWidth),
		"pie":    pie,
		"legend": legend,
	}, Resources: resources.List}, true
}

// payLogTextWidth is the width of a 12px label: digits about 7, a Chinese
// character 12.
func payLogTextWidth(text string) float64 {
	width := 0.0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			width += 12
		} else {
			width += 6.7
		}
	}
	return width
}
