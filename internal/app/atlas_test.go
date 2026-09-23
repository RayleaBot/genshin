package app

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/artwork"
)

func TestAtlasRulesTakeMessagesAsPickRule(t *testing.T) {
	gallery := []string{"图鉴"}
	cards := []string{"七圣", "牌"}
	materials := []string{"突破", "材料", "素材", "培养"}
	for _, tc := range []struct {
		rule      AtlasRule
		msg, name string
	}{
		{AtlasRule{0, gallery}, "#胡桃", "胡桃"},
		{AtlasRule{0, gallery}, "胡桃", "胡桃"},
		{AtlasRule{1, gallery}, "#胡桃图鉴", "胡桃图鉴"},
		{AtlasRule{1, gallery}, "胡桃图鉴", ""},
		{AtlasRule{2, cards}, "砂糖牌", "砂糖"},
		// Condition 2 leaves the prefix in the name, which then names nothing.
		{AtlasRule{2, cards}, "#七圣砂糖", "#砂糖"},
		{AtlasRule{2, cards}, "#砂糖", ""},
		{AtlasRule{3, gallery}, "护摩之杖图鉴", "护摩之杖"},
		{AtlasRule{3, gallery}, "#护摩之杖", "护摩之杖"},
		{AtlasRule{3, gallery}, "#护摩之杖 图鉴", "护摩之杖"},
		{AtlasRule{3, gallery}, "护摩之杖", ""},
		{AtlasRule{3, gallery}, "#图鉴", ""},
		{AtlasRule{4, materials}, "#胡桃突破", "胡桃"},
		{AtlasRule{4, materials}, "胡桃突破", ""},
		{AtlasRule{4, materials}, "#胡桃", ""},
		{AtlasRule{5, materials}, "胡桃培养", "胡桃"},
		{AtlasRule{5, materials}, "#胡桃材料", "胡桃"},
		{AtlasRule{5, materials}, "#胡桃", ""},
		// The prefix counts only before a single line, as ^#.*$ reads it.
		{AtlasRule{1, gallery}, "#胡桃\n天赋", ""},
		// Without Pick words the rule picks 图鉴.
		{AtlasRule{2, nil}, "胡桃图鉴", "胡桃"},
	} {
		rule, err := tc.rule.compile()
		if err != nil {
			t.Fatal(err)
		}
		if name, ok := rule.take(tc.msg); name != tc.name || ok != (tc.name != "") {
			t.Errorf("condition %d took %q as %q %v, want %q", tc.rule.Condition, tc.msg, name, ok, tc.name)
		}
	}
}

func TestShippedAtlasRulesCompile(t *testing.T) {
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	for _, library := range game.Pictures.Atlas {
		if _, ok := library.Rules["config"]; !ok {
			t.Errorf("%s has no config rule", library.Source)
		}
		for name, rule := range library.Rules {
			if _, err := rule.compile(); err != nil || rule.Condition < 0 || rule.Condition > 5 {
				t.Errorf("%s rule %s: %v", library.Source, name, err)
			}
		}
	}
}

// atlasApp is an app with the shipped Atlas rules, a downloaded library and
// Atlas's own alias files.
func atlasApp(t *testing.T) (*App, string) {
	t.Helper()
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"10000046","name":"胡桃","aliases":["堂主"],"kind":"character"},
		{"id":"10000052","name":"雷电将军","aliases":["雷神"],"kind":"character"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeArtwork(t, root, "genshin-atlas/path.json", `{"enemy":{"雷电将军":"/enemy/雷电将军.png","丘丘人":"/enemy/丘丘人.png"},"card":{"砂糖":"/card/砂糖.png"},"material":{"天赋":"/material/天赋.png"},"weapon":{"护摩之杖":"/weapon/护摩之杖.png","雾切":"/weapon/雾切.yaml"},"material for role":{"胡桃":"/material for role/胡桃.png"}}`)
	for _, file := range []string{"enemy/雷电将军.png", "enemy/丘丘人.png", "card/砂糖.png", "material/天赋.png", "weapon/护摩之杖.png", "weapon/雾切.yaml", "material for role/胡桃.png"} {
		writeArtwork(t, root, "genshin-atlas/"+file, "image")
	}
	writeArtwork(t, root, "atlas/resource/Forlibrary/Genshin-Atlas/othername/weapon.yaml", "护摩之杖:\n  - 护摩\n  - 胡桃专武\n")
	return &App{Artwork: &artwork.Store{Root: root, Sources: game.Artwork}, Game: game, Catalog: catalog}, root
}

// atlasReply is the picture Atlas answers a message with, "" when it leaves
// the message to others.
func atlasReply(a *App, msg string) string {
	aliases := func() map[string]string { return nil }
	run := atlasRun{app: a, aliases: aliases}
	if !run.atlas(a.miaoAccept(msg, aliases)) || len(run.answers) == 0 {
		return ""
	}
	return run.answers[0].picture.Path
}

func TestAtlasAnswersMessagesBeforeTheCommands(t *testing.T) {
	a, _ := atlasApp(t)
	for msg, want := range map[string]string{
		// Atlas's own copy of the library's aliases names the key.
		"#护摩图鉴":   "weapon/护摩之杖.png",
		"胡桃专武图鉴":  "weapon/护摩之杖.png",
		"#护摩之杖":   "weapon/护摩之杖.png",
		"#丘丘人":    "enemy/丘丘人.png",
		"#天赋":     "material/天赋.png",
		"胡桃培养":    "material for role/胡桃.png",
		"#胡桃突破":   "material for role/胡桃.png",
		"#堂主材料 ":  "material for role/胡桃.png",
		"砂糖牌":     "card/砂糖.png",
		"七圣砂糖":    "card/砂糖.png",
		"#砂糖牌":    "",
		"丘丘人":     "",
		"#雾切图鉴":   "",
		"#不存在的图鉴": "",
		// miao's checks take a character's name and its wiki words first.
		"#雷电将军":   "",
		"#雷神":     "",
		"#雷神图鉴":   "",
		"雷电将军图鉴":  "",
		"#雷电将军卡片": "",
	} {
		if got := atlasReply(a, msg); got != want {
			t.Errorf("%q answered with %q, want %q", msg, got, want)
		}
	}
}

func TestAtlasReadsTheLibraryAliasesFirstAndAgainAfterADownload(t *testing.T) {
	a, root := atlasApp(t)
	writeArtwork(t, root, "genshin-atlas/othername/weapon.yaml", "# weapons\n'护摩之杖':\n- \"摩摩\"\n")
	if got := atlasReply(a, "#摩摩"); got != "weapon/护摩之杖.png" {
		t.Fatal(got)
	}
	if got := atlasReply(a, "#护摩"); got != "" {
		t.Fatal("Atlas's copy was read beside the library's", got)
	}
	// A new download of the library is read again.
	writeArtwork(t, root, "genshin-atlas/path.json", `{"weapon":{"护摩之杖":"/weapon/护摩之杖.png","天空之刃":"/weapon/护摩之杖.png"}}`)
	if got := atlasReply(a, "#天空之刃"); got != "" {
		t.Fatal("the library was read again before a download", got)
	}
	writeArtwork(t, root, "genshin-atlas.json", `{}`)
	if got := atlasReply(a, "#天空之刃"); got != "weapon/护摩之杖.png" {
		t.Fatal("the downloaded library was not read", got)
	}
}
