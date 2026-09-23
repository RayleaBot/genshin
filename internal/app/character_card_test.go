package app

import "testing"

func TestCardQueryFollowsMiaoCheck(t *testing.T) {
	for text, want := range map[string][2]string{
		"雷神":            {"雷神", ""},
		"雷神卡片100000001": {"雷神", "100000001"},
		"老婆胡桃":          {"胡桃", ""},
		"胡桃 187654321":  {"胡桃", "187654321"},
		"胡桃1876543210":  {"胡桃", "1876543210"},
	} {
		if name, uid := cardQuery(text); name != want[0] || uid != want[1] {
			t.Errorf("cardQuery(%q) = %q, %q; want %q, %q", text, name, uid, want[0], want[1])
		}
	}
}
