package app

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// ark-plugin's 排名统计 and Yunzai's payLog draw their charts with ECharts
// in the browser. Plugin templates do not run scripts, so these build
// the same charts as SVG shapes for the templates to place.

// EChartsDistribution lays out the ECharts line ark-plugin's profile-detail
// draws of a ranking distribution, in an 800 x 400 box with ECharts' default
// grid (80 left and right, 60 top and bottom): x runs over 0–100 with a tick
// every 20, y from 0 to yMax with a tick every interval and at yMax, the
// points join as smooth: true draws them and the area below them is
// filled. at places a data point in the box.
func EChartsDistribution(xs, values []float64, yMax, interval float64) (chart map[string]any, at func(x, y float64) (float64, float64)) {
	left, top, right, bottom := 80.0, 60.0, 720.0, 340.0
	at = func(x, y float64) (float64, float64) {
		if yMax <= 0 {
			return left + x/100*(right-left), bottom
		}
		return left + x/100*(right-left), bottom - y/yMax*(bottom-top)
	}
	points := make([][2]float64, len(values))
	for index, value := range values {
		points[index][0], points[index][1] = at(xs[index], value)
	}
	line := echartsSmooth(points, 0.5)
	area := ""
	if len(points) > 0 {
		area = fmt.Sprintf("%s L %s %s L %s %s Z", line, svgNumber(points[len(points)-1][0]), svgNumber(bottom), svgNumber(points[0][0]), svgNumber(bottom))
	}
	xTicks, yTicks := []any{}, []any{}
	for tick := 0.0; tick <= 100; tick += 20 {
		x, _ := at(tick, 0)
		xTicks = append(xTicks, map[string]any{"x": svgNumber(x), "value": tick})
	}
	for tick := 0.0; interval > 0 && tick <= yMax+interval*1e-9; tick += interval {
		_, y := at(0, tick)
		yTicks = append(yTicks, map[string]any{"y": svgNumber(y), "value": tick})
	}
	if last := len(yTicks) - 1; yMax > 0 && (last < 0 || yTicks[last].(map[string]any)["value"].(float64) < yMax) {
		yTicks = append(yTicks, map[string]any{"y": svgNumber(top), "value": yMax})
	}
	chart = map[string]any{"width": "800", "height": "400", "left": svgNumber(left), "right": svgNumber(right), "top": svgNumber(top), "bottom": svgNumber(bottom),
		"line": line, "area": area, "x_ticks": xTicks, "y_ticks": yTicks}
	return chart, at
}

// echartsNice is ECharts' numberUtil.nice with rounding.
func echartsNice(value float64) float64 {
	exponent := math.Floor(math.Log10(value))
	exp10 := math.Pow(10, exponent)
	f := value / exp10
	nf := 10.0
	switch {
	case f < 1.5:
		nf = 1
	case f < 2.5:
		nf = 2
	case f < 4:
		nf = 3
	case f < 7:
		nf = 5
	}
	return nf * exp10
}

// echartsSmooth is the path ECharts' poly shape draws through points with a
// smooth factor and no monotone direction: each control pair follows the
// neighbours' direction, split by the lengths of the two segments and kept
// within the neighbouring points.
func echartsSmooth(points [][2]float64, smooth float64) string {
	if len(points) == 0 {
		return ""
	}
	path := []string{"M " + svgNumber(points[0][0]) + " " + svgNumber(points[0][1])}
	cp0 := points[0]
	prev := points[0]
	for index := 1; index < len(points); index++ {
		point := points[index]
		if index == len(points)-1 {
			path = append(path, fmt.Sprintf("C %s %s %s %s %s %s", svgNumber(cp0[0]), svgNumber(cp0[1]), svgNumber(point[0]), svgNumber(point[1]), svgNumber(point[0]), svgNumber(point[1])))
			break
		}
		next := points[index+1]
		lenPrev := math.Hypot(point[0]-prev[0], point[1]-prev[1])
		lenNext := math.Hypot(next[0]-point[0], next[1]-point[1])
		ratio := lenNext / (lenNext + lenPrev)
		vx, vy := next[0]-prev[0], next[1]-prev[1]
		nextCp := [2]float64{point[0] + vx*smooth*ratio, point[1] + vy*smooth*ratio}
		nextCp[0] = math.Max(math.Min(nextCp[0], math.Max(next[0], point[0])), math.Min(next[0], point[0]))
		nextCp[1] = math.Max(math.Min(nextCp[1], math.Max(next[1], point[1])), math.Min(next[1], point[1]))
		vx, vy = nextCp[0]-point[0], nextCp[1]-point[1]
		cp1 := [2]float64{point[0] - vx*lenPrev/lenNext, point[1] - vy*lenPrev/lenNext}
		cp1[0] = math.Max(math.Min(cp1[0], math.Max(prev[0], point[0])), math.Min(prev[0], point[0]))
		cp1[1] = math.Max(math.Min(cp1[1], math.Max(prev[1], point[1])), math.Min(prev[1], point[1]))
		vx, vy = point[0]-cp1[0], point[1]-cp1[1]
		nextCp = [2]float64{point[0] + vx*lenNext/lenPrev, point[1] + vy*lenNext/lenPrev}
		path = append(path, fmt.Sprintf("C %s %s %s %s %s %s", svgNumber(cp0[0]), svgNumber(cp0[1]), svgNumber(cp1[0]), svgNumber(cp1[1]), svgNumber(point[0]), svgNumber(point[1])))
		cp0, prev = nextCp, point
	}
	return strings.Join(path, " ")
}

