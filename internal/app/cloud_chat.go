package app

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var cloudRankUID = regexp.MustCompile(`[0-9]{9,10}`)

// Upstream's 角色排名 reads the character and the UID out of its words by
// dropping the digits or everything else.
var (
	rankDigits    = regexp.MustCompile(`\d+`)
	rankNonDigits = regexp.MustCompile(`\D+`)
)

// cloudChatCommand answers ark-plugin's rank words in chat (characterRank in
// apps/user.js): 角色排名<角色><UID> is the character's rank among ark's
// uploaded panels, updated first; 总排名 [UID] ranks every character kept
// for the UID; <角色>排名统计 draws the character's distributions. Each
// replies as upstream does, and with dealError's reply when ark refuses.
func (a *App) cloudChatCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	text := strings.Join(args, " ")
	switch command {
	case "cloud-character-rank":
		return a.characterRankCommand(ctx, event, text)
	case "cloud-total-rank":
		return a.totalRankCommand(ctx, event, text)
	}
	entry, ok := a.Catalog.Resolve(strings.TrimSpace(text), "character", a.aliasMap(event))
	if !ok {
		// Upstream swallows a name it cannot read.
		return event.Result(map[string]any{"handled": true})
	}
	return a.rankStatsCommand(ctx, event, entry)
}

// characterRankCommand is ark-plugin's getRank. A character upstream cannot
// read is left to other plugins.
func (a *App) characterRankCommand(ctx context.Context, event *rayleabot.EventContext, text string) error {
	name := strings.TrimSpace(rankDigits.ReplaceAllString(text, ""))
	uid := rankNonDigits.ReplaceAllString(text, "")
	if name == "" || uid == "" {
		return event.SendText("命令格式错误，示例：" + a.Game.Prefix + "角色排名雷电将军123456789")
	}
	entry, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	id, _ := strconv.Atoi(entry.ID)
	result, failure := a.arkAnswer(ctx, event, "rank/data", map[string]any{"uid": uid, "id": id, "update": 1})
	if failure != "" {
		return event.SendText(failure)
	}
	return event.SendText("uid:" + uid + "的" + entry.Name + "全服伤害排名为 " + asText(result["rank"]) + "，伤害评分: " + arkScore(result["score"]))
}

