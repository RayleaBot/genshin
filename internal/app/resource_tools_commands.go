package app

import (
	"context"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) resourceToolsCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	switch command {
	case "guides", "guide-help", "guide-default":
		return a.guideCommand(ctx, event, command, args)
	case "map":
		return a.mapCommand(ctx, event)
	case "enemy":
		if reply, ok := a.Game.Data.Enemies.value(yunzaiMessage(event)); ok {
			return event.SendText(reply)
		}
	case "blueprint":
		if len(args) < 1 || len(args) > 2 {
			return event.SendText("使用“" + a.Game.Prefix + "摹本 分享码 [UID]”。")
		}
		uid := ""
		if len(args) == 2 {
			uid = args[1]
		}
		client := a.accountClient(event)
		list, err := client.List(ctx, 0)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		choice, _, err := Choose(list, a.Game.ID, uid)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		out, err := a.blueprintAction(ctx, client, "blueprint.read", map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "share_code": args[0]})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return a.sendView(ctx, event, out["view"].(View))
	}
	return event.Result(map[string]any{"handled": false})
}
