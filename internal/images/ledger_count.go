package images

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// ledgerCountArtwork maps the images the ledger-count page names to Yunzai's
// paths.
var ledgerCountArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"genshin-logo", "resources/img/other/原神.png"},
}

var ledgerYearWord = regexp.MustCompile(`(\d{4})年`)

// ledgerMonth is a saved month's primogems, mora and sources.
type ledgerMonth struct {
	month, primogems, mora int
	groups                 []any
}

// LedgerCount draws 原石统计 the way Yunzai's html/ledger/ledger-count-gs
// does: the saved months of the year the command names (去年, 今年 or 2025年),
// or else those among the last twelve; the total primogems, the pulls they
// buy, the best month and its primogems, the total mora, the best mora month
// and the two largest sources; then each month's primogems as a column chart
// and every source as a ring with its legend. Upstream draws both charts with
// G2Plot; this draws them as SVG. Upstream picks the best mora month by
// comparing thousands as text; this compares the amounts.
func LedgerCount(context app.ImageContext, stats app.MonthlyStats) (app.Image, bool) {
	now := context.Now.In(chinaTime)
	year := 0
	switch {
	case strings.Contains(stats.Word, "去年"):
		year = now.Year() - 1
	case strings.Contains(stats.Word, "今年"):
		year = now.Year()
	default:
		if match := ledgerYearWord.FindStringSubmatch(stats.Word); match != nil {
			year, _ = strconv.Atoi(match[1])
		}
	}
	saved := map[string]map[string]any{}
	for _, item := range stats.Months {
		saved[item.Month] = item.Data
	}
	// A year reads January onwards; the last twelve months read newest first.
	keys := []string{}
	for index := range 12 {
		at := time.Date(now.Year(), now.Month()-time.Month(index), 1, 0, 0, 0, 0, chinaTime)
		if year != 0 {
			at = time.Date(year, time.Month(index+1), 1, 0, 0, 0, 0, chinaTime)
		}
		keys = append(keys, at.Format("2006-01"))
	}
	months := []ledgerMonth{}
	for _, key := range keys {
		data, _ := saved[key]["month_data"].(map[string]any)
		if data == nil {
			continue
		}
		groups, _ := data["group_by"].([]any)
		month, _ := strconv.Atoi(key[5:])
		months = append(months, ledgerMonth{month: month, primogems: app.Int(data["current_primogems"]), mora: app.Int(data["current_mora"]), groups: groups})
	}
	if len(months) == 0 {
		return app.Image{}, false
	}
	total, mora := 0, 0
	best, bestMora := months[0], months[0]
	sources := []*ledgerSource{}
	byAction := map[string]*ledgerSource{}
	for _, month := range months {
		total, mora = total+month.primogems, mora+month.mora
		if month.primogems > best.primogems {
			best = month
		}
		if month.mora > bestMora.mora {
			bestMora = month
		}
		for _, raw := range month.groups {
			group, _ := raw.(map[string]any)
			action := app.Text(group["action"])
			if byAction[action] == nil {
				byAction[action] = &ledgerSource{action: action, color: ledgerColor(app.Int(group["action_id"]))}
				sources = append(sources, byAction[action])
			}
			byAction[action].num += app.Int(group["num"])
		}
	}
	return ledgerCountImage(context, stats, year, months, sources, map[string]any{
		"total": fmt.Sprintf("%d.%02dw", (total+50)/100/100, (total+50)/100%100), "gacha": strconv.Itoa((total + 80) / 160),
		"mora": strconv.Itoa((mora+5000)/10000) + "w", "best_month": strconv.Itoa(best.month), "best_value": strconv.Itoa(best.primogems),
		"best_mora_month": strconv.Itoa(bestMora.month),
	}), true
}

// ledgerSource is one primogem source summed over the months.
type ledgerSource struct {
	action, color string
	num           int
}

// ledgerCountImage lays out the charts: the months oldest first as columns,
// the sources largest first, on first-seen order for ties, as the ring.
func ledgerCountImage(context app.ImageContext, stats app.MonthlyStats, year int, months []ledgerMonth, sources []*ledgerSource, data map[string]any) app.Image {
	resources := &app.ImageResources{Context: context}
	for _, item := range ledgerCountArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	// genshinLayout's header shows 闲云's banner from miao.
	resources.Artwork("head-banner", "miao-plugin", "resources/meta-gs/character/闲云/imgs/banner.webp")
	labels, values := []string{}, []float64{}
	for index := range months {
		// The last twelve months arrive newest first.
		month := months[index]
		if year == 0 {
			month = months[len(months)-1-index]
		}
		labels, values = append(labels, strconv.Itoa(month.month)+"月"), append(values, float64(month.primogems))
	}
	slices.SortStableFunc(sources, func(a, b *ledgerSource) int { return b.num - a.num })
	amounts, colors, legend, top, sum := []float64{}, []string{}, []any{}, []any{}, 0
	for index, source := range sources {
		amounts, colors = append(amounts, float64(source.num)), append(colors, source.color)
		legend = append(legend, map[string]any{"color": source.color, "action": source.action, "num": source.num,
			"y": fmt.Sprintf("%.1f", 150+(float64(index)-float64(len(sources)-1)/2)*26)})
		if index < 2 {
			top = append(top, map[string]any{"action": source.action, "num": source.num})
		}
		sum += source.num
	}
	yearText := ""
	if year != 0 {
		yearText = strconv.Itoa(year) + "年-"
	}
	data["uid"], data["year_text"], data["top"], data["legend"], data["sum"] = stats.Role.UID, yearText, top, legend, strconv.Itoa(sum)
	data["columns"] = app.G2Column(labels, values, 470, 300, [4]float64{40, 10, 30, 52})
	// Upstream labels a share of 2% or more.
	data["ring"] = app.G2Ring(amounts, colors, 160, 150, 140, 98, 2)
	// Upstream notes it shows only twelve months once that many are saved.
	data["more"] = year == 0 && len(stats.Months) >= 12
	return app.Image{Template: "ledger-count", Data: data, Resources: resources.List}
}
