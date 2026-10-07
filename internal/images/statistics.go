package images

import (
	"strconv"

	"github.com/RayleaBot/genshin/internal/app"
)

// statisticsArtwork maps the images of miao's common and stat/common
// stylesheets to their paths, and the fonts to miao's family names.
var statisticsArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-cont-logo", "resources/common/cont/logo.png"},
	{"stat-imgs-bg1", "resources/stat/imgs/bg1.png"},
	{"stat-imgs-footer", "resources/stat/imgs/footer.png"},
}

// statPercent is miao's pct: the fraction as a percentage to two places.
func statPercent(value float64) string { return app.JSFixed(value*100, 2) }

// Statistics draws miao's stat pages: stat/character for 角色持有率 and
// 命座分布, stat/abyss-pct for 深渊 and 幽境使用率, stat/abyss-team for
// 深渊配队.
func Statistics(context app.ImageContext, page app.StatisticsPage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range statisticsArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	portrait := func(entry app.Entry, file string) string {
		folder, _ := app.CharacterFolders(entry.ID, entry.Name, "")
		return resources.Artwork(file+"-"+entry.ID, "miao-plugin", folder+"imgs/"+file+".webp")
	}
	star := func(entry app.Entry) int {
		if entry.Rarity == 4 {
			return 4
		}
		return 5
	}
	switch {
	case page.Cons != nil:
		cons := page.Cons
		mode, title := "cons", "命座"
		if cons.Holding {
			mode = "char"
		} else if cons.Constel >= 0 {
			title = []string{"零", "一", "二", "三", "四", "五", "满"}[cons.Constel] + "命"
		}
		chars := []any{}
		for index, character := range cons.Characters {
			name := character.Entry.Name
			if character.Entry.Abbr != "" {
				name = character.Entry.Abbr
			}
			hold, none := "未知", "未知"
			if character.Hold != nil {
				hold, none = statPercent(*character.Hold), statPercent(1-*character.Hold)
			}
			shares := []any{}
			for number, value := range character.Cons {
				key := strconv.Itoa(number)
				if number == 0 {
					key = "n0"
				}
				shares = append(shares, map[string]any{"key": key, "pct": statPercent(value)})
			}
			// miao falls back to three stars for characters it has no rarity for.
			rarity := character.Entry.Rarity
			if rarity == 0 {
				rarity = 3
			}
			chars = append(chars, map[string]any{"index": index + 1, "star": min(rarity, 5), "name": name, "side": portrait(character.Entry, "side"), "hold": hold, "none": none, "cons": shares})
		}
		total := ""
		if cons.TotalCount > 0 {
			total = strconv.Itoa(cons.TotalCount)
		}
		return app.Image{Template: "stat-character", Data: map[string]any{"mode": mode, "holding": cons.Holding, "cons_title": title, "total": total, "last_update": cons.LastUpdate, "chars": chars}, Resources: resources.List}, true
	case page.Usage != nil:
		usage := page.Usage
		title := "#" + usage.Name + "使用率"
		if usage.Chosen {
			title = "#" + usage.Name + "第" + usage.FloorName + "使用率"
		}
		ranks := []any{}
		for _, rank := range usage.Ranks {
			chars := []any{}
			for _, character := range rank.Characters {
				chars = append(chars, map[string]any{"star": star(character.Entry), "face": portrait(character.Entry, "face"), "pct": statPercent(character.Value)})
			}
			ranks = append(ranks, map[string]any{"name": rank.Name, "chars": chars})
		}
		return app.Image{Template: "stat-abyss-pct", Data: map[string]any{"title": title, "name": usage.Name, "total": usage.TotalCount, "last_update": usage.LastUpdate, "floor_name": usage.FloorName, "ranks": ranks}, Resources: resources.List}, true
	case page.Teams != nil:
		pairs := []any{}
		for index, pair := range page.Teams.Pairs {
			halves := []any{}
			for _, team := range []app.AbyssTeam{pair.Up, pair.Down} {
				cards := []any{}
				for _, id := range team.IDs {
					avatar := page.Teams.Avatars[id]
					cards = append(cards, map[string]any{"star": star(avatar.Entry), "owned": avatar.Level > 0, "face": portrait(avatar.Entry, "face"), "cons": avatar.Cons, "level": avatar.Level})
				}
				halves = append(halves, cards)
			}
			pairs = append(pairs, map[string]any{"index": index + 1, "count": strconv.FormatFloat(pair.Count, 'f', -1, 64), "halves": halves})
		}
		return app.Image{Template: "stat-abyss-team", Data: map[string]any{"pairs": pairs}, Resources: resources.List}, true
	}
	return app.Image{}, false
}
