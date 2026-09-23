package app

import (
	"context"
	"encoding/base64"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// miao's 老婆 (AvatarWife): a relation word shows a card of one of the
// sender's characters of that kind, or of those they chose with 设置; 戳一戳
// shows any of their characters.

// wifeRelation is a relation's key, its words (the first names it in
// replies) and the kind of character in miao's wifeCfg.
type wifeRelation struct {
	key, kind string
	words     []string
}

var wifeRelations = []wifeRelation{
	{"wife", "girlfriend", []string{"老婆", "媳妇", "妻子", "娘子", "宝贝"}},
	{"husband", "boyfriend", []string{"老公", "丈夫", "夫君", "郎君", "死鬼"}},
	{"gf", "girlfriend", []string{"女朋友", "女友", "女神", "女王", "女票"}},
	{"bf", "boyfriend", []string{"男朋友", "男友", "男神", "男票"}},
	{"daughter", "daughter", []string{"女儿", "闺女", "小宝贝"}},
	{"son", "son", []string{"儿子", "犬子"}},
}

// wifeMessage is miao's wifeReg over the command word and its arguments.
var wifeMessage = func() *regexp.Regexp {
	words := []string{}
	for _, relation := range wifeRelations {
		words = append(words, relation.words...)
	}
	// Longer words first, so 小宝贝 is not read as 宝贝; upstream lists 是
	// before 是谁, which leaves 老婆是谁 unanswered there.
	slices.SortStableFunc(words, func(a, b string) int { return len(b) - len(a) })
	return regexp.MustCompile(`^\s*(` + strings.Join(words, "|") + `)\s*(设置|选择|指定|添加|列表|查询|是谁|是|照片|相片|图片|写真|图像)?\s*([^\d]*)\s*(\d*)$`)
}()

// playerAvatars are the characters of the sender's UID, read again from the
// account when the kept ones are over an hour old, as miao's
// refreshMysDetail; only those of kind when it is set.
func (a *App) playerAvatars(ctx context.Context, event *rayleabot.EventContext, owner panelOwner, kind string) []CharacterPanel {
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return nil
	}
	if owner.Owned && (saved.Service != "米游社" || time.Since(time.UnixMilli(saved.RefreshedAtMS)) > time.Hour) {
		if panels, err := a.accountPanels(ctx, a.accountClient(event), owner.Choice); err == nil && len(panels) > 0 {
			if kept, err := a.Profiles.Keep(owner.UID, panels, "米游社", &ShowcaseProfile{Nickname: owner.Role.Nickname, Level: owner.Role.Level}); err == nil {
				saved = kept
			}
		}
	}
	panels := []CharacterPanel{}
	for _, item := range saved.Sorted(a.Catalog, nil) {
		if kind == "" || slices.Contains(a.Catalog.WifeTypes[kind], item.Panel.ID) {
			panels = append(panels, item.Panel)
		}
	}
	return panels
}

// wifeCommand answers the relation words with their actions.
func (a *App) wifeCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	match := wifeMessage.FindStringSubmatch(strings.Join(append([]string{event.Event.Command()}, args...), " "))
	if match == nil {
		return event.Result(map[string]any{"handled": false})
	}
	action, param := match[2], strings.TrimSpace(match[3])
	if action == "" {
		action = "卡片"
	}
	setting := slices.Contains([]string{"设置", "选择", "指定", "添加"}, action)
	if !setting && param != "" {
		return event.Result(map[string]any{"handled": false})
	}
	var relation wifeRelation
	for _, item := range wifeRelations {
		if slices.Contains(item.words, match[1]) {
			relation = item
		}
	}
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	subject := chatOwner(event)
	profile, err := a.Interactions.Get(subject)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	name := relation.words[0]
	switch action {
	case "设置", "选择", "指定", "添加":
		chosen := []string{}
		names := strings.FieldsFunc(param, func(r rune) bool { return strings.ContainsRune(",，、;；", r) })
		if slices.ContainsFunc(names, func(name string) bool {
			return slices.Contains([]string{"全部", "任意", "随机", "全都要"}, strings.TrimSpace(name))
		}) {
			chosen = []string{"随机"}
		} else {
			for _, raw := range names {
				entry, _, ok := a.aliasOwner(strings.TrimSpace(raw), a.aliasMap(event))
				if ok && slices.Contains(a.Catalog.WifeTypes[relation.kind], entry.ID) && !slices.Contains(chosen, entry.ID) {
					chosen = append(chosen, entry.ID)
				}
			}
			if action == "添加" {
				for _, id := range profile.Lists[relation.key] {
					if id != "随机" && !slices.Contains(chosen, id) {
						chosen = append(chosen, id)
					}
				}
			}
			if len(chosen) == 0 {
				return event.SendText("在可选的" + name + "列表中未能找到 " + param + " ~")
			}
		}
		if err := a.Interactions.edit(subject, func(p *InteractionProfile) error { p.Lists[relation.key] = chosen; return nil }); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(name + "已经设置：" + strings.Join(a.wifeNames(chosen), "，"))
	case "列表", "是", "是谁":
		if chosen := profile.Lists[relation.key]; len(chosen) > 0 {
			return event.SendText("你的" + name + "是：" + strings.Join(a.wifeNames(chosen), "，"))
		}
		return event.SendText("尚未设置，回复" + a.Game.Prefix + name + "设置+角色名 来设置，如果设置多位请用逗号间隔")
	case "查询":
		// miao's wifeReg takes 查询 but no action answers it.
		return event.Result(map[string]any{"handled": true})
	}
	// 卡片 and the photo words: one of the chosen characters, else any of the
	// sender's of this kind.
	var entry Entry
	chosen := profile.Lists[relation.key]
	if len(chosen) > 0 && chosen[0] != "随机" {
		entry, _ = a.Catalog.Get(chosen[rand.IntN(len(chosen))])
	} else if avatars := a.playerAvatars(ctx, event, owner, relation.kind); len(avatars) > 0 {
		entry, _ = a.Catalog.Get(avatars[rand.IntN(len(avatars))].ID)
	}
	if entry.ID == "" {
		return event.SendText("在当前米游社公开展示的角色中未能找到适合展示的角色..")
	}
	if action == "卡片" {
		return a.characterCardFor(ctx, event, owner, entry)
	}
	return a.sendCardPicture(ctx, event, entry)
}

// wifeNames are the chosen characters' names, 随机 kept as is.
func (a *App) wifeNames(ids []string) []string {
	names := []string{}
	for _, id := range ids {
		if entry, ok := a.Catalog.Get(id); ok {
			names = append(names, entry.Name)
		} else {
			names = append(names, id)
		}
	}
	return names
}

// sendCardPicture sends the photo a card of the character would draw, as
// miao's photo mode.
func (a *App) sendCardPicture(ctx context.Context, event *rayleabot.EventContext, entry Entry) error {
	picture, _, _, ref := a.cardPicture(ctx, entry)
	if ref == "" {
		return event.SendText(entry.Name + "暂无角色图片")
	}
	var data []byte
	if source, file, ok := strings.Cut(strings.TrimPrefix(ref, "artwork:"), "/"); ok && strings.HasPrefix(ref, "artwork:") {
		data, _ = a.Artwork.Open(source, file)
	} else {
		a.Media.mu.Lock()
		file, err := a.Media.read(ref)
		a.Media.mu.Unlock()
		if err == nil {
			data = file.Data
		}
	}
	if len(data) == 0 || picture.Path == "" {
		return event.SendText(entry.Name + "暂无角色图片")
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
}
