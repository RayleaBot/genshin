package app

import (
	"os"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/reference"
	"github.com/RayleaBot/plugin-genshin/internal/reference/miao"
)

// pluginFile reads a file of this plugin, for tests that need the shipped
// data.
func pluginFile(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func calcProfile() reference.Profile {
	return miao.Profile(os.DirFS("../../internal/assets/calc"))
}

func calcEngine(t *testing.T) *reference.Engine {
	t.Helper()
	engine, err := reference.New(calcProfile())
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

// pluginAssets mirrors what the plugin embeds.
func pluginAssets(t *testing.T) Assets {
	t.Helper()
	optional := func(name string) []byte {
		raw, err := os.ReadFile("../../internal/assets/data/" + name)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return raw
	}
	return Assets{
		Game:        pluginFile(t, "internal/assets/game.json"),
		Catalog:     pluginFile(t, "internal/assets/catalog.json"),
		Manifest:    pluginFile(t, "info.json"),
		Calc:        calcProfile(),
		Resources:   pluginFile(t, "internal/assets/data/resources.json"),
		Simulation:  optional("simulation.json"),
		CloudPanels: optional("cloud-panels.json"),
		Enemies:     optional("enemies.json"),
	}
}

// pluginApp builds the app exactly as the plugin does at start.
func pluginApp(t *testing.T) *App {
	t.Helper()
	a, err := New(pluginAssets(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	return a
}

// testGame is the game descriptor with its real calculation engine, for tests
// that reach panel or build calculations without a full app.
func testGame(t *testing.T) Game {
	t.Helper()
	data, err := parseGameData(pluginAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	return Game{ID: "genshin", Calc: calcEngine(t), Data: data}
}
