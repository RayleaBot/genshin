package app

import (
	"context"
	"encoding/json"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

func TestGachaFilesRoundTripAndFillLegacyRecords(t *testing.T) {
	a := pluginApp(t)
	// A UIGF v4 file keeps long IDs exact, drops non-standard fields, and
	// takes names and ranks from the catalog when a record leaves them out.
	file := `{"info":{"version":"v4.0","authkey":"synthetic"},"hkrpg":[{"uid":"100000009","timezone":8,"list":[]}],"hk4e":[{"uid":100000001,"timezone":8,"lang":"zh-cn","list":[
		{"id":1844674407370955101,"gacha_type":"400","item_id":"10000046","time":"2026-09-01 08:00:00","cookie_token":"synthetic"}]}]}`
	archives, err := a.parseGachaFile([]byte(file))
	if err != nil || len(archives) != 1 {
		t.Fatal(archives, err)
	}
	record := archives[0].Records[0]
	if archives[0].UID != "100000001" || record.ID != "1844674407370955101" || record.UIGFType != "301" || record.Name != "胡桃" || record.ItemType != "角色" || record.Rank != "5" {
		t.Fatalf("%+v", archives[0])
	}
	archive := archives[0]
	archive.Region = "cn_gf01"
	for _, legacy := range []bool{false, true} {
		raw, _ := json.Marshal(a.gachaFile(archive, legacy))
		again, err := a.parseGachaFile(raw)
		if err != nil || len(again) != 1 || again[0].Records[0] != record || again[0].UID != archive.UID {
			t.Fatalf("legacy %v: %+v %v", legacy, again, err)
		}
	}
	// Yunzai's legacy files name items only; the catalog gives their IDs.
	legacy, err := a.parseGachaFile([]byte(`{"info":{"uid":"100000001","uigf_version":"v2.2"},"list":[{"id":"1","gacha_type":"302","name":"护摩之杖","item_type":"武器","time":"2026-09-01 08:00:00"}]}`))
	if err != nil || legacy[0].Records[0].ItemID != "13501" || legacy[0].Timezone != noTimezone {
		t.Fatal(legacy, err)
	}
	if other, err := a.parseGachaFile([]byte(`{"info":{"uid":"100000001","srgf_version":"v1.0"},"list":[{"id":"1"}]}`)); err != nil || other != nil {
		t.Fatal("a Star Rail file was read", other, err)
	}
	for text, want := range map[string]string{
		`{"info":{}}`: "json文件内容错误：非统一祈愿记录标准",
		`{"info":{"uigf_version":"v2.2","uid":"1"},"list":[{"id":"1","gacha_type":"301","name":"未收录","time":"2026-09-01 08:00:00"}]}`: "json文件内容错误：缺少必要字段 item_id",
		`not json`: "{name},json格式错误",
	} {
		if _, err := a.parseGachaFile([]byte(text)); friendlyError(err) != want {
			t.Errorf("%s: %v", text, err)
		}
	}
}

func TestGachaImportsGoUnderTheUIDsRegion(t *testing.T) {
	a := pluginApp(t)
	listed := Accounts{Items: []Account{{Roles: []Role{{Game: "genshin", UID: "600000001", Region: "os_usa"}}}}}
	if _, _, err := a.Gacha.Import(gacha.Archive{UID: "500000001", Region: "cn_gf01", Timezone: 8}); err != nil {
		t.Fatal(err)
	}
	for uid, want := range map[string]string{"600000001": "os_usa", "500000001": "cn_gf01", "500000002": "cn_qd01", "1800000001": "os_asia", "100000001": "cn_gf01"} {
		if region := a.archiveRegion(listed, uid); region != want {
			t.Errorf("%s: %s", uid, region)
		}
	}
	if regionTimezone("os_euro") != 1 || regionTimezone("os_usa") != -5 || regionTimezone("cn_qd01") != 8 {
		t.Error("region time zones")
	}
}

func TestGachaImportFindsTheSentFile(t *testing.T) {
	event := &rayleabot.EventContext{}
	event.Event.Message.Segments = []rayleabot.Segment{{Type: "file", Data: map[string]any{"name": "100000001.json", "url": "https://files.example/abc"}}}
	if file, name, found := messageFile(context.Background(), event); !found || file != "https://files.example/abc" || name != "100000001.json" {
		t.Fatal(file, name, found)
	}
	event.Event.Message.Segments = nil
	event.Event.Message.PlainText = "文件在 https://files.example/get?name=%E8%AE%B0%E5%BD%95 。"
	if file, name, found := messageFile(context.Background(), event); !found || file != "https://files.example/get?name=%E8%AE%B0%E5%BD%95" || name != "记录.json" {
		t.Fatal(file, name, found)
	}
	event.Event.Message.PlainText = "导入记录"
	if _, _, found := messageFile(context.Background(), event); found {
		t.Fatal("a message without a file was taken")
	}
	var waits fileImports
	waits.wait("sender")
	if !waits.waiting("sender") || waits.waiting("other") {
		t.Fatal("waits")
	}
	waits.done("sender")
	if waits.waiting("sender") {
		t.Fatal("the wait outlived the import")
	}
}

// In a group the import command suggests a private chat and, once forced, a
// file sent is answered and the sender then asked to recall it; both mention
// the sender, as Yunzai does.
func TestGachaFileInAGroupMentionsTheSender(t *testing.T) {
	a := pluginApp(t)
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	mentions := func(frame map[string]any, text string) bool {
		var message rayleabot.MessageOut
		_ = decodeObject(asObject(frame["data"])["message"], &message)
		return len(message.Segments) == 2 && message.Segments[0].Type == "at" && asText(message.Segments[0].Data["user_id"]) == "u" && sentText(message) == "\n"+text
	}
	if end, _ := host.groupMessage("#导入记录", "导入记录"); !mentions(end, "建议私聊导入，若你确认要在此导入，请发送【#强制导入记录】") {
		t.Fatalf("suggested %+v", end)
	}
	if end, _ := host.groupMessage("#强制导入记录", "强制导入记录"); terminalText(end) != "请发送Json文件" {
		t.Fatalf("asked for the file with %+v", end)
	}
	end, _ := host.groupMessage("https://127.0.0.1:1/record.json", "")
	if len(host.sent) != 1 || sentText(host.sent[0].Message) != "下载json文件错误" || !mentions(end, "已收到文件，请撤回") {
		t.Fatalf("answered %+v, then %+v", host.sent, end)
	}
}
