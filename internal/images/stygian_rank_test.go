package images_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
	"github.com/RayleaBot/genshin/internal/artwork"
	"github.com/RayleaBot/genshin/internal/images"
)

func TestStygianRankFollowsArk(t *testing.T) {
	root := t.TempDir()
	for _, medal := range []string{"medal_6", "medal_6_plus"} {
		file := filepath.Join(root, "ark-plugin", "resources", "character", "img", medal+".png")
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	context := app.ImageContext{Game: app.Game{Prefix: "#"}, Artwork: &artwork.Store{Root: root}}
	entries := []app.StygianRankEntry{{UID: "1", Name: "甲", Index: 6, Seconds: 180}, {UID: "2", Name: "乙", Index: 6, Seconds: 181, Avatar: "https://q1.qlogo.cn/g?b=qq&nk=2&s=100"}}
	drawn, ok := images.StygianRank(context, app.StygianRankImage{Version: "7.0", Entries: entries, Ark: true, Akasha: true})
	if !ok || drawn.Template != "stygian-rank" || drawn.Data["width"] != 860 || drawn.Data["title"] != "#幽境危战排名" {
		t.Fatalf("drawn = %+v", drawn.Data)
	}
	// 6+ within 180 seconds, as ark's medal.
	rows := drawn.Data["rows"].([]any)
	first, second := rows[0].(map[string]any), rows[1].(map[string]any)
	if first["medal"] != "medal_6_plus" || second["medal"] != "medal_6" || first["avatar"] != nil || second["avatar"] == nil {
		t.Fatalf("rows = %v", rows)
	}
	// Each service that did not answer takes its 140-pixel column away.
	for _, tc := range []struct {
		ark, akasha bool
		width       int
	}{{false, true, 720}, {false, false, 580}} {
		drawn, _ = images.StygianRank(context, app.StygianRankImage{Entries: entries, Ark: tc.ark, Akasha: tc.akasha})
		if drawn.Data["width"] != tc.width {
			t.Errorf("ark %v akasha %v: width %v", tc.ark, tc.akasha, drawn.Data["width"])
		}
	}
}