// EChartsBar lays out an ECharts bar chart of one series on a category axis
// in a width x height box with the grid's (top, right, bottom, left)
// margins: the value axis from 0 to the nice maximum ECharts picks for
// splitNumber, each bar 80% of its band, the axis ticks between the bands and
// the category labels ECharts' auto interval keeps, a label taking
// labelWidth, and each bar's label 5 above it.
func EChartsBar(labels []string, values []float64, width, height float64, grid [4]float64, splitNumber int, labelWidth func(string) float64) map[string]any {
	left, top := grid[3], grid[0]
	right, bottom := width-grid[1], height-grid[2]
	maximum := 0.0
	for _, value := range values {
		maximum = math.Max(maximum, value)
	}
	if maximum == 0 {
		// ECharts widens an empty extent to 0–1.
		maximum = 1
	}
	interval := echartsNice(maximum / float64(splitNumber))
	yMax := math.Ceil(maximum/interval-1e-9) * interval
	y := func(value float64) float64 { return bottom - value/yMax*(bottom-top) }
	ticks := []any{}
	for tick := 0.0; tick <= yMax+interval*1e-9; tick = math.Round((tick+interval)*1e9) / 1e9 {
		ticks = append(ticks, map[string]any{"y": svgNumber(y(tick)), "label": strconv.FormatFloat(tick, 'f', -1, 64)})
	}
	band := (right - left) / float64(max(len(values), 1))
	// calculateCategoryInterval: the widest label, grown by 1.3, over a band.
	widest := 0.0
	for _, label := range labels {
		widest = math.Max(widest, labelWidth(label)*1.3)
	}
	step := int(math.Floor(widest/band)) + 1
	bars, separators := []any{}, []any{map[string]any{"x": svgNumber(left)}}
	for index, value := range values {
		x := left + band*float64(index) + band*0.1
		separators = append(separators, map[string]any{"x": svgNumber(left + band*float64(index+1))})
		bar := map[string]any{"x": svgNumber(x), "y": svgNumber(y(value)), "width": svgNumber(band * 0.8), "height": svgNumber(bottom - y(value)), "center": svgNumber(x + band*0.4),
			"value": strconv.FormatFloat(value, 'f', -1, 64), "value_y": svgNumber(y(value) - 5)}
		if index%step == 0 {
			bar["label"] = labels[index]
		}
		bars = append(bars, bar)
	}
	return map[string]any{"width": svgNumber(width), "height": svgNumber(height), "left": svgNumber(left), "right": svgNumber(right), "top": svgNumber(top), "bottom": svgNumber(bottom),
		"bars": bars, "ticks": ticks, "separators": separators, "tick_bottom": svgNumber(bottom + 5), "label_y": svgNumber(bottom + 8), "axis_x": svgNumber(left - 8)}
}

