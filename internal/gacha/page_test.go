package gacha

import (
	"errors"
	"testing"
)

func pageRecord(uid, pool, id string) map[string]any {
	return map[string]any{"uid": uid, "gacha_type": pool, "gacha_id": "9001", "id": id, "item_id": "1001", "time": "2026-09-01 08:00:00", "rank_type": "4", "count": "1"}
}

func TestParsePageConvertsPoolsAndTimezones(t *testing.T) {
	for _, tc := range []struct {
		game, region, pool, actual, want string
		data                             map[string]any
		zone                             int
	}{
		{"genshin", "cn_gf01", "301", "400", "400", map[string]any{}, 8},
		// Overseas regions fall back to their server offset.
		{"genshin", "os_usa", "200", "200", "200", map[string]any{}, -5},
	} {
		data := tc.data
		data["list"] = []any{pageRecord("100000001", tc.actual, "1844674407370955101")}
		page, err := ParsePage("100000001", tc.region, tc.pool, "", data)
		if err != nil || len(page.Records) != 1 || page.Timezone != tc.zone || page.More || page.NextID != "1844674407370955101" {
			t.Fatalf("%s %s: %+v %v", tc.game, tc.region, page, err)
		}
		if page.Records[0].GachaType != tc.want {
			t.Fatalf("%s pool %s became %s", tc.game, tc.actual, page.Records[0].GachaType)
		}
		if tc.game == "genshin" && page.Records[0].UIGFType != map[string]string{"400": "301", "200": "200"}[tc.actual] {
			t.Fatalf("UIGF pool %q", page.Records[0].UIGFType)
		}
	}
}

func TestParsePageRejectsRecordsOutsideTheRequest(t *testing.T) {
	mutations := map[string]func(map[string]any){
		"other uid":       func(r map[string]any) { r["uid"] = "other" },
		"repeated cursor": func(r map[string]any) { r["id"] = "100" },
		"other pool":      func(r map[string]any) { r["gacha_type"] = "200" },
		"bad time":        func(r map[string]any) { r["time"] = "yesterday" },
		"bad item":        func(r map[string]any) { r["item_id"] = "x" },
	}
	for name, mutate := range mutations {
		record := pageRecord("100000001", "100", "99")
		mutate(record)
		_, err := ParsePage("100000001", "cn_gf01", "100", "100", map[string]any{"list": []any{record}})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s accepted: %v", name, err)
		}
	}
	list := []any{}
	for range 21 {
		list = append(list, pageRecord("100000001", "100", "99"))
	}
	if _, err := ParsePage("100000001", "cn_gf01", "100", "", map[string]any{"list": list}); !errors.Is(err, ErrInvalid) {
		t.Fatal("oversized page accepted")
	}
	if _, err := ParsePage("100000001", "cn_gf01", "100", "", map[string]any{"list": []any{}, "region": "cn_qd01"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("page of another region accepted")
	}
}
