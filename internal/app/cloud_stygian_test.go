package app

import (
	"reflect"
	"testing"
	"time"
)

// stygianPools are the event banners around a skipped version number: 6.7
// was followed by 7.0.
var stygianPools = []PoolInfo{
	{Version: "6.6", Half: "上半", From: "2026-05-20 06:00:00", Kind: "event"},
	{Version: "6.7", Half: "上半", From: "2026-07-01 06:00:00", Kind: "event"},
	{Version: "6.7", Half: "上半", From: "2026-07-01 06:00:00", Kind: "chronicled"},
	{Version: "6.7", Half: "下半", From: "2026-07-21 18:00:00", Kind: "event"},
	{Version: "7.0", Half: "上半", From: "2026-08-12 06:00:00", Kind: "event"},
	{Version: "7.0", Half: "下半", From: "2026-09-01 18:00:00", Kind: "event"},
}

func TestStygianSeasonFollowsArk(t *testing.T) {
	at := func(text string) time.Time {
		moment, _ := time.ParseInLocation("2006-01-02 15:04", text, chinaZone)
		return moment
	}
	cases := []struct {
		now     string
		version string
	}{
		{"2026-08-19 10:00", "7.0"},
		// Between seasons the last one still counts.
		{"2026-09-23 12:00", "7.0"},
		// ark counts a season each 42 days past the banners.
		{"2026-09-30 10:00", "7.1"},
		{"2026-11-11 10:00", "7.2"},
		// Before 7.0's season began the season was 6.7's; ark's count named
		// it 6.8.
		{"2026-08-15 12:00", "6.7"},
	}
	for _, tc := range cases {
		if got := stygianVersionName(stygianSeason(stygianPools, at(tc.now))); got != tc.version {
			t.Errorf("%s: season %s, want %s", tc.now, got, tc.version)
		}
	}
	if season := stygianSeason(nil, time.Now()); season != -1 {
		t.Fatalf("season %d without banners", season)
	}
	// ark only counts the latest version's first half.
	if season := stygianSeason(stygianPools[3:4], time.Now()); season != -1 {
		t.Fatalf("season %d without a first half", season)
	}
}

func TestStygianPeriodDatesSeasonsByBanners(t *testing.T) {
	format := func(start, end time.Time) string {
		return start.In(chinaZone).Format("2006-01-02 15:04:05") + " - " + end.In(chinaZone).Format("2006-01-02 15:04:05")
	}
	cases := map[string]string{
		"6.7": "2026-07-08 10:00:00 - 2026-08-11 03:59:59",
		// ark's count from 6.0 gives 7.0 the next season's dates.
		"7.0": "2026-08-19 10:00:00 - 2026-09-22 03:59:59",
		"7.1": "2026-09-30 10:00:00 - 2026-11-03 03:59:59",
	}
	for version, want := range cases {
		if got := format(stygianPeriod(stygianPools, stygianVersion(version))); got != want {
			t.Errorf("%s: %s, want %s", version, got, want)
		}
	}
	// Without banners, ark's own count from 6.0.
	if got := format(stygianPeriod(nil, stygianVersion("6.1"))); got != "2025-10-29 10:00:00 - 2025-12-02 03:59:59" {
		t.Fatal(got)
	}
}

func TestStygianRowMergesSourcesAsArk(t *testing.T) {
	entry := StygianEntry{UID: "100000001", Score: 27}
	row, ok := stygianRow(entry, &arkStygianPlace{Rank: "12", Sum: "113428", Index: 6, Seconds: 27}, &akashaStygianPlace{Index: "1", Difficulty: 6, Seconds: 30})
	if !ok || row.Index != 6 || row.Seconds != 27 || row.Ark != "12 / 113428" || row.Akasha != "1" {
		t.Fatalf("row %+v", row)
	}
	// ark takes the lower difficulty and the fewer seconds, each on its own.
	entry = StygianEntry{UID: "100000002", Score: 100 + 2*2048}
	row, ok = stygianRow(entry, nil, &akashaStygianPlace{Index: "900", Difficulty: 5, Seconds: 90})
	if !ok || row.Index != 4 || row.Seconds != 90 || row.Ark != "? / ?" {
		t.Fatalf("row %+v", row)
	}
	// An answer without the UID leaves the group's own result.
	row, ok = stygianRow(StygianEntry{UID: "100000003", Score: 150 + 2048}, &arkStygianPlace{}, nil)
	if !ok || row.Index != 5 || row.Seconds != 150 || row.Akasha != "?" {
		t.Fatalf("row %+v", row)
	}
	// Whole multiples of 2048 carry no seconds, and nothing else knows them.
	if _, ok = stygianRow(StygianEntry{UID: "100000004", Score: 2048}, nil, nil); ok {
		t.Fatal("row without seconds")
	}
}

func TestStygianEntriesMergeOlderSeason(t *testing.T) {
	data := GroupData{Stygian: map[string][]StygianEntry{
		"63": {{UID: "3", Score: 40}, {UID: "1", Score: 30}, {UID: "2", Score: 30}},
		"61": {{UID: "1", Score: 10}, {UID: "4", Score: 5}},
	}}
	uids := func(entries []StygianEntry) []string {
		list := []string{}
		for _, entry := range entries {
			list = append(list, entry.UID)
		}
		return list
	}
	if got := uids(stygianEntries(data, 63, -1)); !reflect.DeepEqual(got, []string{"1", "2", "3"}) {
		t.Fatal(got)
	}
	// The current season comes first; the older one adds its other UIDs.
	merged := stygianEntries(data, 63, 61)
	if got := uids(merged); !reflect.DeepEqual(got, []string{"1", "2", "3", "4"}) || merged[0].Score != 30 {
		t.Fatal(got, merged)
	}
}
