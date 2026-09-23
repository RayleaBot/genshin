package images_test

import (
	"testing"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/app"
	"github.com/RayleaBot/plugin-genshin/internal/assets"
	"github.com/RayleaBot/plugin-genshin/internal/images"
)

func TestRoleSummaryFollowsMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Furina is the player's own; Hu Tao is lent in the first act and fielded
	// in the tarot challenge, and Nahida is a trial character.
	data := decode(t, `{"data":[{"has_detail_data":true,"schedule":{"start_date_time":{"month":9}},
		"stat":{"difficulty_id":5,"get_medal_round_list":[1,0,1,1,1,1,1,1,1,1,1,1],"coin_num":330,"avatar_bonus_num":12,"rent_cnt":3},
		"detail":{"fight_statisic":{"total_use_time":1234},"rounds_data":[
			{"round_id":1,"is_get_medal":true,"is_tarot":false,"finish_time":"1788300000","enemies":[{"icon":"https://preview.local/enemy.png"}],
			 "splendour_buff":{"summary":{"total_level":3},"buffs":[{"icon":"https://preview.local/buff.png","level":2}]},"choice_cards":[{"icon":"https://preview.local/card.png"}],
			 "avatars":[{"avatar_id":10000089,"avatar_type":1,"level":90},{"avatar_id":10000046,"avatar_type":3,"level":80},{"avatar_id":10000073,"avatar_type":2,"level":70}]},
			{"round_id":0,"is_get_medal":false,"is_tarot":true,"tarot_serial_no":2,"finish_time":"1788300100","enemies":[],
			 "splendour_buff":{"summary":{"total_level":0},"buffs":[]},"choice_cards":[],
			 "avatars":[{"avatar_id":10000046,"avatar_type":1,"level":90}]}]}},
		{"has_detail_data":false}]}`)
	asked := [][]any{}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Now(),
		Query: func(operation string, input map[string]any) (app.QueryResult, error) {
			asked = append(asked, input["character_ids"].([]any))
			return app.QueryResult{Data: decode(t, `{"list":[{"base":{"id":10000089,"level":90,"actived_constellation_num":2}},{"base":{"id":10000046,"level":90,"actived_constellation_num":1}}]}`)}, nil
		}}
	image, ok := images.RoleSummary(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok {
		t.Fatal("role summary")
	}
	// Only the player's own cast is queried.
	if len(asked) != 1 || len(asked[0]) != 2 {
		t.Errorf("asked = %v", asked)
	}
	if image.Data["month"] != "9" || image.Data["difficulty"] != "月谕" || image.Data["time"] != "1234" || image.Data["coins"] != "330" {
		t.Errorf("head = %v", image.Data)
	}
	medals := image.Data["medals"].([]any)
	if len(medals) != 12 || medals[10].(map[string]any)["line"] != true || medals[9].(map[string]any)["line"] != false {
		t.Errorf("medals = %v", medals)
	}
	acts := image.Data["acts"].([]any)
	first, tarot := acts[0].(map[string]any), acts[1].(map[string]any)
	if first["title"] != "第 1 幕" || first["hp"] != 2400 || first["atk"] != 150 || first["em"] != 60 || len(first["gains"].([]any)) != 1 || first["time"] != "09-02 06:00:00" {
		t.Errorf("first act = %v", first)
	}
	// Lent and trial characters show miao's label and their act level; Hu
	// Tao keeps the player's own card with the lent level laid over it, as
	// miao merges the two.
	team := first["team"].([]any)
	furina, lent, trial := team[0].(map[string]any), team[1].(map[string]any), team[2].(map[string]any)
	if furina["cons"] != 2 || furina["type"] != "wide" || lent["cons"] != "助演" || lent["level"] != 80 || trial["cons"] != "试用" || trial["level"] != 70 || trial["type"] != "mini" {
		t.Errorf("team = %v", team)
	}
	if tarot["title"] != "圣牌挑战 II" || tarot["team"].([]any)[0].(map[string]any)["cons"] != "助演" || tarot["enemy"] != "" {
		t.Errorf("tarot = %v", tarot)
	}
	// 上期 reads the second period, which has no detailed record yet.
	context.Word = "上期剧诗"
	if _, ok := images.RoleSummary(context, app.QueryResult{Data: data}); ok {
		t.Error("a period without details should answer in text")
	}
}
