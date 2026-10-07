package app

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/RayleaBot/genshin/internal/artwork"
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

// atlasAnswers are the replies Atlas sends for a sender's message, a picture
// as its path and a list joined, with whether Atlas ends the message.
func atlasAnswers(a *App, sender, msg string) ([]string, bool) {
	aliases := func() map[string]string { return nil }
	run := atlasRun{app: a, owner: Subject{ActorID: sender}, aliases: aliases}
	ended := run.atlas(a.miaoAccept(msg, aliases))
	replies := []string{}
	for _, answer := range run.answers {
		if answer.list != nil {
			replies = append(replies, strings.Join(answer.list, " "))
		} else {
			replies = append(replies, answer.picture.Path)
		}
	}
	return replies, ended
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

func TestAtlasIndexListsAndPicksByNumber(t *testing.T) {
	a, root := atlasApp(t)
	writeArtwork(t, root, "genshin-atlas/index/weapon.yaml", "武器索引:\n  - 五星武器\n  - 四星武器\n五星武器:\n  - 护摩之杖\n  - 雾切\n")
	writeArtwork(t, root, "genshin-atlas/index/card.yaml", "召唤:\n  - 角色\n")
	steps := []struct {
		sender, msg string
		replies     []string
		ended       bool
	}{
		// A key of the index answers with its list, led by the prefix the
		// module's rule takes, and the message goes on to the others.
		{"a", "#五星武器", []string{"#护摩之杖 #雾切"}, false},
		// A number picks an entry, which answers and ends the message; the
		// pick counts twice, once more as the entry runs through Atlas, and
		// an answer resets the count.
		{"a", "1", []string{"weapon/护摩之杖.png"}, true},
		{"b", "1", []string{}, false},
		{"a", "1", []string{"weapon/护摩之杖.png"}, true},
		// An entry without a picture answers nothing.
		{"a", "2", []string{}, false},
		{"a", "你好", []string{}, false},
		// The fourth count without an answer drops the list.
		{"a", "1", []string{}, false},
		{"a", "1", []string{}, false},
		// An entry that is itself a key answers with its list and ends the
		// message; its count is left at two, so the next pick drops the list
		// as it answers.
		{"b", "#武器索引", []string{"#五星武器 #四星武器"}, false},
		{"b", "1", []string{"#护摩之杖 #雾切"}, true},
		{"b", "1", []string{"weapon/护摩之杖.png"}, true},
		{"b", "1", []string{}, false},
		// Condition 2 leads the entries with its first Pick word.
		{"c", "七圣召唤", []string{"七圣角色"}, false},
		// miao's checks leave Atlas their own command word, which still counts.
		{"d", "#五星武器", []string{"#护摩之杖 #雾切"}, false},
		{"d", "#雷神", []string{}, false},
		{"d", "#雷神图鉴", []string{}, false},
		{"d", "#雷电将军", []string{}, false},
		{"d", "1", []string{}, false},
	}
	for _, step := range steps {
		replies, ended := atlasAnswers(a, step.sender, step.msg)
		if !reflect.DeepEqual(replies, step.replies) || ended != step.ended {
			t.Fatalf("%s %q: %q %v, want %q %v", step.sender, step.msg, replies, ended, step.replies, step.ended)
		}
	}
}

func TestAtlasPagesFollowReplyMessageArray(t *testing.T) {
	lines := []string{"请直接发送数字序号或对应指令："}
	for index := range 200 {
		lines = append(lines, strconv.Itoa(index+1)+"、#五星武器")
	}
	pages := atlasPages(lines)
	// Lines join until a message reaches 100 characters.
	if first := pages[0][0]; strings.Count(first, "\n") != 11 || !strings.HasPrefix(first, "请直接发送数字序号或对应指令：\n1、#五星武器") {
		t.Fatalf("%q", first)
	}
	if len(pages) != 1 || pages[0][len(pages[0])-1] != "第1页，共1页" {
		t.Fatal(pages)
	}
	many := []string{}
	for range 2000 {
		many = append(many, strings.Repeat("长", 100))
	}
	pages = atlasPages(many)
	if len(pages) != 22 || len(pages[0]) != 96 || pages[0][95] != "第1页...未完待续" || pages[21][len(pages[21])-1] != "第22页，共22页" {
		t.Fatal(len(pages), len(pages[0]), pages[21][len(pages[21])-1])
	}
}

func TestAtlasHelpWords(t *testing.T) {
	for msg, want := range map[string]bool{
		"#图鉴帮助":       true,
		"#图鉴 帮助":      true,
		"/wiki菜单":     true,
		"#Atlas help": true,
		"#百科功能列表":     true,
		"图鉴帮助":        false,
		"#atlas帮助":    false,
		"#图鉴":         false,
	} {
		if atlasHelpWords.MatchString(msg) != want {
			t.Errorf("%q: %v", msg, !want)
		}
	}
}
