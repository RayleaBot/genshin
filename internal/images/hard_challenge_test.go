package images_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func decode(t *testing.T, text string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestHardChallengeFollowsMiao(t *testing.T) {
	app, err := gamekit.New(assets.Kit(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mode := func(difficulty, second int, hutao string) string {
		return `{"has_data":true,"best":{"difficulty":` + strconv.Itoa(difficulty) + `,"second":` + strconv.Itoa(second) + `},"challenge":[{"name":"第一战","second":90,
			"teams":[{"avatar_id":10000046,"name":"胡桃","level":90,"rarity":5,"rank":` + hutao + `},{"avatar_id":10000089,"name":"芙宁娜","level":90,"rarity":5,"rank":0}],
			"best_avatar":[{"avatar_id":10000046,"dps":"123456"},{"avatar_id":10000089,"dps":"654321"}],
			"monster":{"level":100,"icon":"http://insecure.example/m.png","desc":["","受到<color=#FFD780FF>火元素</color>伤害提高。"]}}]}`
	}
	data := decode(t, `{"data":[{"schedule":{"start_time":"1788220800","end_time":"1790812800"},"single":`+mode(6, 300, "1")+`,"mp":`+mode(5, 100, "6")+`},
		{"schedule":{"start_time":"1785600000","end_time":"1788200000"},"single":{"has_data":false},"mp":{"has_data":false}}]}`)
	asked := []string{}
	context := gamekit.ImageContext{Game: app.Game, Catalog: app.Catalog, Query: func(operation string, input map[string]any) (gamekit.QueryResult, error) {
		asked = append(asked, operation)
		if operation == "genshin.hard_challenge_popularity" {
			return gamekit.QueryResult{Data: decode(t, `{"avatar_list":[{"avatar_id":10000089}]}`)}, nil
		}
		// The requester owns Hu Tao at constellation 3 with her staff and a
		// four-piece set; Furina is not in the reply.
		return gamekit.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000046,"level":90,"actived_constellation_num":3},
			"weapon":{"id":13501,"name":"护摩之杖","level":90,"affix_level":5,"rarity":5},
			"relics":[{"id":1,"pos":1,"set":{"name":"炽烈的炎之魔女"}},{"id":2,"pos":2,"set":{"name":"炽烈的炎之魔女"}},{"id":3,"pos":3,"set":{"name":"炽烈的炎之魔女"}},{"id":4,"pos":4,"set":{"name":"炽烈的炎之魔女"}},{"id":5,"pos":5,"set":{"name":"角斗士的终幕礼"}}],
			"skills":[{"skill_id":10461,"skill_type":1,"level":10},{"skill_id":10462,"skill_type":1,"level":13,"extra_level":3},{"skill_id":10463,"skill_type":1,"level":10}]}]}`)}, nil
	}}

	// Without a mode the better record wins: single at difficulty 6.
	image, ok := images.HardChallenge(context, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: data})
	if !ok || image.Data["difficulty"] != 6 || image.Data["difficulty_name"] != "绝境" || image.Data["second"] != "300" {
		t.Fatalf("image = %v", image.Data)
	}
	if strings.Join(asked, ",") != "genshin.character,genshin.hard_challenge_popularity" {
		t.Errorf("asked = %v", asked)
	}
	battle := image.Data["battles"].([]any)[0].(map[string]any)
	team := battle["team"].([]any)
	hutao, furina := team[0].(map[string]any), team[1].(map[string]any)
	if hutao["type"] != "wide" || hutao["cons"] != 3 || hutao["weapon"].(map[string]any)["badge"] != 6 || len(hutao["artis"].([]any)) != 1 {
		t.Errorf("hu tao = %v", hutao)
	}
	if talents := hutao["talents"].([]any); talents[0].(map[string]any)["crown"] != true || talents[1].(map[string]any)["plus"] != true {
		t.Errorf("talents = %v", talents)
	}
	// Furina is not the requester's, so the card shows only what the record
	// knows, and the popularity buff marks her.
	if furina["talents"] != nil || furina["weapon"] != nil || furina["popular"] != true || furina["name"] != "芙宁娜" {
		t.Errorf("furina = %v", furina)
	}
	traits := battle["monster"].(map[string]any)["traits"].([]any)
	pieces := traits[0].([]any)
	if len(traits) != 1 || pieces[1].(map[string]any)["color"] != "#FFD780FF" || pieces[1].(map[string]any)["text"] != "火元素" {
		t.Errorf("traits = %v", traits)
	}

	// A cooperative Hu Tao above the requester's constellation is a guest.
	context.Word = "幽境组队"
	image, _ = images.HardChallenge(context, gamekit.QueryResult{Data: data})
	if guest := image.Data["battles"].([]any)[0].(map[string]any)["team"].([]any)[0].(map[string]any); guest["cons"] != 6 || guest["weapon"] != nil {
		t.Errorf("guest = %v", guest)
	}
	context.Word = "上期幽境"
	if _, ok := images.HardChallenge(context, gamekit.QueryResult{Data: data}); ok {
		t.Error("an empty last period answers in text like upstream")
	}
}
