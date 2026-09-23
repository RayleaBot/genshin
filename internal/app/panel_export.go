package app

import (
	"context"
	"encoding/json"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panelExportCommand is ark-plugin's 导出面板: the saved panels of the UID in
// use, or of a mentioned player, sent as a miao player data file. Only
// panels read from the official records carry what the file needs to
// rebuild relic substats, as for 导出面板数据. Who may export is ark's
// exportPanelRequire; a refused user gets no reply, as upstream.
func (a *App) panelExportCommand(ctx context.Context, event *rayleabot.EventContext) error {
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil || owner.UID == "" {
		return event.SendText("请先绑定uid")
	}
	if !a.panelExportAllowed(ctx, event, owner.UID) {
		return event.Result(map[string]any{"handled": true})
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

// panelExportAllowed reads ark's exportPanelRequire as its settings describe
// it: 0 anyone, 1 users with an account or whose UID ark verified for their
// QQ, 2 users with an account, 3 no one. Upstream's condition runs the other
// way, exporting for 3, never for 0, and for users without an account or
// whose UID ark has not verified.
func (a *App) panelExportAllowed(ctx context.Context, event *rayleabot.EventContext, uid string) bool {
	level := settings(event).Ark.ExportPanelRequire
	if level == 0 {
		return true
	}
	if level > 2 {
		return false
	}
	if listed, err := a.accountClient(event).List(ctx, 0); err == nil && hasAccount(listed) {
		return true
	}
	if level != 1 || event.Event.SourceProtocol != "onebot11" || !cloudQQPattern.MatchString(event.Event.Actor.ID) {
		return false
	}
	decoded, err := a.arkRequest(ctx, event, "verify/user", map[string]any{"uid": uid, "qq": event.Event.Actor.ID, "type": arkGame})
	return err == nil && asText(asObject(decoded)["retcode"]) == "100"
}
