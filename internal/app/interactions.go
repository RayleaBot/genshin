package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/localdata"
)

type InteractionProfile struct {
	Ref                 string              `json:"ref"`
	Owner               Subject             `json:"owner"`
	Lists               map[string][]string `json:"lists"`
	LastImage           string              `json:"last_image,omitempty"`
	LastImageMS         int64               `json:"last_image_ms,omitempty"`
	LastImageTargetType string              `json:"last_image_target_type,omitempty"`
	LastImageTargetID   string              `json:"last_image_target_id,omitempty"`
	LastEvent           string              `json:"last_event,omitempty"`
	LastPokeMS          int64               `json:"last_poke_ms,omitempty"`
	// EnemyLevel is the level miao's 敌人等级 set for this sender's panel and
	// damage replies; 0 is miao's default.
	EnemyLevel int `json:"enemy_level,omitempty"`
}
type InteractionStore struct {
	mu   sync.Mutex
	Path string
}

func interactionRef(owner Subject) string {
	raw, _ := json.Marshal(owner)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s *InteractionStore) read() ([]InteractionProfile, error) {
	items := []InteractionProfile{}
	err := localdata.Read(s.Path, &items)
	return items, err
}
func (s *InteractionStore) List() ([]InteractionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}
func (s *InteractionStore) Get(owner Subject) (InteractionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return InteractionProfile{}, err
	}
	for _, v := range items {
		if v.Owner == owner {
			return v, nil
		}
	}
	return InteractionProfile{Ref: interactionRef(owner), Owner: owner, Lists: map[string][]string{}}, nil
}
func (s *InteractionStore) edit(owner Subject, fn func(*InteractionProfile) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(v InteractionProfile) bool { return v.Owner == owner })
	if i < 0 {
		if len(items) >= 2000 {
			return gameError("interaction_limit", "互动偏好超过2000个主体，请先清理。")
		}
		items = append(items, InteractionProfile{Ref: interactionRef(owner), Owner: owner, Lists: map[string][]string{}})
		i = len(items) - 1
	}
	if items[i].Lists == nil {
		items[i].Lists = map[string][]string{}
	}
	if err = fn(&items[i]); err != nil {
		return err
	}
	return localdata.Write(s.Path, items)
}
func chatOwner(event *rayleabot.EventContext) Subject {
	return Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
}
func validInteractionOwner(s Subject) bool {
	return s.SourceProtocol != "" && s.SourceAdapter != "" && s.BotID != "" && s.ActorID != "" && len(s.ActorID) <= 256 && len(s.BotID) <= 256
}

// pokeEligible is whether a poke in the conversation shows a card: miao's
// avatarPoke setting, in a group that has not turned the plugin off.
func (a *App) pokeEligible(event *rayleabot.EventContext, owner Subject) (bool, error) {
	if !validInteractionOwner(owner) || !settings(event).PokeCard {
		return false, nil
	}
	if event.Event.Target.Type == "group" {
		config, err := a.Groups.Config(GroupScope{owner.SourceProtocol, owner.SourceAdapter, owner.BotID, event.Event.Target.ID})
		if err != nil {
			return false, err
		}
		if config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	return true, nil
}

// claimPoke answers a sender's poke once, and at most every ten seconds.
func (s *InteractionStore) claimPoke(owner Subject, eventID string, now int64) (bool, error) {
	claimed := false
	err := s.edit(owner, func(v *InteractionProfile) error {
		if v.LastEvent == eventID || v.LastPokeMS+10000 > now {
			return nil
		}
		v.LastEvent = eventID
		v.LastPokeMS = now
		claimed = true
		return nil
	})
	return claimed, err
}

// handlePoke is miao's 戳一戳: a card of any of the sender's characters.
func (a *App) handlePoke(ctx context.Context, event *rayleabot.EventContext) error {
	owner := chatOwner(event)
	if asText(event.Event.Payload["target_id"]) != event.Bot.ID || owner.ActorID == owner.BotID {
		return event.Result(map[string]any{"handled": false})
	}
	eligible, err := a.pokeEligible(event, owner)
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	if !eligible {
		return event.Result(map[string]any{"handled": false})
	}
	claimed, err := a.Interactions.claimPoke(owner, event.Event.EventID, time.Now().UnixMilli())
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	if !claimed {
		return event.Result(map[string]any{"handled": false})
	}
	player, err := a.panelOwner(ctx, event, "")
	if err != nil {
		return event.Result(map[string]any{"handled": false})
	}
	avatars := a.playerAvatars(ctx, event, player, "")
	if len(avatars) == 0 {
		return event.SendText("在当前米游社公开展示的角色中未能找到适合展示的角色..")
	}
	entry, ok := a.Catalog.Get(avatars[rand.IntN(len(avatars))].ID)
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	return a.characterCardFor(ctx, event, player, entry)
}

// sendCharacterPicture is miao's 照片, CharWiki's pic mode: a random one of
// the pictures getCardImg picks the character's from, or 暂无图片.
func (a *App) sendCharacterPicture(event *rayleabot.EventContext, entry Entry) error {
	uploads, photos := a.cardPictures(entry)
	total := len(uploads) + len(photos)
	if total == 0 {
		return event.SendText("暂无图片")
	}
	pick := rand.IntN(total)
	if pick >= len(uploads) {
		return a.sendArtwork(event, photos[pick-len(uploads)])
	}
	a.Media.mu.Lock()
	v, err := a.Media.read(uploads[pick].Ref)
	a.Media.mu.Unlock()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if err = a.rememberImage(event, v.Entry.Ref); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(v.Data)))
}

