package images

import (
	"strconv"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/gacha"
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
	image, ok := Gacha(gamekit.ImageContext{}, gamekit.GachaImage{UID: "100000001", Word: "角色记录", Archive: gacha.Archive{Records: records}})
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
	if _, ok := Gacha(gamekit.ImageContext{}, gamekit.GachaImage{Word: "武器记录", Archive: gacha.Archive{Records: records}}); ok {
		t.Error("an empty pool should keep the summary card")
	}
}

func TestGachaRateUpPeriods(t *testing.T) {
	inside := gacha.Record{Name: "刻晴", Time: "2021-02-20 12:00:00"}
	outside := gacha.Record{Name: "刻晴", Time: "2021-05-01 12:00:00"}
	if !rateUp(inside) || rateUp(outside) || rateUp(gacha.Record{Name: "七七"}) || !rateUp(gacha.Record{Name: "芙宁娜"}) {
		t.Error("rate-up periods do not follow upstream")
	}
}
