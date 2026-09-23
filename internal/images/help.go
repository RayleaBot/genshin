package images

import (
	"fmt"

	"github.com/RayleaBot/plugin-genshin/internal/app"
)

// helpIcons gives each command the icon of the miao help entry that lists it
// (config/help_default.js); commands miao does not list show none, as miao
// hides an entry's missing icon.
var helpIcons = map[string]int{
	"profile": 61, "characters": 61, "note": 15, "training": 62, "talent-stat": 62, "accounts": 10, "select": 10,
	"monthly": 5, "monthly-history": 5,
	"character": 66, "build": 66, "panel-list": 63, "panel-refresh": 63, "score": 65,
	"abyss-summary": 64, "abyss": 64, "abyss-floor": 64, "theater": 64, "hard_challenge": 64, "challenge-submit": 77,
	"gacha-detail": 6, "gacha-background": 6, "gacha-stat": 21, "simulation": 8, "simulation-fate": 8,
	"catalog": 60, "talent-wiki": 53, "materials": 67, "guides": 20, "photo": 88, "image-library": 88, "interaction": 59,
	"live-calendar": 83, "signin": 86, "role-cards": 35, "role-cards-exchange": 35,
	"help": 79, "version": 79, "group-settings": 32, "artwork": 35, "artwork-status": 35,
}

// helpColumns is miao's default column count.
const helpColumns = 3

// helpArtwork maps the images named in the converted stylesheets (miao's
// common and help/index, with the default theme) to their paths.
var helpArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"help-icon", "resources/help/icon.png"},
	{"help-theme-bg", "resources/help/theme/default/bg.jpg"},
	{"help-theme-main", "resources/help/theme/default/main.png"},
}

// Help draws the help menu the way miao's help/index does: the title bar, then
// each group as a table of three columns, each command with its icon, usage
// and description, on miao's default theme.
func Help(context app.ImageContext, help app.HelpImage) (app.Image, bool) {
	if len(help.Groups) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range helpArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	groups := []any{}
	for _, group := range help.Groups {
		rows, row := []any{}, []any{}
		for _, command := range group.Commands {
			cell := map[string]any{"title": command.Usage, "desc": command.Description}
			// miao's sprite has ten 50px icons a row.
			if icon := helpIcons[command.ID]; icon > 0 {
				x, y := (icon-1)%10, (icon-1)/10
				cell["position"] = fmt.Sprintf("-%dpx -%dpx", x*50, y*50)
			}
			row = append(row, cell)
			if len(row) == helpColumns {
				rows, row = append(rows, row), []any{}
			}
		}
		if len(row) > 0 {
			for len(row) < helpColumns {
				row = append(row, nil)
			}
			rows = append(rows, row)
		}
		groups = append(groups, map[string]any{"title": group.Title, "rows": rows})
	}
	return app.Image{Template: "help", Data: map[string]any{"title": help.Title, "subtitle": help.Subtitle, "groups": groups}, Resources: resources.List}, true
}
