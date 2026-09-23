package app

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestEChartsLineFollowsEChartsScale(t *testing.T) {
	// A fixed maximum off the nice interval ends the axis with itself.
	if ticks := echartsTicks(112700, 6); !slices.Equal(ticks, []float64{0, 20000, 40000, 60000, 80000, 100000, 112700}) {
		t.Fatal(ticks)
	}
	if ticks := echartsTicks(60, 6); !slices.Equal(ticks, []float64{0, 10, 20, 30, 40, 50, 60}) {
		t.Fatal(ticks)
	}
	chart := EChartsLine([]float64{0, 50, 100}, []float64{100, 50, 0}, 1120, 520, [4]float64{30, 40, 50, 80}, 100, 6, 0.4, func(v float64) string { return "" })
	points := chart["points"].([]any)
	// The plot runs from x 80 to 1080 and y 30 to 470.
	if first := points[0].(map[string]any); first["x"] != "80" || first["y"] != "30" || points[2].(map[string]any)["y"] != "470" {
		t.Fatal(points)
	}
	// Points on a straight line keep their control points on it.
	if line := chart["line"].(string); !strings.HasPrefix(line, "M 80 30 C 80 30 ") || !strings.HasSuffix(line, "1080 470 1080 470") {
		t.Fatal(line)
	}
}

func TestEChartsPieRoundsPercentsToAHundred(t *testing.T) {
	if got := echartsPercents([]float64{1, 1, 1}, 2); !reflect.DeepEqual(got, []float64{33.34, 33.33, 33.33}) {
		t.Fatal(got)
	}
	// Twelve months of labels too wide for their bands keep every other one.
	chart := EChartsBar([]string{"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"}, make([]float64, 12), 470, 300, [4]float64{60, 47, 60, 47}, 5, func(label string) float64 { return float64(len([]rune(label))) * 12 })
	bars := chart["bars"].([]any)
	if bars[0].(map[string]any)["label"] != "1月" || bars[1].(map[string]any)["label"] != nil || bars[2].(map[string]any)["label"] != "3月" {
		t.Fatal(bars[:3])
	}
}