// EChartsPie lays out an ECharts pie of radius r at (cx, cy) in a box of
// height tall, starting at the top and running clockwise, each item labelled
// outside: ECharts' two-part label line of 15 and 15 with the text 5 beyond
// it, the labels on a side moved apart vertically as avoidOverlap does, and
// the percents to two places summing to 100.
func EChartsPie(names []string, values []float64, cx, cy, r, height float64) []any {
	total := 0.0
	for _, value := range values {
		total += value
	}
	percents := echartsPercents(values, 2)
	type item struct {
		index                  int
		x1, y1, x2, y2, labelY float64
		start, end, side       float64
	}
	items := []*item{}
	angle := -math.Pi / 2
	for index, value := range values {
		span := 0.0
		if total > 0 {
			span = value / total * 2 * math.Pi
		}
		middle := angle + span/2
		dx, dy := math.Cos(middle), math.Sin(middle)
		side := 1.0
		if dx < 0 {
			side = -1
		}
		entry := &item{index: index, x1: cx + dx*r, y1: cy + dy*r, x2: cx + dx*(r+15), y2: cy + dy*(r+15), start: angle, end: angle + span, side: side}
		entry.labelY = entry.y2
		items = append(items, entry)
		angle += span
	}
	// avoidOverlap: on each side the labels, 12 high, are pushed apart and
	// kept inside the box, and a moved label's line bends where its height
	// meets the circle of radius r+15.
	for _, side := range []float64{-1, 1} {
		list := []*item{}
		for _, entry := range items {
			if entry.side == side {
				list = append(list, entry)
			}
		}
		sort.SliceStable(list, func(i, j int) bool { return list[i].labelY < list[j].labelY })
		for index := 1; index < len(list); index++ {
			list[index].labelY = math.Max(list[index].labelY, list[index-1].labelY+12)
		}
		for index := len(list) - 1; index >= 0; index-- {
			limit := height - 6
			if index < len(list)-1 {
				limit = list[index+1].labelY - 12
			}
			list[index].labelY = math.Max(6, math.Min(list[index].labelY, limit))
		}
		for _, entry := range list {
			if entry.labelY != entry.y2 {
				dy := math.Min(math.Abs(entry.labelY-cy), r+15)
				entry.x2, entry.y2 = cx+side*math.Sqrt((r+15)*(r+15)-dy*dy), entry.labelY
			}
		}
	}
	out := []any{}
	for _, entry := range items {
		x3 := entry.x2 + entry.side*15
		anchor := "start"
		if entry.side < 0 {
			anchor = "end"
		}
		line := "M " + svgNumber(entry.x1) + " " + svgNumber(entry.y1) + " L " + svgNumber(entry.x2) + " " + svgNumber(entry.y2) + " L " + svgNumber(x3) + " " + svgNumber(entry.y2)
		out = append(out, map[string]any{"path": echartsSector(cx, cy, r, entry.start, entry.end), "color": echartsPalette[entry.index%len(echartsPalette)], "line": line,
			"x": svgNumber(x3 + entry.side*5), "y": svgNumber(entry.y2), "anchor": anchor, "name": names[entry.index],
			"value": strconv.FormatFloat(values[entry.index], 'f', -1, 64), "percent": strconv.FormatFloat(percents[entry.index], 'f', -1, 64)})
	}
	return out
}

// echartsPalette is ECharts' default series palette.
var echartsPalette = []string{"#5470c6", "#91cc75", "#fac858", "#ee6666", "#73c0de", "#3ba272", "#fc8452", "#9a60b4", "#ea7ccc"}

// echartsSector is the SVG path of a pie sector from angle a0 to a1
// (clockwise, radians); a full turn is two halves.
func echartsSector(cx, cy, r, a0, a1 float64) string {
	point := func(at float64) string { return svgNumber(cx+r*math.Cos(at)) + " " + svgNumber(cy+r*math.Sin(at)) }
	radius := svgNumber(r)
	if a1-a0 >= 2*math.Pi-1e-6 {
		return "M " + point(a0) + " A " + radius + " " + radius + " 0 1 1 " + point(a0+math.Pi) + " A " + radius + " " + radius + " 0 1 1 " + point(a0) + " Z"
	}
	large := "0"
	if a1-a0 > math.Pi {
		large = "1"
	}
	return "M " + svgNumber(cx) + " " + svgNumber(cy) + " L " + point(a0) + " A " + radius + " " + radius + " 0 " + large + " 1 " + point(a1) + " Z"
}

// echartsPercents is ECharts' getPercentWithPrecision for every value: the
// shares rounded by the largest remainders so they sum to 100.
func echartsPercents(values []float64, precision int) []float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	out := make([]float64, len(values))
	if total == 0 {
		return out
	}
	digits := math.Pow(10, float64(precision))
	target := 100 * digits
	seats, remainders := make([]float64, len(values)), make([]float64, len(values))
	current := 0.0
	for index, value := range values {
		votes := value / total * target
		seats[index] = math.Floor(votes)
		remainders[index] = votes - seats[index]
		current += seats[index]
	}
	for current < target {
		best := 0
		for index := range remainders {
			if remainders[index] > remainders[best] {
				best = index
			}
		}
		seats[best]++
		remainders[best] = 0
		current++
	}
	for index := range seats {
		out[index] = seats[index] / digits
	}
	return out
}
