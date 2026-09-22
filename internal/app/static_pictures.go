package app

import (
	"encoding/base64"

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
