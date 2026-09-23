package app

import (
	"context"
	"maps"
	"slices"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// CharactersImage is miao's 角色 list of one UID (ProfileStat.avatarList),
// drawn as miao draws it from the player data it keeps for the UID: Role is
// the UID with what the account gives for it, Characters the player's
// characters as official list entries, Index the official index (nil when it
// could not be read) and Saved the UID's kept panels, last 更新面板 and
// player. Public is a UID read without its account, as miao reads another
// player's with the public cookie pool and notes the data may be incomplete.
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
// mentioned user's or the one in use. As miao, it reads the index and then
// the character list, answers the first query that fails, and draws what it
// has with the panels kept for the UID. miao reads a UID with its owner's
// cookie, else with the public cookie pool: a UID on the requester's account
// is read with it, any other through the accounts plugin's public query.
func (a *App) characters(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 0 {
		uid = args[0]
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	client := a.accountClient(event)
	image := CharactersImage{Role: Role{UID: owner.UID}, Saved: saved, Public: !owner.Owned}
	var index, list QueryResult
	var failed error
	if owner.Owned {
		image.Role = owner.Role
		index, failed = client.Execute(ctx, owner.Choice, a.Game.ID+".profile", nil)
		list, err = client.Execute(ctx, owner.Choice, a.Game.ID+".characters", nil)
	} else {
		region := uidRegion(owner.UID)
		index, failed = client.Public(ctx, "profile", owner.UID, region)
		list, err = client.Public(ctx, "characters", owner.UID, region)
	}
	if failed == nil {
		failed = err
	}
	image.Index = index.Data
	image.Characters = playerCharacters(image.Index, asList(list.Data["list"]), saved.Panels)
	if failed != nil {
		// The answer to a failed query stands in for 查询失败, which still
		// follows 暂无可用CK.
		if len(image.Characters) == 0 && !slices.Contains(noQueryAccount, PublicError(failed).Code) {
			return event.SendText(friendlyError(failed))
		}
		notice(ctx, event, friendlyError(failed))
	}
	if len(image.Characters) == 0 {
		return event.SendText(a.noCharacters(owner.UID))
	}
	operation, _ := a.operation(a.Game.ID + ".characters")
	view := BusinessView(a.Game, operation, QueryResult{Role: image.Role, Data: map[string]any{"list": image.Characters}}, a.Catalog)
	if a.charactersImage != nil {
		imageContext := a.imageContext(ctx)
		imageContext.Word = event.Event.Command()
		if owner.Owned {
			imageContext.Query = func(operation string, input map[string]any) (QueryResult, error) {
				return client.Execute(ctx, owner.Choice, operation, input)
			}
		}
		if drawn, ok := a.charactersImage(imageContext, image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
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
