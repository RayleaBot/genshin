package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChallengeScoresSeparatePeriodsAndMissingMetrics(t *testing.T) {
	fixtures := []struct {
		game, kind, raw, key string
		value                float64
		period               int
	}{
		{"genshin", "abyss", `{"schedule_id":12,"max_floor":"12-3","floors":[{"index":11,"star":9},{"index":12,"star":7}],"total_battle_times":20}`, "star", 7, 1},
		{"genshin", "theater", `{"data":[{"schedule":{"schedule_id":11},"stat":{"difficulty_id":4,"get_medal_round_list":[1,0,1]},"detail":{"rounds_data":[{},{}]}}]}`, "medal", 2, 1},
		{"genshin", "hard_single", `{"data":[{"schedule":{"schedule_id":11},"single":{"best":{"difficulty":6,"second":132}}}]}`, "time", 132, 1},
		{"genshin", "hard_mp", `{"data":[{"schedule":{"schedule_id":11},"mp":{"difficulty":5,"second":100}}]}`, "difficulty", 5, 1},
	}
	for _, f := range fixtures {
		t.Run(f.game+"."+f.kind, func(t *testing.T) {
			var data map[string]any
			json.Unmarshal([]byte(f.raw), &data)
			kind, _ := challengeKind(f.kind)
			r, e := challengeExtract(kind, data, f.period, "cn_gf01")
			if e != nil || r.Metrics[f.key] != f.value {
				t.Fatalf("%+v %v", r, e)
			}
			if f.kind == "peak" && r.Season != "12" {
				t.Fatal("previous period mixed", r)
			}
			if _, e = challengeExtract(kind, map[string]any{}, 1, "cn_gf01"); e == nil {
				t.Fatal("missing data accepted")
			}
		})
	}
	kind, _ := challengeKind("abyss")
	a := ChallengeEntry{Metrics: map[string]float64{"floor": 12, "star": 9, "battle": 12}}
	b := ChallengeEntry{Metrics: map[string]float64{"floor": 12, "star": 9}}
	if challengeCompare(kind, "", a, b) >= 0 {
		t.Fatal("missing battle count ranked as zero")
	}
	b.Metrics["battle"] = 10
	if challengeCompare(kind, "", a, b) <= 0 {
		t.Fatal("fewer battles must win")
	}
}

func TestChallengeKindsTakeRecordNames(t *testing.T) {
	for word, want := range map[string]string{
		"深境螺旋": "abyss", "深渊": "abyss", "幻想真境剧诗": "theater",
		"幽境危战": "hard_single", "幽境危战单人": "hard_single",
		"幽境危战多人": "hard_mp", "多人幽境危战": "hard_mp", "hard_mp": "hard_mp",
	} {
		if kind, ok := challengeKind(word); !ok || kind.ID != want {
			t.Errorf("%s: %+v, %v", word, kind, ok)
		}
	}
}

func TestChallengePeriodsFollowOfficialTimes(t *testing.T) {
	clock := func(month, day int) map[string]any {
		return map[string]any{"year": 2026, "month": month, "day": day, "hour": 4, "minute": 0}
	}
	china := time.FixedZone("", 8*3600)
	begin := time.Date(2026, 10, 1, 4, 0, 0, 0, china).Unix()
	end := time.Date(2026, 11, 1, 4, 0, 0, 0, china).Unix()
	previous := map[string]any{"schedule": map[string]any{"schedule_id": 1, "start_date_time": clock(9, 1), "end_date_time": clock(10, 1)}, "stat": map[string]any{"difficulty_id": 4}}
	current := map[string]any{"schedule": map[string]any{"schedule_id": 2, "start_date_time": clock(10, 1), "end_date_time": clock(11, 1)}, "stat": map[string]any{"difficulty_id": 4}}
	for _, tc := range []struct {
		kind, region string
		period       int
		data         map[string]any
		begin, end   int64
	}{
		{"abyss", "cn_gf01", 1, map[string]any{"schedule_id": 1, "max_floor": "12-3", "start_time": fmt.Sprint(begin), "end_time": fmt.Sprint(end)}, begin, end},
		{"theater", "cn_gf01", 1, map[string]any{"data": []any{current, previous}}, begin, end},
		{"theater", "cn_gf01", 2, map[string]any{"data": []any{current, previous}}, time.Date(2026, 9, 1, 4, 0, 0, 0, china).Unix(), begin},
		{"theater", "os_usa", 1, map[string]any{"data": []any{current}}, begin + 13*3600, end + 13*3600},
		{"hard_single", "cn_gf01", 1, map[string]any{"data": []any{map[string]any{"schedule": map[string]any{"schedule_id": 1, "start_time": begin, "end_time": end}, "single": map[string]any{"difficulty": 6}}}}, begin, end},
		{"hard_mp", "cn_gf01", 1, map[string]any{"data": []any{map[string]any{"schedule": map[string]any{"schedule_id": 1, "start_time": begin, "end_time": end}, "mp": map[string]any{"difficulty": 6}}}}, begin, end},
	} {
		kind, _ := challengeKind(tc.kind)
		entry, err := challengeExtract(kind, tc.data, tc.period, tc.region)
		if err != nil || entry.Begin != tc.begin || entry.End != tc.end {
			t.Errorf("%s %s period %d: %+v, %v", tc.kind, tc.region, tc.period, entry, err)
		}
	}
	entries := []ChallengeEntry{
		{Region: "cn_gf01", Season: "A", Begin: begin, End: end},
		{Region: "cn_gf01", Season: "B", Begin: begin - 1000, End: begin},
		{Region: "cn_gf01", Season: "C", Begin: begin - 2000, End: begin - 1000},
		{Region: "cn_gf01", Season: "X"},
		{Region: "os_usa", Season: "USA", Begin: begin + 13*3600, End: end + 13*3600},
	}
	for period, want := range map[int]string{1: "A", 2: "B"} {
		got := challengePeriodSeasons(entries, period, time.Unix(begin, 0))
		if len(got) != 1 || got["cn_gf01"] != want {
			t.Errorf("period %d: %v", period, got)
		}
	}
	if got := challengePeriodSeasons(entries[1:], 1, time.Unix(begin, 0)); len(got) != 0 {
		t.Fatal("the last ended season is not current", got)
	}
	if got := challengePeriodSeasons(entries[:1], 1, time.Unix(end, 0)); len(got) != 0 {
		t.Fatal("period end is exclusive", got)
	}
}

