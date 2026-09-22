package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
			r, e := challengeExtract(kind, data, f.period)
			if e != nil || r.Metrics[f.key] != f.value {
				t.Fatalf("%+v %v", r, e)
			}
			if f.kind == "peak" && r.Season != "12" {
				t.Fatal("previous period mixed", r)
			}
			if _, e = challengeExtract(kind, map[string]any{}, 1); e == nil {
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
func TestChallengeGroupPaginationRestartWithdrawalAndRevision(t *testing.T) {
	dir := t.TempDir()
	s := &GroupStore{Directory: filepath.Join(dir, "groups")}
	scope := GroupScope{"onebot11", "a", "bot", "group"}
	now := time.Now().UnixMilli()
	for i := 0; i < 23; i++ {
		e := ChallengeEntry{Kind: "abyss", Season: "12", Region: "cn_gf01", ActorID: fmt.Sprint(i), UID: fmt.Sprint(10000000 + i), Metrics: map[string]float64{"floor": 12, "star": float64(i)}, FirstMS: now, UpdatedMS: now}
		if err := s.SubmitChallenge(scope, e); err != nil {
			t.Fatal(err)
		}
	}
	a := App{Game: Game{ID: "genshin"}, Groups: &GroupStore{Directory: s.Directory}}
	kind, _ := challengeKind("abyss")
	out, err := a.challengeList(scope, kind, "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	group := out["groups"].([]map[string]any)[0]
	if len(group["items"].([]ChallengeEntry)) != 3 || group["items"].([]ChallengeEntry)[0].ActorID != "2" {
		t.Fatal(group)
	}
	other := scope
	other.BotID = "other"
	empty, _ := a.challengeList(other, kind, "", "", 0)
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
