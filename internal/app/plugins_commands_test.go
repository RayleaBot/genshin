package app

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/RayleaBot/genshin/internal/pluginmeta"
)

// The shipped manifests follow upstream wording; these cases pin the words
// whose patterns overlap, so a reordered manifest cannot steal a command.
func TestShippedManifestsResolveUpstreamWording(t *testing.T) {
	cases := map[string][]struct {
		word, id string
		args     []string
	}{
		"genshin": {
			{"雷神面板", "character", []string{"雷神"}},
			{"面板", "panel-list", nil},
			{"面板列表100000001", "panel-list", []string{"100000001"}},
			{"更新面板", "panel-refresh", nil},
			{"更新面板100000001", "panel-refresh", []string{"100000001"}},
			{"米游社更新面板", "panel-refresh-account", nil},
			{"删除面板100000001", "panel-delete", []string{"100000001"}},
			{"圣遗物列表", "artifact-list", nil},
			{"今日素材", "daily-material", nil},
			{"雷神换90级5精护摩换绝缘4", "panel-change", nil},
			{"周三材料", "daily-material", nil},
			{"每日天赋", "daily-material", nil},
			{"天赋", "talent-stat", nil},
			{"今日天赋", "talent-stat", nil},
			{"周日天赋", "talent-stat", nil},
			{"胡桃材料", "materials", []string{"胡桃"}},
			{"胡桃圣遗物", "score", []string{"胡桃"}},
			{"胡桃圣遗物", "score", []string{"胡桃"}},
			{"胡桃伤害2", "build", []string{"胡桃", "2"}},
			{"胡桃武器", "build", []string{"胡桃"}},
			{"刻晴养成", "growth", []string{"刻晴"}},
			{"6.7卡池", "calendar", []string{"6.7"}},
			{"卡池", "pool-help", nil},
			{"6.7上半卡池", "calendar", []string{"6.7", "上半"}},
			{"火神卡池详情", "banner-history", []string{"火神", "详情"}},
			{"胡桃卡池", "banner-history", []string{"胡桃"}},
			{"原石7月", "monthly", []string{"7"}},
			{"角色3", "profile", nil},
			{"今日五星天赋统计", "talent-stat", nil},
			{"胡桃天赋", "talent-wiki", []string{"胡桃"}},
			{"夜兰命座", "talent-wiki", []string{"夜兰"}},
			{"七圣召唤查询卡组3", "tcg_decks", nil},
			{"七圣查询行动牌", "tcg_cards", nil},
			{"原石统计", "monthly-history", nil},
			{"十连武器", "simulation", nil},
			{"武器十连", "simulation", nil},
			{"十连2", "simulation", nil},
			{"单抽", "simulation", nil},
			{"抽卡", "simulation", nil},
			{"定轨护摩", "simulation-fate", []string{"护摩"}},
			{"绑定100000001", "select", []string{"100000001"}},
			{"绑定uid100000001", "select", []string{"100000001"}},
			{"uid", "accounts", nil},
			{"UID2", "accounts", nil},
			{"删除uid1", "uid-remove", []string{"1"}},
			{"兑换码", "codes", nil},
			{"兑换码使用ABC123", "redeem", []string{"ABC123"}},
			{"米游社原神签到", "community-sign", nil},
			{"米游社七七", "search", []string{"七七"}},
			{"甜甜花在哪里", "map", []string{"甜甜花"}},
			{"原魔", "enemy", nil},
			{"原魔90级丘丘人生命值", "enemy", nil},
			{"原魔12-3魔偶剑鬼攻击力", "enemy", nil},
			{"尘歌壶模数123456789012", "blueprint", []string{"123456789012"}},
			{"老婆设置心海", "interaction", nil},
			{"老婆照片", "interaction", nil},
			{"老婆是谁", "interaction", nil},
			{"小宝贝", "interaction", nil},
			{"妻子设置全部", "interaction", nil},
			{"心海照片", "photo", []string{"心海"}},
			{"心海图鉴", "catalog", []string{"心海"}},
			{"胡桃排名", "rank", nil},
			{"总排名", "cloud-total-rank", nil},
			{"角色排名胡桃100000001", "cloud-character-rank", []string{"胡桃100000001"}},
			{"角色排名", "cloud-character-rank", nil},
			{"胡桃排名统计", "cloud-rank-stats", []string{"胡桃"}},
			{"导出面板数据", "cloud-export", nil},
			{"导入面板数据100000001", "cloud-import", []string{"100000001"}},
			{"ark胡桃排行", "cloud-custom-rank", nil},
			{"ark胡桃圣遗物排名", "cloud-custom-rank", nil},
			{"ark自定义排行帮助", "cloud-custom-rank-help", nil},
			{"ark获取面板3", "cloud-custom-rank-panel", []string{"3"}},
			{"ARK获取面板20", "cloud-custom-rank-panel", []string{"20"}},
			{"ark获取面板", "cloud-custom-rank-panel", nil},
			{"arktoken用量", "cloud-usage", nil},
			{"ARKTOKEN用量", "cloud-usage", nil},
			{"ark绑定原神uid", "cloud-bind", nil},
			{"ark验证原神uid", "cloud-verify", nil},
			{"幽境危战排名", "cloud-stygian-rank", nil},
			{"top幽境危战排名6.7", "cloud-stygian-rank", []string{"6.7"}},
			{"幽境危战", "hard_challenge", nil},
			{"胡桃圣遗物排行榜", "rank", nil},
			{"群排名", "rank", nil},
			{"最强胡桃", "rank-top", nil},
			{"最高分排行", "rank-top", nil},
			{"重置胡桃排名", "rank-reset", nil},
			{"刷新排名", "rank-refresh", nil},
			{"关闭群排名", "rank-switch", nil},
			{"挑战排行", "challenge-rank", nil},
			// miao's avatarList takes uid or UID with ASCII pluses before a UID,
			// or pluses alone, where nothing need end after the UID.
			{"100000001", "characters", []string{"100000001"}},
			{"uid100000001", "characters", []string{"100000001"}},
			{"UID+1800000001", "characters", []string{"1800000001"}},
			{"+100000001", "characters", []string{"100000001"}},
			{"＋100000001", "characters", []string{"100000001"}},
			{"1000000012", "characters", []string{"100000001"}},
			{"uid＋100000001", "character-card", nil},
			{"开启公告推送", "subscribe", []string{"公告"}},
			{"原神开启到期活动推送", "subscribe", []string{"到期"}},
			{"推送资讯", "content-push", nil},
			{"2025年札记统计", "monthly-history", nil},
			{"原石任务", "monthly-task", nil},
			{"原石七月", "monthly", []string{"七"}},
			{"札记12月", "monthly", []string{"12"}},
			{"原神帮助", "help", nil},
			{"刷新天赋", "talent-refresh", nil},
			{"强制更新所有天赋", "talent-refresh", nil},
			{"原神刷新天赋", "talent-refresh", nil},
			{"角色养成", "growth-help", nil},
			{"刻晴养成81", "growth", []string{"刻晴", "81"}},
			{"尘歌壶模数养成", "blueprint", nil},
			{"设置全量更新抽卡记录关", "gacha-full", nil},
			{"绑定uid+100000001", "select", []string{"100000001"}},
			{"喵喵更新图像", "artwork", nil},
			{"安卓帮助", "gacha-help-port", nil},
			{"卡池帮助", "pool-help", nil},
			{"面板帮助", "panel-help", nil},
			{"更换面板帮助", "panel-help", nil},
			{"公告", "news", nil},
			{"原神公告3", "news", []string{"3"}},
			{"公告列表", "news", []string{"列表"}},
			{"官方资讯2", "info", []string{"2"}},
			{"活动列表", "events", []string{"列表"}},
			{"活动日历", "live-calendar", nil},
			{"原石预估", "estimate", nil},
			{"盘点", "estimate", nil},
			{"喵喵别名", "alias-help", nil},
			{"喵喵别名原神设置", "alias-set", nil},
			{"喵喵别名删除", "alias-remove", nil},
			{"喵喵别名列表", "alias-list", nil},
			{"设置胡桃别名", "alias-add", []string{"胡桃"}},
			{"删除别名堂主", "alias-remove", []string{"堂主"}},
			{"胡桃别名", "aliases", []string{"胡桃"}},
			{"角色命座", "stat-cons", nil},
			{"角色持有率", "stat-cons", nil},
			{"深渊第12层使用率", "stat-usage", nil},
			{"幽境危战使用率", "stat-usage", nil},
			{"深渊配队", "abyss-team", nil},
			{"上传胡桃照片", "photo-upload", []string{"胡桃"}},
			{"上传胡桃面板图", "panel-image-upload", []string{"胡桃"}},
			{"删除胡桃面板图1", "panel-image-remove", []string{"胡桃", "1"}},
			{"胡桃面板图列表", "panel-image-list", []string{"胡桃"}},
			// Where Yunzai and miao take the same words, Miao-Yunzai decides:
			// miao's rules (priority 50, profile before stat and gacha) come
			// before Yunzai's, miao's switches are forced on, and Yunzai
			// gcLog's accept first rewrites 角色统计 and 武器统计 to their
			// 池统计 forms, which profileStat's yzRule then no longer takes.
			{"抽卡记录", "gacha-detail", nil},
			{"抽奖记录", "gacha-detail", nil},
			{"角色记录", "gacha-detail", nil},
			{"武器池记录", "gacha-detail", nil},
			{"常驻祈愿", "gacha-detail", nil},
			{"集录分析", "gacha-detail", nil},
			{"up池记录", "gacha-detail", nil},
			{"喵喵武器记录", "gacha-detail", nil},
			{"全部记录", "gacha", nil},
			{"全部角色记录", "gacha", nil},
			{"新手记录", "gacha", nil},
			{"新手池记录", "gacha", nil},
			{"角色联动记录", "gacha", nil},
			{"武器联动记录", "gacha", nil},
			{"原神抽卡记录", "gacha", nil},
			{"角色池池记录", "gacha", nil},
			{"抽卡统计", "gacha-stat", nil},
			{"角色统计", "gacha-stat", nil},
			{"武器统计", "gacha-stat", nil},
			{"角色池统计", "gacha-stat", nil},
			{"常驻统计", "gacha-stat", nil},
			{"集录统计", "gacha-stat", nil},
			{"up统计", "gacha-stat", nil},
			{"版本统计", "gacha-stat", nil},
			{"全部统计", "gacha-stat", nil},
			{"角色常驻统计", "gacha-stat", nil},
			{"喵喵角色统计", "gacha-stat", nil},
			{"喵喵全部统计", "gacha-stat", nil},
			{"喵喵角色武器统计", "gacha-stat", nil},
			{"新手统计", "gacha-versions", nil},
			{"新手池统计", "gacha-versions", nil},
			{"原神角色统计", "gacha-versions", nil},
			{"角色池池统计", "gacha-versions", nil},
			{"角色武器统计", "training", nil},
			{"练度统计", "training", nil},
			{"面板练度统计", "training", nil},
			{"喵喵练度统计", "training", nil},
			{"五星列表", "training", nil},
			{"角色列表", "training", nil},
			{"武器汇总", "training", nil},
			{"火角色统计", "training", nil},
			{"我的五星角色列表", "training", nil},
			{"202507剧诗练度统计", "theater-training", []string{"202507"}},
			{"202507幻想角色列表", "theater-training", []string{"202507"}},
			{"202507真境练度汇总", "theater-training", []string{"202507"}},
			{"角色", "characters", nil},
			{"查询", "characters", nil},
			{"角色查询", "characters", nil},
			{"人物", "characters", nil},
			{"五星角色", "characters", nil},
			{"5星角色", "characters", nil},
			{"四星查询", "characters", nil},
			{"喵喵角色", "characters", nil},
			{"喵喵查询", "characters", nil},
			{"角色卡片", "profile", nil},
			{"深渊", "abyss-summary", nil},
			{"本期深渊", "abyss-summary", nil},
			{"喵喵深渊", "abyss-summary", nil},
			{"上传深渊", "abyss-summary", nil},
			{"深渊数据", "abyss-summary", nil},
			{"深境螺旋", "abyss-summary", nil},
			{"上期深渊", "abyss", []string{"上期"}},
			{"往期深境螺旋", "abyss", []string{"往期"}},
			{"深渊12层", "abyss-floor", nil},
			{"上期深渊第十二层", "abyss-floor", []string{"上期"}},
			{"深渊使用率", "stat-usage", nil},
			{"剧诗", "theater", nil},
			{"上期剧诗", "theater", nil},
			{"幻想真境剧诗", "theater", nil},
			{"喵喵幻想", "theater", nil},
			{"本期幻境数据", "theater", nil},
			{"幽境", "hard_challenge", nil},
			{"喵喵上期危战单人", "hard_challenge", nil},
			{"月谕圣牌", "role-cards", nil},
			{"幻想卡片收集", "role-cards", nil},
			{"喵喵剧诗塔罗牌数据", "role-cards", nil},
			{"月谕圣牌交换", "role-cards-exchange", nil},
			{"月谕卡牌换牌", "role-cards-exchange", nil},
			{"越狱card互换", "role-cards-exchange", nil},
			{"圣牌", "character-card", nil},
			{"圣牌交换", "character-card", nil},
			{"雷神", "character-card", nil},
			// A UID glued to the word, as upstream's rules accept it, is the
			// same leading argument as a spaced one.
			{"声望", "profile", nil},
			{"探索100000001", "profile", []string{"100000001"}},
			{"探索度100000001", "profile", []string{"100000001"}},
			{"尘歌壶100000001", "profile", []string{"100000001"}},
			{"角色100000001", "characters", []string{"100000001"}},
			{"五星角色100000001", "characters", []string{"100000001"}},
			{"喵喵查询100000001", "characters", []string{"100000001"}},
			{"练度统计100000001", "training", []string{"100000001"}},
			{"五星列表100000001", "training", []string{"100000001"}},
			{"武器汇总5星1800000001", "training", []string{"1800000001"}},
			// miao's ProfileStat steps aside only for the bare 角色统计 and
			// 武器统计, and Yunzai's rewrite takes only those words.
			{"角色统计100000001", "training", []string{"100000001"}},
			{"202507剧诗练度统计100000001", "theater-training", []string{"202507", "100000001"}},
			{"天赋100000001", "talent-stat", []string{"100000001"}},
			{"周三五星天赋统计100000001", "talent-stat", []string{"100000001"}},
			{"今日天赋100000001", "talent-stat", []string{"100000001"}},
			{"今日素材100000001", "daily-material", []string{"100000001"}},
			{"每日天赋100000001", "daily-material", []string{"100000001"}},
			{"周三材料100000001", "daily-material", []string{"100000001"}},
			{"雷神面板100000001", "character", []string{"雷神", "100000001"}},
			{"雷神详情100000001", "character", []string{"雷神", "100000001"}},
			{"胡桃圣遗物100000001", "score", []string{"胡桃", "100000001"}},
			{"胡桃伤害100000001", "build", []string{"胡桃", "100000001"}},
			{"胡桃武器100000001", "build", []string{"胡桃", "100000001"}},
			{"五星武器100000001", "weapons", []string{"100000001"}},
			{"角色面板100000001", "panel-list", []string{"100000001"}},
			// miao's abyssSummary takes a period only before the word;
			// Yunzai's role.js takes one before or after it.
			{"深渊100000001", "abyss-summary", []string{"100000001"}},
			{"本期深渊100000001", "abyss-summary", []string{"100000001"}},
			{"深渊100000001数据", "abyss-summary", []string{"100000001"}},
			{"上期深渊100000001", "abyss", []string{"上期", "100000001"}},
			{"深渊上期", "abyss", []string{"上期"}},
			{"深境螺旋往期", "abyss", []string{"往期"}},
			{"深渊本期", "abyss", []string{"本期"}},
			{"深渊本期100000001", "abyss", []string{"本期", "100000001"}},
			{"深渊12层100000001", "abyss-floor", []string{"100000001"}},
			{"深渊上期第12层", "abyss-floor", []string{"上期"}},
			{"深境上期十二层100000001", "abyss-floor", []string{"上期", "100000001"}},
			// Yunzai's [上期|往期|本期]* also matches stray characters and
			// both sides at once; only the period words on one side count.
			{"期深渊", "character-card", nil},
			{"上期深渊本期", "character-card", nil},
			{"剧诗100000001", "theater", []string{"100000001"}},
			{"上期剧诗100000001", "theater", []string{"100000001"}},
			{"幻想真境剧诗100000001数据", "theater", []string{"100000001"}},
			{"幽境危战单人100000001", "hard_challenge", []string{"100000001"}},
			{"上期危战100000001数据", "hard_challenge", []string{"100000001"}},
			{"月谕圣牌100000001", "role-cards", []string{"100000001"}},
			{"幻想卡片收集100000001数据", "role-cards", []string{"100000001"}},
		},
	}
	for game, list := range cases {
		manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
		if err != nil {
			t.Fatal(err)
		}
		set, err := newCommandSet(manifest)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range list {
			id, args, ok := set.resolve(tc.word, nil)
			if len(args) == 0 {
				args = nil
			}
			if !ok || id != tc.id || !reflect.DeepEqual(args, tc.args) {
				t.Errorf("%s %s resolved to %q %v %v, want %s %v", game, tc.word, id, args, ok, tc.id, tc.args)
			}
		}
	}
}

