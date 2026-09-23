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
		Photos:  []PictureSource{{Source: "miao-plugin", Paths: []string{"resources/character-img/{name}"}}},
		Atlas:   []AtlasLibrary{{Source: "genshin-atlas", Index: "path.json"}},
		Xiaoyao: PictureSource{Source: "xiaoyao-plus", Paths: []string{"wuqi_tujian/{name}.png"}},
	}}}
	if photos := a.characterPhotos(Entry{Name: "七七"}); len(photos) != 0 {
		t.Fatal("photos before a download", photos)
	}
	if hint := a.pictureHint(a.Game.Pictures.catalogSources()...); !strings.Contains(hint, "#素材更新 genshin-atlas xiaoyao-plus") {
		t.Fatal(hint)
	}
	writeArtwork(t, root, "miao-plugin/resources/character-img/空/01.jpg", "a")
	writeArtwork(t, root, "miao-plugin/resources/character-img/荧/01.webp", "b")
	writeArtwork(t, root, "miao-plugin/resources/character-img/荧/notes.txt", "c")
	// The Traveler's photos are both twins', as miao's.
	if photos := a.characterPhotos(Entry{Name: "旅行者"}); len(photos) != 2 || photos[1].Path != "resources/character-img/荧/01.webp" {
		t.Fatal(photos)
	}
	writeArtwork(t, root, "genshin-atlas/path.json", `{}`)
	writeArtwork(t, root, "xiaoyao-plus/wuqi_tujian/天空之刃.png", "f")
	if file, ok := a.xiaoyaoPicture([]string{"天空", "天空之刃"}); !ok || file != (artworkFile{"xiaoyao-plus", "wuqi_tujian/天空之刃.png"}) {
		t.Fatal(file, ok)
	}
	if _, ok := a.xiaoyaoPicture([]string{"不存在"}); ok {
		t.Fatal("found a missing name")
	}
	if hint := a.pictureHint(a.Game.Pictures.catalogSources()...); hint != "" {
		t.Fatal(hint)
	}
}