// cardPictures are the pictures miao's getCardImg picks a character's from,
// for 照片 and the character card: the photos uploaded for it, as 上传照片
// fills its upload folder, and its downloaded photos.
func (a *App) cardPictures(entry Entry) ([]MediaEntry, []artworkFile) {
	a.Media.mu.Lock()
	all, _ := a.Media.entries()
	a.Media.mu.Unlock()
	uploads := []MediaEntry{}
	for _, item := range all {
		if item.CatalogID == entry.ID {
			uploads = append(uploads, item)
		}
	}
	return uploads, a.characterPhotos(entry)
}

func (a *App) interactionCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	owner := chatOwner(event)
	if !validInteractionOwner(owner) {
		return event.SendText("聊天身份不完整。")
	}
	switch command {
	case "original-image":
		p, err := a.Interactions.Get(owner)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if p.LastImage == "" || time.Now().UnixMilli()-p.LastImageMS > 600000 || p.LastImageTargetType != event.Event.Target.Type || p.LastImageTargetID != event.Event.Target.ID {
			return event.SendText("最近十分钟没有由你查询的本地图片。")
		}
		if ref, ok := strings.CutPrefix(p.LastImage, "panel:"); ok {
			return a.sendPanelPicture(event, ref)
		}
		if ref, ok := strings.CutPrefix(p.LastImage, "artwork:"); ok {
			source, file, _ := strings.Cut(ref, "/")
			return a.sendArtwork(event, artworkFile{source, file})
		}
		a.Media.mu.Lock()
		v, err := a.Media.read(p.LastImage)
		a.Media.mu.Unlock()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(v.Data)))
	case "photo":
		// miao's wikiReg ends at the word, and a name it cannot read leaves the
		// message to others.
		if len(args) != 1 {
			return event.Result(map[string]any{"handled": false})
		}
		entry, ok := a.Catalog.Resolve(args[0], "character", a.aliasMap(event))
		if !ok {
			return event.Result(map[string]any{"handled": false})
		}
		return a.sendCharacterPicture(event, entry)
	}
	return event.Result(map[string]any{"handled": false})
}
func (a *App) interactionManage(action string, input map[string]any) (map[string]any, error) {
	s := a.Interactions
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	if action == "interaction.list" {
		var q struct {
			Offset int `json:"offset"`
		}
		if decodeObject(input, &q) != nil || q.Offset < 0 {
			return nil, gameError("input_invalid", "分页无效。")
		}
		start, end := min(q.Offset, len(items)), min(q.Offset+50, len(items))
		var next *int
		if end < len(items) {
			next = &end
		}
		return map[string]any{"items": items[start:end], "next_offset": next}, nil
	}
	if action != "interaction.remove" || input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认移除该主体的互动偏好。")
	}
	ref := asText(input["ref"])
	i := slices.IndexFunc(items, func(p InteractionProfile) bool { return p.Ref == ref })
	if i < 0 {
		return nil, gameError("interaction_missing", "互动偏好不存在。")
	}
	items = slices.Delete(items, i, i+1)
	err = localdata.Write(s.Path, items)
	return map[string]any{"removed": err == nil}, err
}
