package app

import (
	"reflect"
	"testing"
)

func TestParseCustomRankFollowsArk(t *testing.T) {
	// A super administrator with the token: every column, by Chinese name,
	// the words around the filters joined into the name, and a row count.
	name, query, err := parseCustomRank("雷电 等级>=90 将军 精炼!=1 数量:10", customRankColumns, "dmg_avg", "")
	want := []customRankFilter{{Type: "level", Rule: 0, Value: 90.0, RawValue: "90"}, {Type: "weapon_affix", Rule: 5, Value: 1.0, RawValue: "1"}}
	if err != nil || name != "雷电将军" || query.Nums != 10 || query.Rank.Col != "dmg_avg" || query.Rank.Order != "desc" || !reflect.DeepEqual(query.Filter, want) {
		t.Fatalf("parsed %q %+v %v", name, query, err)
	}
	// Sets may be named, slots keep their stat, counts stay within 1–50.
	_, query, _ = parseCustomRank("胡桃 套装=魔女4 部位3=暴击 NUMS：0", customRankColumns, "mark_score", "")
	want = []customRankFilter{{Type: "artis_sets", Rule: 1, Value: "魔女4", RawValue: "魔女4"}, {Type: "pos3", Rule: 1, Value: "暴击", RawValue: "暴击"}}
	if query.Nums != 1 || query.Rank.Col != "mark_score" || !reflect.DeepEqual(query.Filter, want) {
		t.Fatalf("parsed %+v", query)
	}
	for raw, nums := range map[string]int{"胡桃": 20, "胡桃 数量=70": 50, "胡桃 nums:99999999999999999999": 50} {
		if _, query, _ := parseCustomRank(raw, customRankColumns, "dmg_avg", ""); query.Nums != nums {
			t.Errorf("%s asks for %d rows", raw, query.Nums)
		}
	}
	cases := []struct {
		raw     string
		columns []string
		denied  string
		want    string
	}{
		{"胡桃 升序", customRankColumns, "", "当前命令仅支持默认降序排序"},
		{"胡桃 排序:dmg", customRankColumns, "", "当前命令仅支持默认降序排序"},
		{"胡桃 等级=九十", customRankColumns, "", "等级 的值必须是数字"},
		{"胡桃 foo=1", customRankColumns, "", "不支持筛选列 foo"},
		// Without the token only the constellation filters, whoever asks; the
		// value is checked before the column, as upstream.
		{"胡桃 等级>=90", []string{"cons"}, "除 cons 之外的筛选仅主人可用", "除 cons 之外的筛选仅主人可用"},
		{"胡桃 等级>=abc", []string{"cons"}, "除 cons 之外的筛选仅主人可用", "等级 的值必须是数字"},
	}
	for _, tc := range cases {
		if _, _, err := parseCustomRank(tc.raw, tc.columns, "dmg_avg", tc.denied); err == nil || err.Error() != tc.want {
			t.Errorf("%s: %v, want %s", tc.raw, err, tc.want)
		}
	}
	if name, query, err := parseCustomRank("胡桃 命座<=2 cons>0", []string{"cons"}, "dmg_avg", "除 cons 之外的筛选仅主人可用"); err != nil || name != "胡桃" || len(query.Filter) != 2 {
		t.Fatalf("constellation filters refused: %v %+v", err, query)
	}
}

func TestCustomRankLimitWritesArkNote(t *testing.T) {
	_, query, _ := parseCustomRank("胡桃", customRankColumns, "dmg_avg", "")
	if got := customRankLimit(query); got != "排序: 伤害均值 降序 / 筛选: 无" {
		t.Fatal(got)
	}
	// ark notes the resolved column with the value as written.
	_, query, _ = parseCustomRank("胡桃 命=6 部位3=暴击", customRankColumns, "mark_score", "")
	if got := customRankLimit(query); got != "排序: 圣遗物评分 降序 / 筛选: cons=6 部位3=暴击" {
		t.Fatal(got)
	}
}

