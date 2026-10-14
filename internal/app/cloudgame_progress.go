package app

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// cloudGameProgress lists the sender's cloud game tasks.
func (a *App) cloudGameProgress(ctx context.Context, event *rayleabot.EventContext) error {
	items, err := a.Reminders.List()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	labels := map[string]string{}
	if listed, err := a.accountClient(event).List(ctx, 0); err == nil {
		for _, account := range listed.Items {
			labels[account.Ref] = account.AccountLabel
		}
	}
	owner := Subject{SourceProtocol: event.Event.SourceProtocol, BotID: event.Event.BotID, ActorID: event.Event.Actor.ID}
	lines := []string{}
	for _, task := range items {
		if !task.Owner.same(owner) || task.Kind != "cloudgame" {
			continue
		}
		lines = append(lines, cloudGameTaskLine(task, cmp.Or(labels[task.AccountRef], "已不可用")))
	}
	if len(lines) == 0 {
		return event.SendText("没有云游戏任务。发送“" + a.Game.Prefix + "云游戏任务”查看用法。")
	}
	return event.SendText(strings.Join(lines, "\n"))
}

// cloudGameTaskLine describes one account task for 云游戏任务进度.
func cloudGameTaskLine(task Reminder, account string) string {
	zone := time.FixedZone("UTC+8", 28800)
	when := "每天"
	if task.Once {
		when = "一次"
	}
	state := "等待执行"
	if task.LastCode != "" {
		state = "上次已执行"
		if !strings.HasPrefix(task.LastCode, "cloud_") {
			state = "上次未完成"
		}
	}
	line := fmt.Sprintf("云游戏任务 · 账号 %s · %s %02d:%02d · %s", account, when, task.Hour, task.Minute, state)
	switch {
	case !task.Enabled:
		line += " · 已停止"
	case task.NextCheckMS > 0:
		line += " · 下次 " + time.UnixMilli(task.NextCheckMS).In(zone).Format("01-02 15:04")
	}
	if !task.Once && task.ExpiresAtMS > 0 {
		line += " · " + time.UnixMilli(task.ExpiresAtMS).In(zone).Format("01-02") + " 到期"
	}
	return line
}
