// Package assets holds the game data compiled into this plugin.
package assets

import (
	"embed"
	"io/fs"

	plugin "github.com/RayleaBot/plugin-genshin"
	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/images"
	"github.com/RayleaBot/plugin-genshin/internal/reference/miao"
	"github.com/RayleaBot/plugin-genshin/internal/showcase"
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

// Load is the compiled-in game data and image builders the app runs with.
func Load() app.Assets {
	files, err := fs.Sub(calc, "calc")
	if err != nil {
		panic(err) // the embedded directory name is fixed at build time
	}
	return app.Assets{Game: game, Catalog: catalog, Manifest: plugin.Info, Calc: miao.Profile(files), Resources: dataFile("resources.json"), Simulation: dataFile("simulation.json"), CloudPanels: dataFile("cloud-panels.json"), Enemies: dataFile("enemies.json"), Images: images.Builders(), Queries: images.Queries(), Panel: images.Panel, Gacha: images.Gacha, Help: images.Help, MonthlyStats: images.LedgerCount, Calendar: images.Calendar, Entry: images.Entry, SimulationImage: images.GachaTrial, Rank: images.Rank, CloudRank: images.CloudRank, StygianRank: images.StygianRank, Showcase: showcase.Source, PanelList: images.PanelList, UIDList: images.UIDList, RankStats: images.RankStats, ArtifactList: images.ArtifactList, DailyMaterial: images.DailyMaterial, Pools: images.PoolInfo, Statistics: images.Statistics, CharacterCard: images.CharacterCard, PayLog: images.PayLog, RoleCards: images.RoleCards, AtlasIndex: images.AtlasIndex}
}
