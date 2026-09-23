package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/artwork"
)

// xiaoyaoApp is an app with the shipped xiaoyao settings and a downloaded
// library holding a few cards.
func xiaoyaoApp(t *testing.T) (*App, string) {
	t.Helper()
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"10000046","name":"胡桃","aliases":["堂主"],"kind":"character"},
		{"id":"10000005","name":"空","kind":"character"},
		{"id":"20000000","name":"旅行者","kind":"character"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	var lists XiaoyaoAliases
	if err := json.Unmarshal([]byte(`{"version":"test",
		"cards":[{"category":"场地:","cards":[{"name":"层岩巨渊","aliases":["层岩"]}]}],
		"weapons":[{"name":"护摩之杖","aliases":["护摩"]}],
		"enemies":[{"name":"丘丘岩盔王","aliases":["岩盔王"]}]}`), &lists); err != nil {
		t.Fatal(err)
	}
	game.Data = &GameData{Xiaoyao: &lists}
	root := t.TempDir()
	for _, file := range []string{"role/胡桃/胡桃.png", "role/胡桃/胡桃.mp4", "role/丘丘岩盔王/丘丘岩盔王.png", "event/层岩巨渊.png", "event/护摩之杖.png"} {
		writeArtwork(t, root, "xiaoyao-plus/basicInfo_tujian/"+file, "card")
	}
	return &App{Artwork: &artwork.Store{Root: root, Sources: game.Artwork}, Game: game, Catalog: catalog}, root
}

func TestXiaoyaoSendsTheCardsAsRoleInfoAndGetBasicEvent(t *testing.T) {
	a, _ := xiaoyaoApp(t)
	aliases := func() map[string]string { return nil }
	for msg, want := range map[string]string{
		"#七圣胡桃":   "basicInfo_tujian/role/胡桃/胡桃.png",
		"#堂主原牌":   "basicInfo_tujian/role/胡桃/胡桃.png",
		"#七圣召唤胡桃": "basicInfo_tujian/role/胡桃/胡桃.png",
		// 动态 plays the video beside the picture.
		"#胡桃七圣动态": "video basicInfo_tujian/role/胡桃/胡桃.mp4",
		// An enemy is named by yuanmo_tujian.
		"#七圣岩盔王": "basicInfo_tujian/role/丘丘岩盔王/丘丘岩盔王.png",
		// A card without a video is sent as its picture for 动态, where
		// upstream sends the missing video and fails.
		"#七圣岩盔王动态": "basicInfo_tujian/role/丘丘岩盔王/丘丘岩盔王.png",
		"#护摩动态":    "basicInfo_tujian/event/护摩之杖.png",
		"#层岩七圣":    "basicInfo_tujian/event/层岩巨渊.png",
		"#七圣空":     "请选择：风主图鉴、岩主图鉴、雷主图鉴、草主图鉴、水主图鉴、火主图鉴",
		// Without a card the message goes on, as does one xiaoyao does not
		// take: without #, or without its words.
		"#七圣风主":  "",
		"#七圣":    "",
		"#胡桃动态":  "",
		"七圣胡桃":   "",
		"胡桃七圣图鉴": "",
		"#胡桃":    "",
	} {
		reply, ok := a.xiaoyaoCard(msg, a.miaoAccept(msg, aliases), aliases)
		got := reply.text
		switch {
		case reply.play:
			got = "video " + reply.video
		case reply.picture.Path != "":
			got = reply.picture.Path
		}
		if got != want || ok != (want != "") {
			t.Errorf("%q: %q %v, want %q", msg, got, ok, want)
		}
	}
}

// A card remembers the video beside it, which a quote of its picture plays;
// a card without one is not remembered.
func TestXiaoyaoRemembersOnlyCardsWithVideos(t *testing.T) {
	a, _ := xiaoyaoApp(t)
	aliases := func() map[string]string { return nil }
	if reply, _ := a.xiaoyaoCard("#七圣胡桃", "#七圣胡桃", aliases); reply.video != "basicInfo_tujian/role/胡桃/胡桃.mp4" || reply.play {
		t.Fatalf("%+v", reply)
	}
	if reply, _ := a.xiaoyaoCard("#七圣岩盔王", "#七圣岩盔王", aliases); reply.video != "" {
		t.Fatalf("%+v", reply)
	}
	now := time.Unix(1_800_000_000, 0)
	video := artworkFile{"xiaoyao-plus", "basicInfo_tujian/role/胡桃/胡桃.mp4"}
	if reply := a.quoteReply(video, now.Add(-2*time.Hour), now); reply.Type != "video" || !strings.HasPrefix(asText(reply.Data["file"]), "file:///") || !strings.HasSuffix(asText(reply.Data["file"]), "%E8%83%A1%E6%A1%83.mp4") {
		t.Fatalf("%+v", reply)
	}
	// Without a remembered video, a picture sent within the hour gets miao's
	// what.jpg, an older one the reply that it was too long ago.
	if reply := a.quoteReply(artworkFile{}, now.Add(-59*time.Minute), now); reply.Type == "video" {
		t.Fatalf("%+v", reply)
	}
	if reply := a.quoteReply(artworkFile{}, now.Add(-time.Hour), now); asText(reply.Data["text"]) != "消息太过久远了，俺也忘了动态是啥了，下次早点来吧~" {
		t.Fatalf("%+v", reply)
	}
}

func TestQuotedPictureIsOneImageTheBotSent(t *testing.T) {
	image := []any{map[string]any{"type": "image", "data": map[string]any{"url": "https://example.test/a.png"}}}
	for _, tc := range []struct {
		quoted map[string]any
		want   bool
	}{
		{map[string]any{"sender": map[string]any{"user_id": float64(10001)}, "time": float64(1_800_000_000), "message": image}, true},
		{map[string]any{"sender": map[string]any{"user_id": "20002"}, "message": image}, false},
		{map[string]any{"sender": map[string]any{"user_id": "10001"}, "message": append(image, map[string]any{"type": "text", "data": map[string]any{"text": "看"}})}, false},
		{map[string]any{"sender": map[string]any{"user_id": "10001"}, "message": []any{map[string]any{"type": "text", "data": map[string]any{"text": "图"}}}}, false},
	} {
		sent, ok := quotedPicture(tc.quoted, "10001")
		if ok != tc.want || ok && sent.Unix() != 1_800_000_000 {
			t.Errorf("%v: %v %v", tc.quoted, sent, ok)
		}
	}
}
