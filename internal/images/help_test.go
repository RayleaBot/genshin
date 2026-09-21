package images_test

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestHelpFollowsMiao(t *testing.T) {
	help := gamekit.HelpImage{Title: "原神帮助", Subtitle: "插件 1.0.0", Groups: []gamekit.HelpGroup{{Title: "信息查询", Commands: []gamekit.HelpCommand{
		{ID: "note", Usage: "#体力", Description: "实时便笺"}, {ID: "characters", Usage: "#角色", Description: "角色列表"},
		{ID: "training", Usage: "#练度统计", Description: "练度"}, {ID: "tcg", Usage: "#七圣", Description: "七圣召唤"}}}}}
	image, ok := images.Help(gamekit.ImageContext{}, help)
	if !ok || image.Data["title"] != "原神帮助" {
		t.Fatalf("image = %v", image.Data)
	}
	rows := image.Data["groups"].([]any)[0].(map[string]any)["rows"].([]any)
	first, second := rows[0].([]any), rows[1].([]any)
	// miao's icon 15 sits at column 5 of row 2 in its sprite; three columns a
	// row, the last padded with empty cells; unlisted commands have no icon.
	if first[0].(map[string]any)["position"] != "-200px -50px" || first[1].(map[string]any)["position"] != "-0px -300px" {
		t.Errorf("first row = %v", first)
	}
	if len(second) != 3 || second[0].(map[string]any)["position"] != nil || second[1] != nil {
		t.Errorf("second row = %v", second)
	}
}