func TestCustomRankEntriesAddConstellationTalents(t *testing.T) {
	record, err := findReferenceCharacter(calcEngine(t), CharacterPanel{ID: "10000046", Element: "pyro"})
	if err != nil {
		t.Fatal(err)
	}
	rows := []any{
		map[string]any{"uid": "10****543", "cons": 5, "talent_a": 10, "talent_e": 10, "talent_q": 9, "weapon_name": "护摩之杖", "weapon_level": 90, "weapon_affix": 1,
			"mark_score": 281.6, "dmg_avg": 64248.5, "artisSet": map[string]any{"names": []any{"炽烈的炎之魔女"}, "name": "炽烈的炎之魔女4"}},
		map[string]any{"uid": "15****910"},
	}
	entries := customRankEntries(record, rows)
	first, second := entries[0], entries[1]
	if first.UID != "10****543" || first.Level != 90 || first.Talents["e"] != (PanelTalent{Level: 13, Original: 10}) || first.Talents["q"] != (PanelTalent{Level: 12, Original: 9}) || first.Talents["a"] != (PanelTalent{Level: 10, Original: 10}) {
		t.Fatalf("first row %+v", first)
	}
	if first.Damage == nil || *first.Damage != 64248.5 || first.SetName != "炽烈的炎之魔女4" || !reflect.DeepEqual(first.Sets, []string{"炽烈的炎之魔女"}) {
		t.Fatalf("first row %+v", first)
	}
	// ark's defaults: level 90, no damage without dmg_avg.
	if second.Level != 90 || second.Damage != nil || second.Mark != 0 {
		t.Fatalf("second row %+v", second)
	}
}

func TestCustomRankAvatarReadsArkPlayerData(t *testing.T) {
	answer := cloudObject(t, `{"retcode":0,"data":{"info":{"uid":"10@@@@543","avatars":{"10000046":{"name":"胡桃","id":10000046,"elem":"pyro","level":90,"promote":6,"cons":1,
		"talent":{"a":10,"e":10,"q":10},"weapon":{"name":"护摩之杖","level":90,"promote":6,"affix":4},
		"artis":{"1":{"level":20,"name":"魔女的炎之花","star":5,"mainId":14001,"attrIds":[501033,501201,501223,501244]}},"_source":"enka"}}}}}`)
	uid, avatar, failure := customRankAvatar(answer, "10000046")
	if failure != "" || uid != "10@@@@543" || avatar["name"] != "胡桃" {
		t.Fatalf("read %q %v %q", uid, avatar, failure)
	}
	// A character the answer does not hold falls back to its first one.
	if _, other, _ := customRankAvatar(answer, "10000002"); other["name"] != "胡桃" {
		t.Fatalf("fallback %v", other)
	}
	_, raw, err := cleanCloudAvatar(avatar)
	if err != nil {
		t.Fatal(err)
	}
	panel, err := pluginApp(t).cloudAvatarPanel(t.Context(), raw)
	if err != nil || panel.ID != "10000046" || panel.Rank != 1 || panel.Weapon == nil || panel.Weapon.Name != "护摩之杖" || panel.Weapon.Refinement != 4 || len(panel.Equipment) != 1 {
		t.Fatalf("panel %+v %v", panel, err)
	}
	for raw, want := range map[string]string{
		`{"retcode":-1}`: "未知错误",
		`{"retcode":1,"message":"query expired"}`:         "query expired",
		`{"retcode":0,"data":{"playerData":{"uid":"1"}}}`: "返回数据异常",
		`{"retcode":0,"data":{"uid":"1","avatars":{}}}`:   "未找到对应角色面板",
		`{"retcode":0,"data":{"avatars":{"1":{"id":1}}}}`: "返回数据异常",
	} {
		if _, _, failure := customRankAvatar(cloudObject(t, raw), "10000046"); failure != want {
			t.Errorf("%s: %q, want %q", raw, failure, want)
		}
	}
}
