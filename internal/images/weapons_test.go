package images_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/artwork"
	"github.com/RayleaBot/genshin/internal/assets"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestWeaponsFollowYunzai(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	side := filepath.Join(root, "miao-plugin", filepath.FromSlash("resources/meta-gs/character/荧/imgs/side.webp"))
	if err := os.MkdirAll(filepath.Dir(side), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(side, []byte("webp"), 0o600); err != nil {
		t.Fatal(err)
	}
	list := decode(t, `{"list":[
		{"id":10000046,"level":90,"rarity":5,"weapon":{"id":13501,"type":13,"rarity":5,"level":90,"affix_level":1}},
		{"id":10000007,"level":80,"rarity":5,"weapon":{"id":11412,"type":1,"rarity":4,"level":90,"affix_level":5}},
		{"id":10000032,"level":90,"rarity":4,"weapon":{"id":11302,"type":1,"rarity":3,"level":90,"affix_level":5}},
		{"id":10000020,"level":80,"rarity":4,"weapon":{"id":12503,"type":11,"rarity":5,"level":90,"affix_level":2}},
		{"id":10000021,"level":1,"rarity":4,"weapon":{"id":15101,"type":12,"rarity":1,"level":1,"affix_level":1}}]}`)
	image, ok := images.Weapons(app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Now(), Artwork: &artwork.Store{Root: root}},
		app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	if !ok || image.Template != "weapons-narrow" || image.Data["counted"] != false {
		t.Fatalf("image = %v %#v", ok, image)
	}
	// Rarity, refinement and level decide; one-star weapons are left out and
	// long names give way to upstream's short ones.
	items := image.Data["list"].([]any)
	names := []string{}
	for _, raw := range items {
		names = append(names, raw.(map[string]any)["name"].(string))
	}
	if len(names) != 4 || names[0] != "松籁之时" || names[1] != "护摩之杖" || names[2] != "降临之剑" || names[3] != "黎明神剑" {
		t.Fatalf("names = %v", names)
	}
	traveler := items[2].(map[string]any)
	if traveler["affix"] != "5" || traveler["affix_class"] != "life5" || traveler["bg"] != "bg4" || traveler["side"] == "" {
		t.Fatalf("traveler = %#v", traveler)
	}
	if _, refined := items[1].(map[string]any)["affix"]; refined {
		t.Fatal("a first refinement carries no badge")
	}
	count := image.Data["count"].(map[string]any)
	if count["5"] != "2" || count["4"] != "1" || count["3"] != "1" || count["sword"] != "2" || count["claymore"] != "1" || count["polearm"] != "1" || count["bow"] != "0" {
		t.Fatalf("count = %#v", count)
	}
}