// Words sent after a command whose upstream rule reads to the end are read
// as that rule reads them: a UID among digits counts, other digits are
// ignored, and words the rule does not take pass the command over.
func TestTrailingWordsFollowUpstreamRules(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		word string
		sent []string
		id   string
		args []string
	}{
		{"深渊", []string{"2"}, "abyss-summary", nil},
		{"深渊", []string{"2", "100000001"}, "abyss-summary", []string{"100000001"}},
		{"深渊", []string{"100000001", "数据"}, "abyss-summary", []string{"100000001"}},
		{"深渊100000001", []string{"1800000001"}, "abyss-summary", []string{"100000001"}},
		{"上期深渊", []string{"2"}, "abyss", []string{"上期"}},
		{"深渊12层", []string{"2"}, "abyss-floor", nil},
		{"五星武器", []string{"100000001"}, "weapons", []string{"100000001"}},
		{"探索", []string{"3"}, "profile", nil},
		{"练度统计", []string{"100000001"}, "training", []string{"100000001"}},
		{"202507剧诗练度统计", []string{"100000001"}, "theater-training", []string{"202507", "100000001"}},
		{"今日素材", []string{"2"}, "daily-material", nil},
		{"剧诗", []string{"100000001", "数据"}, "theater", []string{"100000001"}},
		// The plugin's own word keeps its words.
		{"信息", []string{"100000001"}, "profile", []string{"100000001"}},
	} {
		id, args, ok := set.resolve(tc.word, tc.sent)
		if len(args) == 0 {
			args = nil
		}
		if !ok || id != tc.id || !reflect.DeepEqual(args, tc.args) {
			t.Errorf("%s %v resolved to %q %v %v, want %s %v", tc.word, tc.sent, id, args, ok, tc.id, tc.args)
		}
	}
	// Words upstream's rule does not take leave the command out.
	for _, tc := range []struct {
		word string
		sent []string
		id   string
	}{
		{"深渊", []string{"abc"}, "abyss-summary"},
		{"深渊数据", []string{"100000001"}, "abyss-summary"},
		{"上期深渊", []string{"上期"}, "abyss"},
		{"武器", []string{"雷"}, "weapons"},
		{"角色卡片", []string{"100000001"}, "profile"},
		{"面板练度统计", []string{"100000001"}, "training"},
		{"uid100000001", []string{"2"}, "characters"},
	} {
		if id, _, _ := set.resolve(tc.word, tc.sent); id == tc.id {
			t.Errorf("%s %v resolved to %s", tc.word, tc.sent, id)
		}
	}
}

