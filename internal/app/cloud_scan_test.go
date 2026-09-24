package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestScanGearReadsSlotsOfAReforge(t *testing.T) {
	a := App{Game: testGame(t)}
	a.Cloud.HTTP = cloudDoer(func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		if req.URL.String() != "https://ark.ivny.cn/ocr/profilechange/gs" || json.NewDecoder(req.Body).Decode(&body) != nil || body["forge"] != true {
			t.Fatal("unexpected OCR request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":100,"data":[{"type":"arti1","data":{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204]}},{"type":"arti1","data":{"star":5,"level":20,"mainId":13001,"attrIds":[501204,501224]}}]}`))}, nil
	})
	gears, err := a.scanGear(t.Context(), "https://multimedia.nt.qq.com.cn/download?rkey=1", true)
	if err != nil || len(gears) != 2 || gears[0].Slot != 1 || gears[1].Slot != 1 {
		t.Fatal(gears, err)
	}
	if _, err = a.scanGear(t.Context(), "http://gchat.qpic.cn/a.png", true); err == nil {
		t.Fatal("plain HTTP image accepted")
	}
}

func TestReforgeMatchesKeptPieceBySubstats(t *testing.T) {
	game := testGame(t)
	// Without a slot in the answer, the piece's name gives it.
	read := readGear(game, scannedGear{Gear: cloudObject(t, `{"name":"角斗士的留恋","star":5,"level":20,"mainId":13001,"attrIds":[501204,501204]}`)})
	if read == nil || read.Slot != 1 {
		t.Fatal(read)
	}
	kept := PanelEquipment{Slot: 1, Sub: []PanelStat{{Key: "cpct", Value: "7.7%"}}}
	if !sameGear(kept, *read) {
		t.Fatal("the game's shown value did not match")
	}
	for _, other := range []PanelStat{{Key: "cpct", Value: "7.0%"}, {Key: "cdmg", Value: "7.7%"}} {
		if sameGear(PanelEquipment{Slot: 1, Sub: []PanelStat{other}}, *read) {
			t.Fatal("another piece matched", other)
		}
	}
}

// The management page lists each distribution ark answers with by ark's
// ranking percentiles.
func TestDistributionListsArkPercentileScores(t *testing.T) {
	data := cloudList(t, `[{"retcode":100,"data":{"scores":["268.70","255.60","248.70","240.40","233.90","223.00","210.80","187.10","170.00","104.90"],"total":60184,"top1":"294.00"}},
		{"retcode":100,"data":{"scores":["68249.76","56693.54","52188.35","48193.57","45706.58","41279.80","35789.10","23696.87","15476.92","4425.55"],"total":60391,"top1":"86620.75"}}]`)
	result, err := projectCloud(testGame(t), CloudInput{Mode: "distribution", CharacterID: "10000046"}, data)
	if err != nil {
		t.Fatal(err)
	}
	if text := result.View.Text(); !strings.Contains(text, "TOP 1%：268.7") || !strings.Contains(text, "TOP 99%：4425.55") || !strings.Contains(text, "收录总量：60391") {
		t.Fatal(text)
	}
}

// The host hands a quoted message's ID over as message_id, OneBot's id
// renamed.
func TestQuotedMessageReadsTheHostReplySegment(t *testing.T) {
	event := &rayleabot.EventContext{Event: rayleabot.Event{Message: rayleabot.Message{Segments: []rayleabot.Segment{
		{Type: "reply", Data: map[string]any{"message_id": "98765"}}, rayleabot.Text("ark获取面板1"),
	}}}}
	if got := quotedMessage(event); got != "98765" {
		t.Fatalf("quoted %q", got)
	}
	event.Event.Message.Segments = event.Event.Message.Segments[1:]
	if got := quotedMessage(event); got != "" {
		t.Fatalf("quoted %q without a reply", got)
	}
}
