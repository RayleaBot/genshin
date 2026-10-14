package app

import (
	"context"
	"fmt"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) cloudGameTaskCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	kind := "cloudgame"
	if command == "cloud-game-stop" {
		items, err := a.Reminders.List()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		owner := Subject{SourceProtocol: event.Event.SourceProtocol, BotID: event.Event.BotID, ActorID: event.Event.Actor.ID}
		count := 0
		for _, task := range items {
			if task.Kind == kind && task.Owner.same(owner) {
				if _, err = a.removeDelegatedTask(ctx, event, task.Ref, kind); err != nil {
					return event.SendText(friendlyError(err))
				}
				count++
			}
		}
		return event.SendText(fmt.Sprintf("已停止你的 %d 个云游戏任务。", count))
	}
	usage := a.Game.Prefix + "云游戏任务 <一次|每天> [账号序号]，明确查询钱包并领取可能发放的时长；每日默认北京时间 08:00，有效 30 天。"
	if len(args) < 1 || len(args) > 2 || (args[0] != "一次" && args[0] != "每天") {
		return event.SendText("格式：" + usage)
	}
	index := 1
	if len(args) == 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			return event.SendText("账号序号须为正整数。")
		}
		index = n
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if index > len(listed.Items) {
		return event.SendText("没有对应账号，请先扫码登录。")
	}
	_, err = a.signinTask(ctx, event, kind+".task.create", map[string]any{"account_ref": listed.Items[index-1].Ref, "hour": 8, "days": 30, "once": args[0] == "一次", "confirm": true, "notify": args[0] == "一次"})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("云游戏任务已创建。发送“" + a.Game.Prefix + "云游戏任务进度”查看进度，或发送“" + a.Game.Prefix + "关闭云游戏任务”停止。")
}