// totalRankCommand is ark-plugin's getAllRank: ark's rank of every character
// kept for the UID named, mentioned or in use, listing those ark ranked.
func (a *App) totalRankCommand(ctx context.Context, event *rayleabot.EventContext, text string) error {
	owner, err := a.panelOwner(ctx, event, cloudRankUID.FindString(text))
	if err != nil || owner.UID == "" {
		return event.SendText(a.needUIDReply())
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	ids := []string{}
	for id := range saved.Panels {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	result, failure := a.arkAnswer(ctx, event, "rank/self", map[string]any{"ids": ids, "uid": owner.UID, "type": arkGame})
	if failure != "" {
		return event.SendText(failure)
	}
	reply := "uid:" + owner.UID + "的" + a.Game.Name + "全服排名数据:\n"
	for index, raw := range asList(result["rank"]) {
		item := asObject(raw)
		if index >= len(ids) || asText(item["retcode"]) != "100" {
			continue
		}
		name := ids[index]
		if entry, ok := a.Catalog.Get(name); ok {
			name = entry.Name
		}
		reply += name + "全服伤害排名为" + asText(item["rank"]) + "，伤害评分: " + arkScore(item["score"]) + "\n"
	}
	return event.SendText(reply)
}

// arkScore is a damage score as ark-plugin writes it, with toFixed(2).
func arkScore(value any) string {
	score, err := strconv.ParseFloat(asText(value), 64)
	if err != nil {
		return asText(value)
	}
	return JSFixed(score, 2)
}

// rankStatsCommand is ark-plugin's <角色>排名统计. Upstream draws the one
// damage distribution rank/specific answered with, which ark no longer
// sends: it now answers with the artifact and damage distributions ark's
// panel page reads for its 排名统计, so this draws those two as the panel
// does, with no panel to mark. Without a picture the reply lists them as the
// management page does. An answer that holds neither is refused as upstream
// refuses one without retcode 100.
func (a *App) rankStatsCommand(ctx context.Context, event *rayleabot.EventContext, character Entry) error {
	id, _ := strconv.Atoi(character.ID)
	decoded, err := a.arkRequest(ctx, event, "rank/specific", map[string]any{"id": id, "percent": 0})
	if err != nil {
		return event.SendText(arkError(a.Game.Prefix, nil))
	}
	image, ok := readRankStats(character, decoded)
	if !ok {
		return event.SendText(arkError(a.Game.Prefix, asObject(decoded)))
	}
	view := View{Title: character.Name + "排名统计", Rows: []Row{}, Sections: distributionSections(asList(decoded))}
	if a.rankStatsImage != nil {
		if drawn, ok := a.rankStatsImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// readRankStats reads rank/specific's answer as ark's panel page reads it,
// with no panel placed on the distributions; false when it holds neither.
func readRankStats(character Entry, decoded any) (RankStatsImage, bool) {
	distributions, _ := decoded.([]any)
	none := RankCurve{Percent: -100, Score: -100}
	image := RankStatsImage{Character: character}
	image.Damage, image.Artis = rankCurves(distributions, [2]RankCurve{none, none})
	if image.Damage == nil && image.Artis == nil {
		return RankStatsImage{}, false
	}
	if image.Damage != nil {
		image.DamageTitle = plainGameText(asText(asObject(asObject(distributions[1])["data"])["name"]))
	}
	return image, true
}

// RankStatsImage is what <角色>排名统计 draws: the character, ark's damage
// and artifact distributions, each nil when ark has none, and the
// calculation the damage ranks.
type RankStatsImage struct {
	Character     Entry
	Damage, Artis *RankCurve
	DamageTitle   string
}

// RankStatsImageBuilder draws 排名统计 with the plugin's template.
type RankStatsImageBuilder func(ImageContext, RankStatsImage) (Image, bool)

// cloudExchangeCommand is ark-plugin's 导出面板数据 and 导入面板数据<UID>
// (uploadPanelData and downloadPanelData): export uploads all the UID's kept
// panels, as the exchange format, for ten minutes; import downloads a UID's
// upload into the UID's kept panels. The UID is the one named, mentioned or
// in use; who may do either is ark's exportPanelData and importPanelData.
func (a *App) cloudExchangeCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	owner, err := a.panelOwner(ctx, event, cloudRankUID.FindString(strings.Join(args, "")))
	if err != nil || owner.UID == "" {
		return event.SendText(a.needUIDReply())
	}
	level := settings(event).Ark.ImportPanelData
	if command == "cloud-export" {
		level = settings(event).Ark.ExportPanelData
	}
	listed, _ := a.accountClient(event).List(ctx, 0)
	if refusal := panelDataRefusal(level, hasAccount(listed), slices.Contains(event.SuperAdmins, event.Event.Actor.ID)); refusal != "" {
		return event.SendText(refusal)
	}
	if command == "cloud-export" {
		saved, err := a.Profiles.Read(owner.UID)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		avatars := a.exportAvatars(ctx, saved)
		if len(avatars) == 0 {
			return event.SendText("面板数据文件不存在，请先更新面板数据")
		}
		raw, _ := json.Marshal(map[string]any{"uid": owner.UID, "avatars": avatars})
		if _, failure := a.arkAnswer(ctx, event, "panel/upload", map[string]any{"uid": owner.UID, "type": arkGame, "data": string(raw)}); failure != "" {
			return event.SendText(failure)
		}
		return event.SendText("导出成功，请在另一个安装此插件的Bot上输入 " + a.Game.Prefix + "导入面板数据" + owner.UID + " ，有效期十分钟~")
	}
	result, failure := a.arkAnswer(ctx, event, "panel/download", map[string]any{"uid": owner.UID, "type": arkGame})
	if failure != "" {
		return event.SendText(failure)
	}
	data, err := cloudPlayerObject(result["data"])
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	avatars, err := cleanCloudPlayer(owner.UID, data)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	// As upstream writes the download into the UID's player data, the panels
	// are kept where 面板 reads them.
	panels := []CharacterPanel{}
	for _, raw := range avatars {
		panel, err := a.cloudAvatarPanel(ctx, raw)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		panels = append(panels, panel)
	}
	if _, err = a.Profiles.Keep(owner.UID, panels, "share", nil); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("导入成功")
}

// cloudAvatarPanel turns an exchange avatar (miao player data, as cleaned by
// cleanCloudAvatar) into a panel, its properties calculated from its parts as
// miao does when it loads saved player data.
func (a *App) cloudAvatarPanel(ctx context.Context, raw json.RawMessage) (CharacterPanel, error) {
	var avatar struct {
		ID      int            `json:"id"`
		Elem    string         `json:"elem"`
		Level   int            `json:"level"`
		Promote *int           `json:"promote"`
		Cons    int            `json:"cons"`
		Talent  map[string]int `json:"talent"`
		Weapon  *struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Level   int    `json:"level"`
			Promote *int   `json:"promote"`
			Affix   int    `json:"affix"`
		} `json:"weapon"`
		Artis map[string]map[string]any `json:"artis"`
	}
	if err := json.Unmarshal(raw, &avatar); err != nil || a.Game.Calc == nil {
		return CharacterPanel{}, gameError("cloud_invalid", "面板文件的角色数据无法读取。")
	}
	id := strconv.Itoa(avatar.ID)
	record, err := findReferenceCharacter(a.Game.Calc, CharacterPanel{ID: id, Element: avatar.Elem})
	if err != nil {
		return CharacterPanel{}, err
	}
	panel := CharacterPanel{ID: id, Level: avatar.Level, Promote: avatar.Promote, Rank: max(0, min(6, avatar.Cons)), Element: record.Element, Source: "share", RankKnown: true, WeaponKnown: true, EquipmentKnown: true,
		Stats: []PanelStat{}, Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}}
	if entry, ok := a.Catalog.Get(id); ok {
		panel.Name = entry.Name
	}
	for slot := 1; slot <= 6; slot++ {
		if gear := avatar.Artis[strconv.Itoa(slot)]; gear != nil {
			if _, piece := decodeCloudGear(a.Game, slot, gear); piece != nil {
				panel.Equipment = append(panel.Equipment, *piece)
			}
		}
	}
	weapon := PanelEquipment{Main: []PanelStat{}, Sub: []PanelStat{}, Complete: true}
	if avatar.Weapon != nil {
		// Player data names the weapon.
		for _, item := range a.Game.Calc.Metadata().Weapons {
			if avatar.Weapon.ID != 0 && item.ID == strconv.Itoa(avatar.Weapon.ID) || avatar.Weapon.Name != "" && item.Name == avatar.Weapon.Name {
				weapon.ID, weapon.Name = item.ID, item.Name
			}
		}
		if entry, ok := a.Catalog.Get(weapon.ID); ok {
			weapon.Rarity = strconv.Itoa(entry.Rarity)
		}
		weapon.Level, weapon.Promote, weapon.Refinement = avatar.Weapon.Level, avatar.Weapon.Promote, max(1, min(5, avatar.Weapon.Affix))
	}
	talents := map[string]int{}
	for key, level := range avatar.Talent {
		talents[key] = level + talentBonus(record, key, panel.Rank)
	}
	return a.computedPanel(ctx, record, panel, weapon, talents)
}

// panelDataRefusal is ark-plugin's checkPermission: the reply refusing
// 导出面板数据 or 导入面板数据 at a permission level, empty when allowed.
func panelDataRefusal(level int, account, superAdmin bool) string {
	switch {
	case level == 0, level == 1 && (account || superAdmin), level == 2 && superAdmin:
		return ""
	case level == 1:
		return "为确保数据安全，目前仅允许绑定CK用户导入/导出自己UID的面板数据，请联系Bot主人导入/导出..."
	case level == 2:
		return "为确保数据安全，目前仅允许主人导入/导出自己UID的面板数据，请联系Bot主人导入/导出..."
	}
	return "当前功能已被禁用..."
}

// hasAccount is Yunzai's user.hasCk: an account of the game is bound.
func hasAccount(listed Accounts) bool {
	return slices.ContainsFunc(listed.Items, func(account Account) bool { return len(account.Roles) > 0 })
}

func cmpOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// cloudUsage is ark-plugin's #arktoken用量 (apps/usage.js) for super
// administrators: the quota ark reports for the ark token, or for anonymous
// requests while none is set.
func (a *App) cloudUsage(ctx context.Context, event *rayleabot.EventContext) error {
	decoded, err := a.arkRequest(ctx, event, "auth/usage", map[string]any{})
	if err != nil {
		return event.SendText("查询失败，服务暂不可用")
	}
	return event.SendText(arkUsageText(decoded))
}

// arkUsageText is usage.js's formatUsageResult: the permission, what is left
// of every request's minute, hour and day quota, the custom ranking's
// multiplier and what is left of its ordinary and advanced quota.
func arkUsageText(decoded any) string {
	if text, ok := decoded.(string); ok {
		return cloudText(text)
	}
	result := asObject(decoded)
	data := asObject(result["data"])
	if asText(result["retcode"]) != "0" || data == nil {
		return cmpOr(cloudText(result["message"]), "查询失败")
	}
	auth, quota := asObject(data["auth"]), asObject(data["quota"])
	rank, custom := asObject(quota["rank"]), asObject(quota["custom"])
	remaining := func(item any) string { return cmpOr(asText(asObject(item)["remaining"]), "-") }
	permission := map[string]string{"1": "高级"}[asText(auth["permission"])]
	return strings.Join([]string{
		"权限类型：" + cmpOr(permission, "普通"),
		"全部请求剩余额度：" + remaining(rank["minute"]) + "/" + remaining(rank["hour"]) + "/" + remaining(rank["day"]),
		"自定义排名请求额度倍率：" + cmpOr(asText(auth["limit_normal"]), "1") + "x",
		"自定义排名普通请求剩余额度：" + remaining(custom["normal"]),
		"自定义排名高级请求剩余额度：" + remaining(custom["advanced"]),
	}, "\n")
}

// arkPanelRefresh is ark-plugin's refreshPanel, which runs before miao's
// 更新面板 for the sender's current UID, whatever UID the words name: with
// newUserPanel a UID without kept panels first takes the panels ark keeps for
// it (ark hands them over to the QQ that verified the UID), then ark is told
// to refresh its own copy. Upstream does not wait for ark's answer; the
// accounts plugin answers only while the event is open, so the notice is
// waited for, five seconds at most.
func (a *App) arkPanelRefresh(ctx context.Context, event *rayleabot.EventContext) {
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil {
		return
	}
	uid := a.currentUID(listed)
	if uid == "" {
		return
	}
	if settings(event).Ark.NewUserPanel {
		a.arkNewUserPanel(ctx, event, uid)
	}
	notified, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, _ = a.arkRequest(notified, event, "panel/refresh", map[string]any{"uid": uid, "type": arkGame})
}

// arkNewUserPanel keeps the panels ark keeps for a UID that has none kept,
// and says how many characters ark sent. ark checks the UID against the QQ
// that verified it, so only OneBot11 senders are asked for.
func (a *App) arkNewUserPanel(ctx context.Context, event *rayleabot.EventContext, uid string) {
	if event.Event.SourceProtocol != "onebot11" || !cloudQQPattern.MatchString(event.Event.Actor.ID) {
		return
	}
	if saved, err := a.Profiles.Read(uid); err != nil || len(saved.Panels) > 0 {
		return
	}
	decoded, err := a.arkRequest(ctx, event, "panel/data", map[string]any{"uid": uid, "type": arkGame, "qq": event.Event.Actor.ID})
	result := asObject(decoded)
	if err != nil || asText(result["retcode"]) != "100" {
		return
	}
	player := asObject(asObject(result["data"])["playerData"])
	if panels := a.arkPlayerPanels(ctx, uid, player); len(panels) > 0 {
		if _, err := a.Profiles.Keep(uid, panels, "share", nil); err == nil {
			notice(ctx, event, "[ark-plugin]已自动从API获取"+strconv.Itoa(len(asObject(player["avatars"])))+"个数据")
		}
	}
}

// arkPlayerPanels are the panels of the player data ark keeps for a UID,
// calculated from their parts as 导入面板数据 does; characters that cannot be
// read are left out.
func (a *App) arkPlayerPanels(ctx context.Context, uid string, player map[string]any) []CharacterPanel {
	panels := []CharacterPanel{}
	avatars, err := cleanCloudPlayer(uid, player)
	if err != nil {
		return panels
	}
	for _, raw := range avatars {
		if panel, err := a.cloudAvatarPanel(ctx, raw); err == nil {
			panels = append(panels, panel)
		}
	}
	return panels
}
