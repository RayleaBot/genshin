package app

import (
	"reflect"
	"strings"
	"testing"
)

func TestEChartsDistributionSmoothsAsECharts(t *testing.T) {
	chart, at := EChartsDistribution([]float64{0, 50, 100}, []float64{100, 50, 0}, 100, 20)
	// The plot runs from x 80 to 720 and y 60 to 340.
	if x, y := at(0, 100); x != 80 || y != 60 {
		t.Fatal(x, y)
	}
	// Points on a straight line keep their control points on it.
	if line := chart["line"].(string); !strings.HasPrefix(line, "M 80 60 C 80 60 ") || !strings.HasSuffix(line, "720 340 720 340") {
		t.Fatal(line)
	}
	if ticks := chart["y_ticks"].([]any); len(ticks) != 6 {
		t.Fatal(ticks)
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
