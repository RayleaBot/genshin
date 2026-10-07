package images

import (
	"strconv"

	"github.com/RayleaBot/genshin/internal/app"
)

// stygianRankArtwork maps the images named in the converted stylesheets to
// their sources and paths: the fonts, the hydro background and the card
// background ark's resources hold (the same files as miao's).
var stygianRankArtwork = [][3]string{
	{"Number", "miao-plugin", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "miao-plugin", "resources/common/font/NZBZ.woff"},
	{"YS", "miao-plugin", "resources/common/font/HYWH-65W.woff"},
	{"common-bg-bg-hydro", "miao-plugin", "resources/common/bg/bg-hydro.webp"},
	{"common-cont-card-bg", "miao-plugin", "resources/common/cont/card-bg.png"},
}

// StygianRank draws ark-plugin's 幽境危战排名 the way its
// character/stygian-rank-list does: the season and its period, then a row per
// UID with its place, the member's avatar (or akasha.cv's), name and UID, the
// difficulty's medal (6+ within 180 seconds), the time and the global places
// on ark and akasha.cv, each column only when its service answered. The page
// is 860 pixels wide less 140 for each column left out.
func StygianRank(context app.ImageContext, rank app.StygianRankImage) (app.Image, bool) {
	if len(rank.Entries) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range stygianRankArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	rows := []any{}
	for index, entry := range rank.Entries {
		medal := "medal_" + strconv.Itoa(entry.Index)
		if entry.Index == 6 && entry.Seconds <= 180 {
			medal = "medal_6_plus"
		}
		row := map[string]any{"place": index + 1, "uid": entry.UID, "name": entry.Name, "index": entry.Index, "seconds": entry.Seconds, "ark": entry.Ark, "akasha": entry.Akasha,
			"medal": resources.Artwork(medal, "ark-plugin", "resources/character/img/"+medal+".png")}
		if entry.Avatar != "" {
			row["avatar"] = resources.Remote("avatar-"+strconv.Itoa(index), entry.Avatar)
		}
		rows = append(rows, row)
	}
	width := 860
	for _, shown := range []bool{rank.Ark, rank.Akasha} {
		if !shown {
			width -= 140
		}
	}
	data := map[string]any{"title": context.Game.Prefix + "幽境危战排名", "version": rank.Version, "start": rank.Start.In(chinaTime).Format("2006-01-02 15:04:05"), "end": rank.End.In(chinaTime).Format("2006-01-02 15:04:05"),
		"rows": rows, "ark": rank.Ark, "akasha": rank.Akasha, "width": width}
	return app.Image{Template: "stygian-rank", Data: data, Resources: resources.List}, true
}
