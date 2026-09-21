// Package assets holds the game data compiled into this plugin.
package assets

import (
	"embed"
	"io/fs"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/reference/miao"
)

//go:embed catalog.json
var catalog []byte

//go:embed game.json
var game []byte

//go:embed manifest.json
var manifest []byte

// calc holds the pinned upstream calculation scripts, see calc/catalog.json.
//
//go:embed calc
var calc embed.FS

// Kit is everything the shared game library needs from this plugin.
func Kit() gamekit.Assets {
	files, err := fs.Sub(calc, "calc")
	if err != nil {
		panic(err) // the embedded directory name is fixed at build time
	}
	return gamekit.Assets{Game: game, Catalog: catalog, Manifest: manifest, Calc: miao.Profile(files)}
}
