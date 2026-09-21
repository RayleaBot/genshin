// Package images draws this plugin's replies with its own templates, following
// the upstream Yunzai images for the same commands.
package images

import (
	"fmt"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// Builders lists the image builders by the operation they draw.
func Builders() map[string]gamekit.ImageBuilder {
	return map[string]gamekit.ImageBuilder{"genshin.note": Note, "genshin.abyss": Abyss, "genshin.theater": Combat, "genshin.hard_challenge": HardChallenge, "genshin.abyss_floor": AbyssFloor, "genshin.characters": Characters}
}

var weekdays = [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// noteIcons are the row icons of the upstream real-time note, from the
// downloaded Yunzai genshin resources.
var noteIcons = []string{"树脂", "洞天宝钱", "委托", "派遣", "周本", "参量质变仪"}

// Note draws the real-time note the way Yunzai's daily-note-gs does: six rows
// with the same wording, times and highlight rules.
func Note(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	data := result.Data
	if _, ok := data["current_resin"]; !ok {
		return gamekit.Image{}, false
	}
	now := context.Now.In(time.FixedZone("UTC+8", 8*3600))
	row := func(icon, name, detail, value string, alert bool) map[string]any {
		return map[string]any{"icon": "icon-" + icon, "name": name, "detail": detail, "value": value, "alert": alert}
	}
	ratio := func(current, maximum string) string {
		return gamekit.Text(data[current]) + "/" + gamekit.Text(data[maximum])
	}

	resin := "树脂已完全恢复"
	if seconds := gamekit.Int(data["resin_recovery_time"]); seconds > 0 {
		resin = "将于" + clock(now, seconds, " ") + " 全部恢复"
	}

	coin := "存储已满"
	if seconds := gamekit.Int(data["home_coin_recovery_time"]); seconds > 0 {
		if days := seconds / 86400; days > 0 {
			coin = fmt.Sprintf("预计%d天%d小时%d分钟后达到上限", days, seconds/3600%24, seconds/60%60)
		} else {
			coin = "预计" + clock(now, seconds, "") + "后达到上限"
		}
	}
	coinFull := gamekit.Int(data["max_home_coin"]) > 0 && float64(gamekit.Int(data["current_home_coin"]))/float64(gamekit.Int(data["max_home_coin"])) > 0.9

	commission := "今日委托奖励未领取"
	if received, _ := data["is_extra_task_reward_received"].(bool); received || gamekit.Int(data["is_extra_task_reward_received"]) == 1 {
		commission = "今日委托奖励已领取"
	}

	expedition := "尚未进行派遣"
	if expeditions, _ := data["expeditions"].([]any); len(expeditions) > 0 {
		earliest := -1
		for _, raw := range expeditions {
			item, _ := raw.(map[string]any)
			if remained := gamekit.Int(item["remained_time"]); earliest < 0 || remained < earliest {
				earliest = remained
			}
		}
		expedition = "派遣已完成"
		if earliest > 0 {
			expedition = "将于" + clock(now, earliest, " ") + " 完成"
		}
	}

	weekly := "周本树脂减半次数已用"
	remaining := gamekit.Int(data["remain_resin_discount_num"])
	if remaining <= 0 {
		weekly = "周本已完成"
	}

	transformer, _ := data["transformer"].(map[string]any)
	transformerDetail, transformerValue, transformerReady := "尚未获得", "尚未获得", false
	if obtained, _ := transformer["obtained"].(bool); obtained {
		recovery, _ := transformer["recovery_time"].(map[string]any)
		if reached, _ := recovery["reached"].(bool); reached {
			transformerDetail, transformerValue, transformerReady = "已准备完成", "可使用", true
		} else {
			wait := ""
			for _, part := range [][2]string{{"Day", "天"}, {"Hour", "小时"}, {"Minute", "分钟"}} {
				if value := gamekit.Int(recovery[part[0]]); value > 0 {
					wait += strconv.Itoa(value) + part[1]
				}
			}
			transformerDetail, transformerValue = wait+"后可使用", "冷却中"
		}
	}

	rows := []any{
		row("树脂", "原粹树脂", resin, ratio("current_resin", "max_resin"), false),
		row("洞天宝钱", "洞天宝钱", coin, ratio("current_home_coin", "max_home_coin"), coinFull),
		row("委托", "每日委托任务", commission, ratio("finished_task_num", "total_task_num"), false),
		row("派遣", "探索派遣", expedition, ratio("current_expedition_num", "max_expedition_num"), false),
		// Upstream counts the used discounts from a fixed three.
		row("周本", "值得铭记的强敌", weekly, fmt.Sprintf("%d/%s", 3-remaining, gamekit.Text(data["resin_discount_num_limit"])), false),
		row("参量质变仪", "参量质变仪", transformerDetail, transformerValue, transformerReady),
	}
	resources := []rayleabot.RenderImageResource{}
	// Upstream sets the whole card in its tttgbnumber font.
	if font, ok := context.ArtworkResource("tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf"); ok {
		resources = append(resources, font)
	}
	for _, icon := range noteIcons {
		if resource, ok := context.ArtworkResource("icon-"+icon, "yunzai-genshin", "resources/html/player/items/"+icon+".png"); ok {
			resources = append(resources, resource)
		}
	}
	return gamekit.Image{
		Template: "note",
		Data: map[string]any{
			"uid":  result.Role.UID,
			"day":  now.Format("01-02 15:04") + " " + weekdays[now.Weekday()],
			"rows": rows,
		},
		Resources: resources,
	}, true
}

// clock formats when a countdown ends as upstream does: HH:mm, prefixed with
// 明天 when it ends on another day and with sameDay otherwise.
func clock(now time.Time, seconds int, sameDay string) string {
	end := now.Add(time.Duration(seconds) * time.Second)
	if end.Day() != now.Day() {
		return "明天 " + end.Format("15:04")
	}
	return sameDay + end.Format("15:04")
}
