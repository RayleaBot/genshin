package images

import (
	"testing"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// The expected words are what moment's zh-cn humanize() prints.
func TestHumanizeFollowsMoment(t *testing.T) {
	for span, want := range map[time.Duration]string{
		30 * time.Second: "几秒", 80 * time.Second: "1 分钟", 20 * time.Minute: "20 分钟", 50 * time.Minute: "1 小时",
		5 * time.Hour: "5 小时", 23 * time.Hour: "1 天", 3 * 24 * time.Hour: "3 天", 27 * 24 * time.Hour: "1 个月",
		70 * 24 * time.Hour: "2 个月", 400 * 24 * time.Hour: "1 年",
	} {
		if got := humanize(span); got != want {
			t.Errorf("humanize(%v) = %q, want %q", span, got, want)
		}
	}
}

func TestCalendarTimesFollowMiao(t *testing.T) {
	contents := []any{
		map[string]any{"ann_id": "1", "title": "「空月之歌」版本更新维护预告", "content": "〓更新时间〓<p>2026/09/10 06:00:00开始</p>"},
		map[string]any{"ann_id": "2", "title": "限时活动", "content": "<p>〓活动时间〓</p><p>2026/09/12 10:00:00 ~ 2026/09/30 03:59:59</p>"},
		map[string]any{"ann_id": "3", "title": "新活动", "content": "〓活动时间〓「空月之歌」版本更新后永久开放"},
		map[string]any{"ann_id": "4", "title": "持续活动", "content": "〓活动时间〓「空月之歌」版本期间持续开放"},
	}
	times := calendarTimes(contents)
	if times["2"] != [2]string{"2026-09-12 10:00:00", "2026-09-30 03:59:59"} {
		t.Errorf("event = %v", times["2"])
	}
	// A version update's 06:00 reads as 11:00; a version lasts six weeks to 06:00.
	if times["4"] != [2]string{"2026-09-10 11:00:00", "2026-10-22 06:00:00"} {
		t.Errorf("version period = %v", times["4"])
	}
	// Upstream only takes a numbered version for 版本更新后, so a named one leaves it unset.
	if _, ok := times["3"]; ok {
		t.Errorf("named version after update = %v", times["3"])
	}
}

func TestCalendarTalentsOfTheDay(t *testing.T) {
	catalog := gamekit.Catalog{Entries: []gamekit.Entry{
		{ID: "10000014", Name: "芭芭拉", Kind: "character", Rarity: 4, Materials: map[string]string{"天赋材料": "「自由」的哲学", "周本材料": "北风之环"}},
		{ID: "10000030", Name: "钟离", Kind: "character", Rarity: 5, Materials: map[string]string{"天赋材料": "「黄金」的哲学", "周本材料": "北风之环"}},
		{ID: "10000031", Name: "菲谢尔", Kind: "character", Rarity: 4, Materials: map[string]string{"天赋材料": "「自由」的哲学", "周本材料": "北风之环"}},
		{ID: "10000041", Name: "莫娜", Kind: "character", Rarity: 5, Materials: map[string]string{"天赋材料": "「自由」的哲学", "周本材料": "北风之环"}},
	}}
	// Monday 03:00 still counts as Sunday, when every book is farmable.
	sunday := calendarTalents(gamekit.ImageContext{Catalog: catalog}, &gamekit.ImageResources{}, time.Date(2026, 9, 21, 3, 0, 0, 0, chinaTime))
	monday := calendarTalents(gamekit.ImageContext{Catalog: catalog}, &gamekit.ImageResources{}, time.Date(2026, 9, 21, 12, 0, 0, 0, chinaTime))
	if len(sunday) != 23 || len(monday) != 8 {
		t.Fatalf("books: sunday %d, monday %d", len(sunday), len(monday))
	}
	freedom := monday[0].(map[string]any)
	chars := freedom["chars"].([]any)
	// Rarest first, then newest.
	if freedom["label"] != "蒙德·自由" || len(chars) != 3 || chars[0].(map[string]any)["star"] != 5 || chars[1].(map[string]any)["position"] != "" || chars[2].(map[string]any)["position"] != "last" {
		t.Errorf("freedom = %v", freedom)
	}
}

func TestTalentBooksFollowMiaoDaily(t *testing.T) {
	for book, want := range map[string][2]any{"「自由」的哲学": {1, "蒙德·自由"}, "「黄金」的哲学": {3, "璃月·黄金"}, "「荣光」的哲学": {3, "至冬·荣光"}, "「新书」的哲学": {0, "新书"}} {
		week, label := talentBook(book)
		if week != want[0] || label != want[1] {
			t.Errorf("talentBook(%s) = %d %s", book, week, label)
		}
	}
}
