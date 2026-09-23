package app

import (
	"context"
	"maps"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// CharactersImage is a UID's player data as miao keeps it, which its
// ProfileStat pages draw: the 角色 list (avatarList), 练度统计, 天赋统计 and
// 剧诗练度统计. Role is the UID with what the account gives for it,
// Characters the player's characters as official list entries, Index the
// official index (nil when it could not be read) and Saved the UID's kept
// panels, last 更新面板 and player. Public is a UID read without its account,
// as miao reads another player's with the public cookie pool and notes the
// data may be incomplete.
type CharactersImage struct {
	Role       Role
	Characters []any
	Index      map[string]any
	Saved      SavedProfiles
	Public     bool
}

type CharactersImageBuilder func(ImageContext, CharactersImage) (Image, bool)

// noQueryAccount are the failures of a query that found no account to read
// with, miao's 暂无可用CK.
var noQueryAccount = []string{"plugin.service_unavailable", "plugin.account_public_unavailable", "plugin.account_region_unsupported"}

// characters answers miao's 角色 list of the UID written in the command, a
// mentioned user's or the one in use.
func (a *App) characters(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 0 {
		uid = args[0]
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	image, answered, err := a.playerData(ctx, event, owner)
	if answered {
		return err
	}
	operation, _ := a.operation(a.Game.ID + ".characters")
	return a.sendView(ctx, event, a.playerView(ctx, event, owner, image, operation, a.charactersImage, nil))
}

// training answers miao's 练度统计 and 天赋统计 (ProfileStat.stat) of the UID
// written in the command, a mentioned user's or the one in use, drawn from
// its player data as miao draws it. miao refreshes talents only with the
// owner's cookie: a UID on the requester's account has the account's
// details, any other the panels kept for it.
func (a *App) training(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	uid := ""
	if len(args) > 0 {
		uid = args[0]
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	image, answered, err := a.playerData(ctx, event, owner)
	if answered {
		return err
	}
	// The text reply reads as the character list both pages draw on.
	operation, _ := a.operation(a.Game.ID + "." + strings.ReplaceAll(command, "-", "_"))
	_, textOperation := a.routedQuery(operation, operation.Name)
	return a.sendView(ctx, event, a.playerView(ctx, event, owner, image, textOperation, a.trainingImage, nil))
}

// playerView is a ProfileStat page of a UID's player data: its text reads as
// operation, and build draws it with the command's word and input. A UID on
// the requester's account lends the image the account's character details.
func (a *App) playerView(ctx context.Context, event *rayleabot.EventContext, owner panelOwner, image CharactersImage, operation Operation, build CharactersImageBuilder, input map[string]any) View {
	view := BusinessView(a.Game, operation, QueryResult{Role: image.Role, Data: map[string]any{"list": image.Characters}}, a.Catalog)
	if build == nil {
		return view
	}
	imageContext := a.imageContext(ctx)
	imageContext.Word, imageContext.Input = event.Event.Command(), input
	if owner.Owned {
		client := a.accountClient(event)
		imageContext.Query = func(operation string, input map[string]any) (QueryResult, error) {
			return client.Execute(ctx, owner.Choice, operation, input)
		}
	}
	if drawn, ok := build(imageContext, image); ok {
		view.Image = &drawn
	}
	return view
}

// playerData reads a UID's player data as miao's ProfileStat does. As miao,
// it answers the first query that fails and keeps what it has; answered is
// whether it has already replied instead, as miao's 查询失败 when no
// character is left.
func (a *App) playerData(ctx context.Context, event *rayleabot.EventContext, owner panelOwner) (CharactersImage, bool, error) {
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return CharactersImage{}, true, event.SendText(friendlyError(err))
	}
	image, failed := a.readPlayer(ctx, a.accountClient(event), owner, saved)
	if failed != nil {
		// The answer to a failed query stands in for 查询失败, which still
		// follows 暂无可用CK.
		if len(image.Characters) == 0 && !slices.Contains(noQueryAccount, PublicError(failed).Code) {
			return image, true, event.SendText(friendlyError(failed))
		}
		notice(ctx, event, friendlyError(failed))
	}
	if len(image.Characters) == 0 {
		return image, true, event.SendText(a.noCharacters(owner.UID))
	}
	return image, false, nil
}

// readPlayer reads a UID's player data as miao's refreshAndGetAvatarData:
// the index and then the character list, merged with the panels kept for
// the UID. miao reads a UID with its owner's cookie, else with one cookie
// of the public pool: a UID on the requester's account is read with it, any
// other through the accounts plugin's public query, whose one call reads
// both. failed is the first query that failed; what the others read still
// counts.
func (a *App) readPlayer(ctx context.Context, client AccountsClient, owner panelOwner, saved SavedProfiles) (CharactersImage, error) {
	image := CharactersImage{Role: Role{UID: owner.UID}, Saved: saved, Public: !owner.Owned}
	var index, list QueryResult
	var failed error
	if owner.Owned {
		image.Role = owner.Role
		var err error
		index, failed = client.Execute(ctx, owner.Choice, a.Game.ID+".profile", nil)
		list, err = client.Execute(ctx, owner.Choice, a.Game.ID+".characters", nil)
		if failed == nil {
			failed = err
		}
	} else {
		index.Data, list, failed = client.PublicCharacters(ctx, owner.UID, uidRegion(owner.UID))
	}
	image.Index = index.Data
	image.Characters = playerCharacters(image.Index, asList(list.Data["list"]), saved.Panels)
	return image, failed
}

// playerCharacters are the characters miao keeps in a UID's player data, as
// official list entries: the index's, the character list's in their place,
// then those only a kept panel has. As miao's hasData, a character without a
// level above 1 or a weapon is left out; a kept panel carries its weapon, a
// list entry only when it names it.
func playerCharacters(index map[string]any, list []any, kept map[string]SavedPanel) []any {
	characters, order := map[string]map[string]any{}, []string{}
	for _, raw := range slices.Concat(asList(index["avatars"]), list) {
		avatar := asObject(raw)
		id := asText(avatar["id"])
		if _, ok := characters[id]; !ok {
			order = append(order, id)
		}
		characters[id] = avatar
	}
	for _, id := range slices.Sorted(maps.Keys(kept)) {
		if _, ok := characters[id]; !ok {
			order = append(order, id)
			characters[id] = map[string]any{"id": id, "level": kept[id].Panel.Level, "actived_constellation_num": kept[id].Panel.Rank}
		}
	}
	out := []any{}
	for _, id := range order {
		avatar := characters[id]
		if Int(avatar["level"]) > 1 || asText(asObject(avatar["weapon"])["name"]) != "" || kept[id].Panel.Weapon != nil {
			out = append(out, avatar)
		}
	}
	return out
}

// noCharacters is miao's reply for a UID it has no characters of.
func (a *App) noCharacters(uid string) string {
	return "查询失败，暂未获得" + a.Game.Prefix + uid + "角色数据，请绑定CK或 " + a.Game.Prefix + "更新面板"
}
