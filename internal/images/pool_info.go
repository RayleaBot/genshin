package images

import (
	"slices"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// poolArtwork are the fonts and backgrounds of miao's gacha-info page.
var poolArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
}

// poolElem is the element miao gives the page.
const poolElem = "hydro"

// poolGroups are miao's banner groups in its fixed order.
var poolGroups = []struct {
	title, kind string
	star        int
	names       func(app.PoolInfo) []string
}{
	{"五星角色", "character", 5, func(p app.PoolInfo) []string { return p.Characters5 }},
	{"四星角色", "character", 4, func(p app.PoolInfo) []string { return p.Characters4 }},
	{"五星武器", "weapon", 5, func(p app.PoolInfo) []string { return p.Weapons5 }},
	{"四星武器", "weapon", 4, func(p app.PoolInfo) []string { return p.Weapons4 }},
}

// PoolInfo draws 卡池 the way miao's gacha/gacha-info does: each banner with
// its version, half, chronicled mark and dates, and its characters and
// weapons eight to a row; a character's or weapon's query keeps only the
// row it is in unless 详情 asked for the whole banner. Names miao cannot find
// are left out, as upstream skips them.
func PoolInfo(context app.ImageContext, page app.PoolImage) (app.Image, bool) {
	if context.Game.Calc == nil {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range poolArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	types := map[string]string{}
	for _, weapon := range context.Game.Calc.Metadata().Weapons {
		types[weapon.ID] = weapon.Type
	}
	pools := []any{}
	for _, pool := range page.Pools {
		groups := []any{}
		for _, group := range poolGroups {
			names := group.names(pool)
			if page.Item.Name != "" && !page.Detail && (group.kind != page.Item.Kind || !slices.Contains(names, page.Item.Name)) {
				continue
			}
			rows, row := []any{}, []any{}
			for _, name := range names {
				entry, ok := context.Catalog.Resolve(name, group.kind, nil)
				if !ok {
					continue
				}
				path := poolPicture(entry, types[entry.ID])
				shown := entry.Name
				if utf8.RuneCountInString(shown) > 4 && entry.Abbr != "" {
					shown = entry.Abbr
				}
				row = append(row, map[string]any{"star": entry.Rarity, "name": shown, "img": resources.Artwork(group.kind+"-"+entry.ID, "miao-plugin", path)})
				if len(row) == 8 {
					rows, row = append(rows, row), []any{}
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
			if len(rows) > 0 {
				groups = append(groups, map[string]any{"title": group.title, "rows": rows})
			}
		}
		pools = append(pools, map[string]any{"version": pool.Version, "half": pool.Half, "from": pool.From, "to": pool.To, "mix": pool.Kind == "chronicled", "groups": groups})
	}
	return app.Image{Template: "pool-info", Data: map[string]any{"elem": poolElem, "pools": pools}, Resources: resources.List}, true
}

// poolPicture is miao's face for a character and icon for a weapon.
func poolPicture(entry app.Entry, weaponType string) string {
	if entry.Kind == "weapon" {
		return "resources/meta-gs/weapon/" + weaponType + "/" + entry.Name + "/icon.webp"
	}
	portraits, _ := app.CharacterFolders(entry.ID, entry.Name, "")
	return portraits + "imgs/face.webp"
}