// As on the host, a fallback command takes a word only when no ordinary
// command does, wherever the manifest declares it.
func TestFallbackCommandsResolveAfterOrdinaryOnes(t *testing.T) {
	manifest, err := pluginmeta.Read([]byte(`{"id":"raylea.test","version":"1.0.0","commands":[
		{"id":"card","trigger":{"type":"pattern","pattern":"^.+$","fallback":true}},
		{"id":"panel","trigger":{"type":"pattern","pattern":"^(?P<character>.+?)面板$"}}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for word, want := range map[string]string{"雷神面板": "panel", "雷神": "card"} {
		if id, _, ok := set.resolve(word, nil); !ok || id != want {
			t.Errorf("%s resolved to %q %v, want %s", word, id, ok, want)
		}
	}
}

// A hint or static picture answers a command by ID, so each must name one
// the manifest has.
func TestShippedHintsNameManifestCommands(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &data); err != nil {
		t.Fatal(err)
	}
	ids := slices.AppendSeq(slices.Collect(maps.Keys(data.Hints)), maps.Keys(data.Pictures.Static))
	for _, id := range ids {
		if !slices.ContainsFunc(manifest.Commands, func(command pluginmeta.Command) bool { return command.ID == id }) {
			t.Errorf("answer %s names no command", id)
		}
	}
}
