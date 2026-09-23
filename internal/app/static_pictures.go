package app

import (
	"cmp"
	"context"
	"encoding/base64"
	"slices"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// StaticPicture is a command upstream answers with an image it ships, as
// miao-plugin's 面板帮助.
type StaticPicture struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

// staticCommand sends a static picture command's image.
func (a *App) staticCommand(event *rayleabot.EventContext, static StaticPicture) error {
	if !a.Artwork.Ready(static.Source) {
		return event.SendText("暂无图片素材。" + a.pictureHint([]PictureSource{{Source: static.Source}}))
	}
	data, err := a.Artwork.Open(static.Source, static.Path)
	if err != nil {
		return event.SendText("图片素材不完整，请管理员重新下载素材。")
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
}

// sendForward sends parts as one forwarded message, as upstream's
// makeForwardMsg, or as one ordinary message where forwarding is not
// available.
func (a *App) sendForward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) error {
	if event.Event.SourceProtocol == "onebot11" {
		messages := []rayleabot.ForwardMessage{}
		for _, part := range parts {
			content := []any{}
			for _, segment := range part {
				content = append(content, map[string]any{"type": segment.Type, "data": segment.Data})
			}
			messages = append(messages, rayleabot.ForwardMessage{"type": "node", "data": map[string]any{"user_id": event.Bot.ID, "nickname": cmp.Or(event.Bot.Nickname, a.Game.Name), "content": content}})
		}
		if _, err := event.Actions().MessageForwardSend(ctx, rayleabot.MessageForwardSendRequest{TargetType: rayleabot.ConversationType(event.Event.Target.Type), TargetID: event.Event.Target.ID, Messages: messages}); err == nil {
			return event.Result(map[string]any{"handled": true})
		}
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, slices.Concat(parts...)...)
}
