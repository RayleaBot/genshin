package images

import (
	"os"
	"path/filepath"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/artwork"
)

func TestCharacterTalentReadsMiaoData(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "miao-plugin", "resources", "meta-gs", "character", "胡桃", "data.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	values := `["1","2","3","4","5","6","7","8","9","10","11","12","13","14","15"]`
	data := `{"name":"胡桃","title":"雪霁梅香","desc":"往生堂<b>堂主</b><script>x</script>","elem":"pyro","weapon":"polearm",
		"baseAttr":{"hp":16657.69,"atk":130,"def":938.42},"growAttr":{"key":"cdmg","value":38.4},"talentCons":{"e":3,"q":5},
		"talent":{"a":{"name":"往生秘传枪法","desc":["<h3>普通攻击</h3>","进行至多六段的连续枪击。","<h3>重击</h3>","消耗体力。"],
			"tables":[{"name":"一段伤害","isSame":false,"values":` + values + `},{"name":"重击体力消耗","isSame":true,"values":["25"]}]},
			"e":{"name":"蝶引来生","desc":["e"]},"q":{"name":"安神秘法","desc":["q"]}},
		"cons":{"1":{"name":"赤团开时斜飞去","desc":["<span style=\"color:#FFD780FF;\">蝶引来生</span>"]},"2":{"name":"二","desc":["2"]}},
		"passive":[{"name":"蝶隐之时","desc":["p"]}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	context := gamekit.ImageContext{Artwork: &artwork.Store{Root: root}}
	entry := gamekit.Entry{ID: "10000046", Name: "胡桃", Kind: "character"}
	image, ok := Entry(context, gamekit.EntryImage{Command: "talent-wiki", Word: "胡桃天赋", Entry: entry})
	if !ok || image.Data["name"] != "雪霁梅香·胡桃" || image.Data["desc"] != "往生堂<b>堂主</b>x" {
		t.Fatalf("image = %v", image.Data)
	}
	line := image.Data["line"].([]any)
	if line[0].(map[string]any)["num"] != "16,657.7" || line[1].(map[string]any)["num"] != "130.0" || line[3].(map[string]any)["label"] != "成长·爆伤" {
		t.Errorf("line = %v", line)
	}
	attack := image.Data["talents"].([]any)[0].(map[string]any)["parts"].([]any)[0].(map[string]any)
	row := attack["rows"].([]any)[0].(map[string]any)["values"].([]string)
	// A heading line is not followed by a break; the table shows Lv6 to Lv13.
	if attack["desc"] != "<h3>普通攻击</h3>进行至多六段的连续枪击。<br><h3>重击</h3>消耗体力。" || len(row) != 8 || row[0] != "6" || row[7] != "13" || len(attack["shared"].([]any)) != 1 {
		t.Errorf("attack = %v", attack)
	}
	image, _ = Entry(context, gamekit.EntryImage{Command: "talent-wiki", Word: "胡桃命座", Entry: entry})
	constellations := image.Data["constellations"].([]any)
	if len(constellations) != 2 || constellations[0].(map[string]any)["desc"] != `<span style="color:#FFD780FF">蝶引来生</span>` {
		t.Errorf("constellations = %v", constellations)
	}
}

// JavaScript's toFixed rounds an exact tie up, where Go's formatting rounds
// it to even.
func TestMiaoCommaRoundsLikeJavaScript(t *testing.T) {
	if got := miaoComma(942.25, 1); got != "942.3" {
		t.Errorf("miaoComma(942.25) = %s", got)
	}
}

// miao breaks a long description at a dash, else between its clauses; the
// expected lines are miao's getDesc output for the same text.
func TestWikiDescBreaksLikeMiao(t *testing.T) {
	for desc, want := range map[string]string{
		"「往生堂」七十七代堂主，年纪轻轻就已主掌璃月的葬仪事务。": "「往生堂」七十七代堂主</br>年纪轻轻就已主掌璃月的葬仪事务",
		"短描述。": "短描述",
		"璃月的「七星」之一，玉衡星。对她而言，「规则」是用来打破的，而「万全的准备」是用来颠覆的——相信自己，远比相信神明更重要。": "璃月的「七星」之一，玉衡星，对她而言</br>「规则」是用来打破的，而「万全的准备」是用来颠覆的",
	} {
		if got := wikiDesc(desc); got != want {
			t.Errorf("wikiDesc(%s) = %q, want %q", desc, got, want)
		}
	}
}
