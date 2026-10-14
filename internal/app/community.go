package app

import (
	"context"
	"fmt"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (c AccountsClient) AuthorizeAccount(ctx context.Context, ref string) (Account, error) {
	page := 0
	for {
		listed, err := c.List(ctx, page)
		if err != nil {
			return Account{}, err
		}
		for _, a := range listed.Items {
			if a.Ref == ref {
				return a, nil
			}
		}
		if listed.NextPage == nil || *listed.NextPage <= page {
			break
		}
		page = *listed.NextPage
	}
	return Account{}, gameError("account_missing", "未找到已授权账号。")
}
func (a *App) communityAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"])}
	operation := ""
	params := map[string]any{}
	write := false
	switch action {
	case "community.sign":
		operation = "community_sign"
		write = true
	case "cloudgame.status":
		operation = "cloud_status"
	case "cloudgame.sign":
		operation = "cloud_sign"
		write = true
	case "cloudgame.rewards":
		operation = "cloud_rewards"
	case "cloudgame.claim":
		operation = "cloud_claim"
		write = true
		params["reward_id"] = asText(input["reward_id"])
	default:
		return nil, gameError("operation_denied", "社区签到或云游戏操作不存在。")
	}
	if write && input["confirm"] != true {
		return nil, gameError("input_invalid", "请明确确认本次社区签到或云游戏操作。")
	}
	var result QueryResult
	var err error
	if write {
		result, err = client.ExecuteConfirmed(ctx, choice, a.Game.ID+"."+operation, params)
	} else {
		result, err = client.Execute(ctx, choice, a.Game.ID+"."+operation, params)
	}
	if err != nil {
		return nil, err
	}
	view := View{Title: a.Game.Name + "云游戏", Rows: []Row{}, Note: "凭据由米游社账号插件使用；此处只显示业务结果。"}
	data := result.Data
	if operation == "community_sign" {
		view.Title = a.Game.Name + "米游社签到"
		view.Note = "米游币与米游币任务由米游社账号插件统计。"
		if data["accepted"] == true {
			view.Rows = append(view.Rows, Row{Label: "操作结果", Value: "官方已接受本次社区签到"})
		}
	} else {
		for _, pair := range [][2]string{{"free_time", "免费时长（分钟）"}, {"send_freetime", "本次发放时长（分钟）"}, {"coin_num", "云端余额"}, {"total_time", "总时长（分钟）"}, {"play_card", "畅玩卡"}} {
			if data[pair[0]] != nil {
				view.Rows = append(view.Rows, Row{Label: pair[1], Value: asText(data[pair[0]])})
			}
		}
		if operation == "cloud_status" {
			view.Note = "最近一次已确认操作的快照；此查询不访问云钱包、不领取奖励。"
			if at := asText(data["checked_at_ms"]); at != "" {
				view.Rows = append(view.Rows, Row{Label: "记录时间", Value: calendarMS(data["checked_at_ms"])})
			}
			if data["checked"] != true {
				view.Note = "还没有云游戏状态。配置授权后，可明确选择查询并领取当日免费时长。"
			}
		}
		if operation == "cloud_claim" && data["accepted"] == true {
			view.Rows = append(view.Rows, Row{Label: "领取请求", Value: "官方已接受"})
		}
	}
	return map[string]any{"result": result, "view": view}, nil
}
func calendarMS(raw any) string {
	n, err := strconv.ParseInt(asText(raw), 10, 64)
	if err != nil {
		return ""
	}
	return time.UnixMilli(n).In(time.FixedZone("UTC+8", 28800)).Format("2006-01-02 15:04 UTC+8")
}
func (a *App) communityCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	index := 1
	if len(args) > 1 {
		return event.SendText("请提供可选账号序号，例如 1。")
	}
	if len(args) == 1 {
		var err error
		index, err = strconv.Atoi(args[0])
		if err != nil || index < 1 {
			return event.SendText("账号序号应为正整数。")
		}
	}
	client := a.accountClient(event)
	list, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if index > len(list.Items) {
		return event.SendText("没有对应账号，请先扫码登录或查看账号列表。")
	}
	account := list.Items[index-1]
	action := map[string]string{"community-sign": "community.sign", "cloud-game-status": "cloudgame.status", "cloud-game-sign": "cloudgame.sign"}[command]
	input := map[string]any{"account_ref": account.Ref, "confirm": true}
	result, err := a.communityAction(ctx, client, action, input)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := result["view"].(View)
	view.Subtitle = fmt.Sprintf("账号 %d · %s", index, account.AccountLabel)
	return a.sendView(ctx, event, view)
}
