package app

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/RayleaBot/plugin-genshin/internal/pluginmeta"
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
			{"月谕圣牌交换", "role-cards-exchange", nil},
			{"周三材料", "daily-material", nil},
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
			{"上期深渊", "abyss", []string{"上期"}},
			{"原石7月", "monthly", []string{"7"}},
			{"角色3", "profile", nil},
			{"今日五星天赋统计", "talent-stat", nil},
			{"胡桃天赋", "talent-wiki", []string{"胡桃"}},
			{"夜兰命座", "talent-wiki", []string{"夜兰"}},
			{"角色", "characters", nil},
			{"七圣召唤查询卡组3", "tcg_decks", nil},
			{"七圣查询行动牌", "tcg_cards", nil},
			{"原石统计", "monthly-history", nil},
			{"十连武器", "simulation", nil},
			{"武器十连", "simulation", nil},
			{"十连2", "simulation", nil},
			{"单抽", "simulation", nil},
			{"抽卡", "simulation", nil},
			{"抽卡记录", "gacha", nil},
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
			{"尘歌壶模数123456789012", "blueprint", []string{"123456789012"}},
			{"老婆设置心海", "interaction", []string{"老婆", "设置", "心海"}},
			{"老婆照片", "interaction", []string{"老婆", "照片"}},
			{"心海照片", "photo", []string{"心海"}},
			{"心海图鉴", "catalog", []string{"心海"}},
			{"胡桃排名", "rank", nil},
			{"总排名", "cloud-total-rank", nil},
			{"角色排名胡桃100000001", "cloud-character-rank", []string{"胡桃100000001"}},
			{"胡桃排名统计", "cloud-rank-stats", []string{"胡桃"}},
			{"导出面板数据", "cloud-export", nil},
			{"导入面板数据100000001", "cloud-import", []string{"100000001"}},
			{"胡桃圣遗物排行榜", "rank", nil},
			{"群排名", "rank", nil},
			{"最强胡桃", "rank-top", nil},
			{"最高分排行", "rank-top", nil},
			{"重置胡桃排名", "rank-reset", nil},
			{"刷新排名", "rank-refresh", nil},
			{"关闭群排名", "rank-switch", nil},
			{"挑战排行", "challenge-rank", nil},
			{"100000001", "public-profile", []string{"100000001"}},
			{"开启公告推送", "subscribe", []string{"公告"}},
			{"2025年札记统计", "monthly-history", nil},
			{"原神帮助", "help", nil},
			{"刷新天赋", "talent-refresh", nil},
			{"强制更新所有天赋", "talent-refresh", nil},
			{"原神刷新天赋", "talent-refresh", nil},
			{"月谕卡牌换牌", "role-cards-exchange", nil},
			{"幻想卡片收集", "role-cards", nil},
			{"角色养成", "growth-help", nil},
			{"刻晴养成81", "growth", []string{"刻晴", "81"}},
			{"尘歌壶模数养成", "blueprint", nil},
			{"抽奖记录", "gacha", nil},
			{"武器池记录", "gacha", nil},
			{"角色统计", "gacha-versions", nil},
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
			{"202507剧诗练度统计", "theater-training", []string{"202507"}},
			{"上传胡桃照片", "photo-upload", []string{"胡桃"}},
			{"上传胡桃面板图", "panel-image-upload", []string{"胡桃"}},
			{"删除胡桃面板图1", "panel-image-remove", []string{"胡桃", "1"}},
			{"胡桃面板图列表", "panel-image-list", []string{"胡桃"}},
			{"雷神", "character-card", nil},
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
