package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/artwork"
)

func writeArtwork(t *testing.T, root, name, content string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPicturesReadDownloadedLibraries(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Prefix: "#", Pictures: Pictures{
		Photos: []PictureSource{{Source: "miao-plugin", Paths: []string{"resources/character-img/{name}"}}},
		Atlas:  []PictureSource{{Source: "genshin-atlas", Index: "path.json"}, {Source: "xiaoyao-plus", Paths: []string{"wuqi_tujian/{name}.png"}}},
	}}}
	if photos := a.characterPhotos(Entry{Name: "七七"}); len(photos) != 0 {
		t.Fatal("photos before a download", photos)
	}
	if hint := a.pictureHint(a.Game.Pictures.Atlas); !strings.Contains(hint, "#素材更新 genshin-atlas xiaoyao-plus") {
		t.Fatal(hint)
	}
	writeArtwork(t, root, "miao-plugin/resources/character-img/空/01.jpg", "a")
	writeArtwork(t, root, "miao-plugin/resources/character-img/荧/01.webp", "b")
	writeArtwork(t, root, "miao-plugin/resources/character-img/荧/notes.txt", "c")
	// The Traveler's photos are both twins', as miao's.
	if photos := a.characterPhotos(Entry{Name: "旅行者"}); len(photos) != 2 || photos[1].Path != "resources/character-img/荧/01.webp" {
		t.Fatal(photos)
	}
	// Atlas searches path.json's modules in the file's order.
	writeArtwork(t, root, "genshin-atlas/path.json", `{"weapon":{"护摩之杖":"/weapon/护摩之杖.png"},"card":{"护摩之杖":"/card/护摩之杖.png"}}`)
	writeArtwork(t, root, "genshin-atlas/weapon/护摩之杖.png", "d")
	writeArtwork(t, root, "genshin-atlas/card/护摩之杖.png", "e")
	writeArtwork(t, root, "xiaoyao-plus/wuqi_tujian/天空之刃.png", "f")
	if file, ok := a.atlasPicture([]string{"护摩", "护摩之杖"}); !ok || file != (artworkFile{"genshin-atlas", "weapon/护摩之杖.png"}) {
		t.Fatal(file, ok)
	}
	if file, ok := a.atlasPicture([]string{"天空之刃"}); !ok || file.Source != "xiaoyao-plus" {
		t.Fatal(file, ok)
	}
	if _, ok := a.atlasPicture([]string{"不存在"}); ok {
		t.Fatal("found a missing name")
	}
	if hint := a.pictureHint(a.Game.Pictures.Atlas); hint != "" {
		t.Fatal(hint)
	}
}

func TestAtlasFindsPicturesByTheLibraryAliases(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Pictures: Pictures{Atlas: []PictureSource{{Source: "atlas", Index: "path.json", Skip: []string{"guide for role"}}}}}}
	writeArtwork(t, root, "atlas/guide/1001.png", "guide")
	if _, ok := a.atlasPicture([]string{"role"}); ok {
		t.Error("sent an alias file")
	}
	// Star Rail's library keys pictures by ID; othername lists each key's
	// names.
	writeArtwork(t, root, "atlas/path.json", `{"guide for role":{"1001":"/guide/1001.png"},"othername":{"role":"/othername/role.yaml"},"role":{"1001":"/role/1001.png","1002":"/role/1002.png"}}`)
	writeArtwork(t, root, "atlas/othername/role.yaml", "# roles\n'1001':\n  - 三月七\n  - mar7th\n\"1002\":\n- 丹恒\n- 三月七\n")
	writeArtwork(t, root, "atlas/role/1001.png", "a")
	writeArtwork(t, root, "atlas/role/1002.png", "b")
	// The guide module answers 攻略, not 图鉴.
	for name, want := range map[string]string{"三月七": "role/1001.png", "mar7th": "role/1001.png", "丹恒": "role/1002.png", "1002": "role/1002.png"} {
		if file, ok := a.atlasPicture([]string{name}); !ok || file.Path != want {
			t.Errorf("%s: %+v %v", name, file, ok)
		}
	}
}