func TestChallengeRankChatSelectsPeriodAndChineseMetric(t *testing.T) {
	a := pluginApp(t)
	scope := GroupScope{"onebot11", "bot", "g"}
	now := time.Now()
	for _, tc := range []struct {
		actor, season string
		stars         float64
		begin, end    time.Time
	}{
		{"低分", "当前期", 5, now.AddDate(0, 0, -7), now.AddDate(0, 0, 7)},
		{"高分", "当前期", 9, now.AddDate(0, 0, -7), now.AddDate(0, 0, 7)},
		{"往期", "上一期", 9, now.AddDate(0, 0, -37), now.AddDate(0, 0, -7)},
	} {
		entry := ChallengeEntry{Kind: "abyss", Region: "cn_gf01", Season: tc.season, ActorID: tc.actor, Nickname: tc.actor, UID: "100000001", Metrics: map[string]float64{"floor": 123, "star": tc.stars}, Begin: tc.begin.Unix(), End: tc.end.Unix(), FirstMS: now.UnixMilli(), UpdatedMS: now.UnixMilli()}
		if err := a.Groups.SubmitChallenge(scope, entry); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Groups.SubmitChallenge(scope, ChallengeEntry{Kind: "abyss", Region: "cn_gf01", Season: "当前期", ActorID: "无时间", Nickname: "无时间", UpdatedMS: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	// Read the snapshots with a fresh store, including their saved bounds.
	a.Groups = &GroupStore{Directory: a.Groups.Directory}
	host := newSDKHost(t, a, nil, nil)
	end, _ := host.groupMessage("#挑战排行 深境螺旋 本期 星数", "挑战排行", "深境螺旋", "本期", "星数")
	text := terminalText(end)
	if !strings.Contains(text, "当前期") || strings.Contains(text, "上一期") || strings.Contains(text, "无时间") || !strings.Contains(text, "高分") || !strings.Contains(text, "低分") || strings.Index(text, "高分") > strings.Index(text, "低分") {
		t.Fatal(text)
	}
	for _, tc := range []struct{ season, metric, want string }{
		{"上期", "star", "上一期"}, {"全部", "综合", "上一期"},
		{"全部", "星星", "深渊的指标"}, {"星数", "", "参数顺序"},
		{"未知期次", "", "指标写在第三项"}, {"当前期", "星数", "当前期"},
	} {
		args := []string{"深渊", tc.season}
		if tc.metric != "" {
			args = append(args, tc.metric)
		}
		end, _ = host.groupMessage("#挑战排行 "+strings.Join(args, " "), "挑战排行", args...)
		if text := terminalText(end); !strings.Contains(text, tc.want) {
			t.Errorf("%v: %s", args, text)
		}
	}
}
func TestChallengeGroupPaginationRestartWithdrawalAndRevision(t *testing.T) {
	dir := t.TempDir()
	s := &GroupStore{Directory: filepath.Join(dir, "groups")}
	scope := GroupScope{"onebot11", "bot", "group"}
	now := time.Now().UnixMilli()
	for i := 0; i < 23; i++ {
		e := ChallengeEntry{Kind: "abyss", Season: "12", Region: "cn_gf01", ActorID: fmt.Sprint(i), UID: fmt.Sprint(10000000 + i), Metrics: map[string]float64{"floor": 12, "star": float64(i)}, FirstMS: now, UpdatedMS: now}
		if err := s.SubmitChallenge(scope, e); err != nil {
			t.Fatal(err)
		}
	}
	a := App{Game: Game{ID: "genshin"}, Groups: &GroupStore{Directory: s.Directory}}
	kind, _ := challengeKind("abyss")
	out, err := a.challengeList(scope, kind, "", 0, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	group := out["groups"].([]map[string]any)[0]
	if len(group["items"].([]ChallengeEntry)) != 3 || group["items"].([]ChallengeEntry)[0].ActorID != "2" {
		t.Fatal(group)
	}
	other := scope
	other.BotID = "other"
	empty, _ := a.challengeList(other, kind, "", 0, "", 0)
	if len(empty["groups"].([]map[string]any)) != 0 {
		t.Fatal("cross bot leak")
	}
	_, err = a.manageChallenge("challenge.clear", map[string]any{"scope": scope, "confirmed": true, "revision": 0})
	if err == nil {
		t.Fatal("stale destructive clear")
	}
	_, err = a.manageChallenge("challenge.clear", map[string]any{"scope": scope, "confirmed": true, "revision": out["revision"], "kind": "abyss"})
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := s.Read(scope)
	if len(fresh.Challenges) != 0 {
		t.Fatal(fresh)
	}
}
