package app

import (
	"context"
	"encoding/json"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panelExportCommand is ark-plugin's 导出面板: the saved panels of the UID in
// use, or of a mentioned player, sent as a miao player data file. Only
// panels read from the official records carry what the file needs to
// rebuild relic substats, as for 导出面板数据.
func (a *App) panelExportCommand(ctx context.Context, event *rayleabot.EventContext) error {
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil || owner.UID == "" {
		return event.SendText("请先绑定uid")
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText("面板数据文件不存在，请先更新面板数据")
	}
	avatars := map[string]json.RawMessage{}
	for _, kept := range saved.Panels {
		if kept.Official == nil {
			continue
		}
		if id, raw, err := officialCloudAvatar(ctx, a.Game, kept.panel(), kept.Official); err == nil {
			avatars[id] = raw
		}
	}
	if len(avatars) == 0 {
		return event.SendText("面板数据文件不存在，请先更新面板数据")
	}
	content, err := json.Marshal(map[string]any{"uid": owner.UID, "avatars": avatars})
	if err != nil {
		return err
	}
	if len(content) > gachaFileLimit {
		return event.SendText("导出失败：面板文件超过 5 MB")
	}
	return sendFile(event, owner.UID+".json", content)
}
