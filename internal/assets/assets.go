// Package assets holds the game data compiled into this plugin.
package assets

import (
	"embed"
	"io/fs"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/reference/miao"
	plugin "github.com/RayleaBot/plugin-genshin"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

//go:embed catalog.json
var catalog []byte

//go:embed game.json
var game []byte

// calc holds the pinned upstream calculation scripts, see calc/catalog.json.
//
//go:embed calc
var calc embed.FS

// data holds this game's fixed reference data: materials, banners and
// birthdays, plus the optional feature data listed in Kit.
//
//go:embed data
var data embed.FS

func dataFile(name string) []byte {
	raw, err := data.ReadFile("data/" + name)
	if err != nil {
		panic(err) // the embedded file names are fixed at build time
	}
	return raw
}

// Kit is everything the shared game library needs from this plugin.
func Kit() gamekit.Assets {
	files, err := fs.Sub(calc, "calc")
	if err != nil {
		panic(err) // the embedded directory name is fixed at build time
	}
	return gamekit.Assets{Game: game, Catalog: catalog, Manifest: plugin.Info, Calc: miao.Profile(files), Resources: dataFile("resources.json"), Simulation: dataFile("simulation.json"), CloudPanels: dataFile("cloud-panels.json"), Enemies: dataFile("enemies.json"), Images: images.Builders(), Queries: images.Queries(), Panel: images.Panel, Gacha: images.Gacha, Help: images.Help, MonthlyStats: images.LedgerCount}
}
