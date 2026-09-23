package images

import (
	"strconv"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

func TestGachaFollowsYunzaiAnalysis(t *testing.T) {
	// Oldest first: Mona (standard) on pull 70, Hu Tao on pull 20 after her,
	// then ten pulls without a five-star.
	records := []gacha.Record{}
	add := func(count int, name, rank, itemType, when string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(1000 + len(records)), GachaType: "400", Name: name, Rank: rank, ItemType: itemType, Time: when})
		}
	}
	add(69, "黎明神剑", "3", "武器", "2023-01-01 10:00:00")
	add(1, "莫娜", "5", "角色", "2023-01-01 10:00:00")
	add(19, "黎明神剑", "3", "武器", "2023-02-01 10:00:00")
	add(1, "胡桃", "5", "角色", "2023-02-01 10:00:00")
	add(10, "黎明神剑", "3", "武器", "2023-03-01 10:00:00")
	image, ok := Gacha(app.ImageContext{}, app.GachaImage{UID: "100000001", Word: "角色记录", Archive: gacha.Archive{Records: records}})
	if !ok || image.Data["pool"] != "301" || image.Data["all"] != 100 {
		t.Fatalf("image = %+v", image.Data)
	}
	want := map[string]string{"未出五星": "10", "五星": "2", "五星平均": "45", "小保底不歪": "0.0%", "最非": "70", "五星常驻": "1", "UP平均": "90", "UP花费原石": "1.44w", "最欧": "20"}
	for _, line := range image.Data["lines"].([][]any) {
		for _, raw := range line {
			cell := raw.(map[string]any)
			if expected, ok := want[cell["label"].(string)]; ok && cell["num"] != expected {
				t.Errorf("%s = %v, want %s", cell["label"], cell["num"], expected)
			}
		}
	}
	cards := image.Data["cards"].([]any)
	huTao, mona := cards[0].(map[string]any), cards[1].(map[string]any)
	if huTao["num"] != 20 || huTao["up"] != true || huTao["class"] != "good" || mona["num"] != 70 || mona["up"] != false || mona["class"] != "normal" {
		t.Errorf("cards = %v", cards)
	}
	// A pool without records draws with every count at zero, as upstream.
	if empty, ok := Gacha(app.ImageContext{}, app.GachaImage{Word: "武器记录", Archive: gacha.Archive{Records: records}}); !ok || empty.Data["all"] != 0 || len(empty.Data["cards"].([]any)) != 0 {
		t.Errorf("empty pool = %v %v", empty.Data, ok)
	}
}

func TestGachaCountsTheFirstFourStarAndDropsUnknownFiveStars(t *testing.T) {
	// Newest first: a four-star, three pulls, a four-star, nine pulls, an
	// unnamed five-star, nineteen pulls and a named one.
	records := []gacha.Record{}
	add := func(count int, name, rank string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(2000 - len(records)), GachaType: "200", Name: name, Rank: rank, ItemType: "角色", Time: "2024-05-01 10:00:00"})
		}
	}
	add(1, "菲谢尔", "4")
	add(3, "黎明神剑", "3")
	add(1, "菲谢尔", "4")
	add(9, "黎明神剑", "3")
	add(1, "未知", "5")
	add(19, "黎明神剑", "3")
	add(1, "莫娜", "5")
	image, _ := Gacha(app.ImageContext{}, app.GachaImage{Word: "常驻记录", Archive: gacha.Archive{Records: records}})
	got := map[string]string{}
	for _, line := range image.Data["lines"].([][]any) {
		for _, raw := range line {
			cell := raw.(map[string]any)
			got[cell["label"].(string)] = cell["num"].(string)
		}
	}
	// The newest pull was a four-star, so none since; the unnamed five-star
	// and the twenty pulls it took are left out.
	if got["未出四星"] != "0" || got["五星"] != "1" || image.Data["all"] != 15 || len(image.Data["cards"].([]any)) != 1 {
		t.Fatalf("lines = %v, all = %v", got, image.Data["all"])
	}
}

func TestGachaRateUpPeriods(t *testing.T) {
	inside := gacha.Record{Name: "刻晴", Time: "2021-02-20 12:00:00"}
	outside := gacha.Record{Name: "刻晴", Time: "2021-05-01 12:00:00"}
	if !rateUp(inside) || rateUp(outside) || rateUp(gacha.Record{Name: "七七"}) || !rateUp(gacha.Record{Name: "芙宁娜"}) {
		t.Error("rate-up periods do not follow upstream")
	}
}

func TestGachaAllFollowsYunzai(t *testing.T) {
	records := []gacha.Record{}
	add := func(count int, pool, name, rank, itemType string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(1000 + len(records)), GachaType: pool, Name: name, Rank: rank, ItemType: itemType, Time: "2024-05-01 10:00:00"})
		}
	}
	// Two five-stars in the character pool, one in the weapon pool, none elsewhere.
	add(30, "301", "黎明神剑", "3", "武器")
	add(1, "301", "胡桃", "5", "角色")
	add(20, "301", "黎明神剑", "3", "武器")
	add(1, "301", "莫娜", "5", "角色")
	add(40, "302", "黎明神剑", "3", "武器")
	add(1, "302", "护摩之杖", "5", "武器")
	image, ok := Gacha(app.ImageContext{}, app.GachaImage{UID: "100000001", Word: "全部记录", Archive: gacha.Archive{Records: records}})
	if !ok || image.Template != "gacha-all" || image.Data["pool"] != "301" {
		t.Fatalf("image = %v", image.Data)
	}
	logs := image.Data["logs"].([]any)
	if len(logs) != 2 {
		t.Fatalf("logs = %d", len(logs))
	}
	// Every block's five-star row is padded to the longest.
	weapon := logs[1].(map[string]any)["cards"].([]any)
	if weapon[0].(map[string]any)["null"] != nil || weapon[1].(map[string]any)["null"] != true {
		t.Errorf("weapon cards = %v", weapon)
	}
}
