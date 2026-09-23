package app

import (
	"cmp"
	"context"
	"encoding/base64"
	"slices"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// StaticPicture is an image upstream ships and answers with, as
// miao-plugin's 面板帮助.
type StaticPicture struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

// staticCommand sends a static picture command's image.
func (a *App) staticCommand(event *rayleabot.EventContext, static StaticPicture) error {
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, a.staticReply(static))
}

// staticReply is a static picture to send, or the text that tells why it
// cannot be sent.
func (a *App) staticReply(static StaticPicture) rayleabot.Segment {
	if !a.Artwork.Ready(static.Source) {
		return rayleabot.Text("暂无图片素材。" + a.pictureHint(static.Source))
	}
	data, err := a.Artwork.Open(static.Source, static.Path)
	if err != nil {
		return rayleabot.Text("图片素材不完整，请管理员重新下载素材。")
	}
	return rayleabot.Image("base64://" + base64.StdEncoding.EncodeToString(data))
}

// sendForward sends parts as one forwarded message, as upstream's
// makeForwardMsg, or as one ordinary message where forwarding is not
// available.
func (a *App) sendForward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) error {
	if a.forward(ctx, event, parts) {
		return event.Result(map[string]any{"handled": true})
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, slices.Concat(parts...)...)
}

// forward sends parts as one forwarded message without ending the event;
// false when the chat platform does not take it.
func (a *App) forward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) bool {
	if event.Event.SourceProtocol != "onebot11" {
		return false
	}
	messages := []rayleabot.ForwardMessage{}
	for _, part := range parts {
		content := []any{}
		for _, segment := range part {
			content = append(content, map[string]any{"type": segment.Type, "data": segment.Data})
		}
		messages = append(messages, rayleabot.ForwardMessage{"type": "node", "data": map[string]any{"user_id": event.Bot.ID, "nickname": cmp.Or(event.Bot.Nickname, a.Game.Name), "content": content}})
	}
	_, err := event.Actions().MessageForwardSend(ctx, rayleabot.MessageForwardSendRequest{TargetType: rayleabot.ConversationType(event.Event.Target.Type), TargetID: event.Event.Target.ID, Messages: messages})
	return err == nil
}
