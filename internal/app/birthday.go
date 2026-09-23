package app

import (
	"context"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) birthdayAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{asText(input["account_ref"]), asText(input["role_ref"])}
	var result QueryResult
	var err error
	if action == "birthday.claim" {
		if input["confirm"] != true {
			return nil, gameError("input_invalid", "请明确领取所选生日留影。")
		}
		result, err = client.ExecuteConfirmed(ctx, choice, "genshin.birthday_claim", map[string]any{"role_id": asText(input["role_id"])})
	} else {
		result, err = client.Execute(ctx, choice, "genshin.birthday_list", nil)
	}
	if err != nil {
		return nil, err
	}
	view := View{Title: "留影叙佳期", Subtitle: result.Role.UID, Rows: []Row{}}
	if action == "birthday.claim" {
		view.Rows = append(view.Rows, Row{"领取请求", "官方已接受"})
	} else {
		for _, v := range asList(result.Data["items"]) {
			row := asObject(v)
			view.Rows = append(view.Rows, Row{asText(row["name"]), "角色编号 " + asText(row["role_id"])})
		}
		if len(view.Rows) == 0 {
			view.Note = "今天没有可读取的生日角色。"
		} else {
			view.Note = "在游戏插件管理页查看官方图片并明确领取。"
		}
	}
	return map[string]any{"view": view, "data": result.Data}, nil
}

// birthdayCommand is Yunzai's 留影叙佳期: for each character whose birthday
// it is, the official picture is sent and claimed at once.
func (a *App) birthdayCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) > 1 {
		return event.SendText("使用“" + a.Game.Prefix + "留影叙佳期 [本人UID]”。")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	list, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(list, a.Game.ID, uid)
	if err != nil {
		return event.SendText("请先绑定ck再使用本功能哦~")
	}
	result, err := client.Execute(ctx, choice, "genshin.birthday_list", nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	items := asList(result.Data["items"])
	if len(items) == 0 {
		return event.SendText("今天没有生日角色哦~")
	}
	for index, raw := range items {
		item := asObject(raw)
		name := asText(item["name"])
		notice(ctx, event, "正在获取"+name+"的图片，请稍等~")
		if image := asText(item["image_url"]); image != "" {
			_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: event.Event.SourceProtocol, SourceAdapter: event.Event.SourceAdapter, TargetType: event.Event.Target.Type, TargetID: event.Event.Target.ID,
				Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Image(image)}}})
		}
		if _, err := client.ExecuteConfirmed(ctx, choice, "genshin.birthday_claim", map[string]any{"role_id": asText(item["role_id"])}); err != nil {
			return event.SendText(friendlyError(err))
		}
		if index == len(items)-1 {
			return event.SendText("获取" + name + "的图片成功~")
		}
		notice(ctx, event, "获取"+name+"的图片成功~")
	}
	return nil
}
