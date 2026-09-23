package app

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestReadPanelRankFollowsArk(t *testing.T) {
	answers := asList(cloudList(t, `[{"retcode":100,"rank":"1148 / 60392","score_raw":64248.5,"score":74.17,"percent":"1.90"},{"retcode":102}]`))
	rows := func(ark ArkSettings, changed bool) []Row { return readPanelRank(ark, answers, nil, changed).Rows }
	both := []Row{{Label: "总伤害排名", Value: "1148 / 60392 (1.90%)"}, {Label: "圣遗物排名", Value: "暂无数据"}}
	if got := rows(ArkSettings{QueryType: 2, RankType: 2}, false); !reflect.DeepEqual(got, both) {
		t.Fatal(got)
	}
	// Only the mixed rule writes a rank ark failed to give.
	if got := rows(ArkSettings{QueryType: 2, RankType: 0}, false); !reflect.DeepEqual(got, []Row{{Label: "总伤害排名", Value: "1148 / 60392"}}) {
		t.Fatal(got)
	}
	if got := rows(ArkSettings{QueryType: 0, RankType: 1}, false); !reflect.DeepEqual(got, []Row{{Label: "总伤害排名", Value: "1.90"}}) {
		t.Fatal(got)
	}
	// queryType 1 reads the first answer as the artifact rank.
	if got := rows(ArkSettings{QueryType: 1, RankType: 0}, false); !reflect.DeepEqual(got, []Row{{Label: "圣遗物排名", Value: "1148 / 60392"}}) {
		t.Fatal(got)
	}
	if got := rows(ArkSettings{QueryType: 0, RankType: 2, MarkRankType: true}, true); got[0].Label != "总伤害排名(面板变换)" {
		t.Fatal(got)
	}
	if got := rows(ArkSettings{QueryType: 0, RankType: 2, MarkRankType: true}, false); got[0].Label != "总伤害排名(本地)" {
		t.Fatal(got)
	}
}

func TestReadPanelRankChartsDistributions(t *testing.T) {
	answers := asList(cloudList(t, `[{"retcode":100,"rank":"1148 / 60392","score":74.17,"percent":"1.90"},{"retcode":100,"rank":"69 / 60186","score":95.78,"percent":"0.11"}]`))
	distributions := asList(cloudList(t, `[{"retcode":100,"data":{"scores":["268.70","255.60","248.70","240.40","233.90","223.00","210.80","187.10","170.00","104.90"],"total":60184,"top1":"294.00"}},
		{"retcode":100,"data":{"scores":["68249.76","56693.54","52188.35","48193.57","45706.58","41279.80","35789.10","23696.87","15476.92","4425.55"],"total":60391,"top1":"86620.75"}}]`))
	rank := readPanelRank(ArkSettings{QueryType: 3, RankType: 2}, answers, distributions, false)
	chart := rank.Chart
	if len(rank.Rows) != 0 || chart == nil || chart.Places != [2]string{"1148 / 60392 (1.90%)", "69 / 60186 (0.11%)"} {
		t.Fatalf("rank %+v", rank)
	}
	// Damage scores are percents of the top one; the artifact score is ark's
	// percent of the best artifact score.
	damage, artis := chart.Damage, chart.Artis
	if damage.Top != 100 || math.Abs(damage.Scores[0]-78.79) > 0.01 || damage.Percent != 1.9 || damage.Score != 74.17 {
		t.Fatalf("damage %+v", damage)
	}
	if artis.Top != 294 || artis.Scores[9] != 104.9 || artis.Percent != 0.11 || math.Abs(artis.Score-281.59) > 0.01 {
		t.Fatalf("artis %+v", artis)
	}
	// Without distributions there is nothing to draw, and no rows either.
	if rank := readPanelRank(ArkSettings{QueryType: 3, RankType: 2}, answers, nil, false); rank.Chart != nil || len(rank.Rows) != 0 {
		t.Fatalf("rank %+v", rank)
	}
	// A rank ark failed to give marks nothing.
	rank = readPanelRank(ArkSettings{QueryType: 3, RankType: 2}, asList(cloudList(t, `[{"retcode":102},{"retcode":102}]`)), distributions, false)
	if rank.Chart.Damage.Percent != -100 || rank.Chart.Damage.Score != -100 || rank.Chart.Places[0] != "暂无数据" {
		t.Fatalf("rank %+v", rank.Chart)
	}
}

// cloudList decodes a JSON answer as ark requests read it.
func cloudList(t *testing.T, s string) any {
	t.Helper()
	var v any
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
