package images

import (
	"github.com/RayleaBot/genshin/internal/app"
)

// rankStatsArtwork maps the miao images the 排名统计 stylesheet names, those of
// the panel's 排名统计 block, to their repository paths.
var rankStatsArtwork = [][2]string{
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
}

// RankStats draws <角色>排名统计 as the panel's 排名统计 block, on the
// character's element background: ark's damage and artifact distributions
// with no panel to mark, so the area keeps one colour and nothing is written
// on the curve; beside the title 排名趋势 and the calculation the damage
// ranks, and under each chart the UIDs ark counted, as ark's graph/stats
// wrote them, or 暂无数据 as under a chart ark did not send.
func RankStats(context app.ImageContext, image app.RankStatsImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range rankStatsArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	elem := ""
	if context.Game.Calc != nil {
		for _, record := range context.Game.Calc.Metadata().Characters {
			if record.ID == image.Character.ID {
				elem = record.Element
				break
			}
		}
	}
	if elem != "" {
		resources.Artwork("common-bg-bg-"+elem, "miao-plugin", "resources/common/bg/bg-"+elem+".webp")
	}
	charts, totals := []any{}, []any{}
	for _, item := range []struct {
		kind  string
		curve *app.RankCurve
	}{{"damage", image.Damage}, {"artis", image.Artis}} {
		chart, at := rankChart(item.kind, item.curve)
		total := "暂无数据"
		if at != nil {
			chart["stops"] = rankStops(rankStop{0, rankFill}, rankStop{1, rankFill})
			if item.curve.Total != "" {
				total = "统计样本 " + item.curve.Total + " 个UID"
			}
		}
		charts, totals = append(charts, chart), append(totals, total)
	}
	return app.Image{Template: "rank-stats", Data: map[string]any{"elem": elem, "name": image.Character.Name, "title": image.DamageTitle, "charts": charts, "totals": totals}, Resources: resources.List}, true
}
