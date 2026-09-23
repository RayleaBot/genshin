package app

import (
	"context"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// xiaoyao-cvs-plugin (apps/xiaoyao_image.js) sends the 七圣召唤 cards of its
// 图鉴 library: roleInfo a character's or an enemy's card for 七圣 and 原牌,
// getBasicEvent an event or equipment card; with 动态 or 幻影 the card's
// video instead, and getBasicVoide the video of a card picture a message
// quotes.

// Aliased is a name with its aliases, as upstream's alias lists keep them.
type Aliased struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

// XiaoyaoAliases are the lists of xiaoyao's resources/Atlas_alias its card
// 图鉴 reads: Basic_Event's cards by category, wuqi_tujian's weapons and
// yuanmo_tujian's enemies.
type XiaoyaoAliases struct {
	Version string `json:"version"`
	Cards   []struct {
		Category string    `json:"category"`
		Cards    []Aliased `json:"cards"`
	} `json:"cards"`
	Weapons []Aliased `json:"weapons"`
	Enemies []Aliased `json:"enemies"`
}

// XiaoyaoPictures is xiaoyao's 图鉴 library: file paths with {name} that 图鉴
// reads, and those of the character (Role) and event (Event) cards; Wrong is
// the picture getBasicVoide answers a quote of another picture with.
type XiaoyaoPictures struct {
	Source string        `json:"source"`
	Paths  []string      `json:"paths"`
	Role   string        `json:"role"`
	Event  string        `json:"event"`
	Wrong  StaticPicture `json:"wrong"`
}

// xiaoyaoName is info_img over one list: the name the word is, or one of
// whose aliases it is.
func xiaoyaoName(list []Aliased, word string) (string, bool) {
	for _, item := range list {
		for _, alias := range item.Aliases {
			if alias == word || item.Name == word {
				return item.Name, true
			}
		}
	}
	return "", false
}

// card is info_img over Basic_Event: a category the word names, else the card
// the word is or one of whose aliases it is.
func (x *XiaoyaoAliases) card(word string) (string, bool) {
	for _, group := range x.Cards {
		for _, card := range group.Cards {
			if group.Category == word {
				return group.Category, true
			}
			if card.Name == word || slices.Contains(card.Aliases, word) {
				return card.Name, true
			}
		}
	}
	return "", false
}

var (
	// xiaoyaoAtlasWords is AtlasAlias's rule; it then takes only messages
	// with #, its default.
	xiaoyaoAtlasWords = regexp.MustCompile(`^(#(.*)|.*图鉴)$`)
	xiaoyaoCardWords  = regexp.MustCompile(`原牌|七圣`)
	xiaoyaoEventWords = regexp.MustCompile(`原牌|七圣召唤|七圣|动态|幻影`)
	xiaoyaoVideoWords = regexp.MustCompile(`动态|幻影`)
	// roleInfo and getBasicEvent take these words out of the name.
	xiaoyaoRoleName  = regexp.MustCompile(`#|＃|信息|图鉴|命座|天赋|原牌|七圣召唤|七圣|动态|幻影`)
	xiaoyaoEventName = regexp.MustCompile(`#|＃|信息|图鉴|原牌|七圣召唤|七圣|动态|幻影`)
	travelerElements = []string{"风主", "岩主", "雷主", "草主", "水主", "火主"}
)

// xiaoyaoReply is what xiaoyao sends: a text, or a card picture with the
// video beside it, empty when it has none.
type xiaoyaoReply struct {
	text    string
	picture artworkFile
	video   string
	// play sends the video instead of the picture.
	play bool
}

// xiaoyaoCard is what AtlasAlias sends of roleInfo's 七圣召唤 cards and
// getBasicEvent, for a message as it reached xiaoyao (original) and as miao's
// checks left it (msg); false when they send nothing. AtlasAlias answers only
// while its library is downloaded, which stands for its Atlas.all switch.
// roleInfo also asks for the element when the message names only the
// traveler. Both end the message even when they find no card, though
// getBasicEvent means not to take other plugins' words; the message goes on
// then.
func (a *App) xiaoyaoCard(original, msg string, aliases func() map[string]string) (xiaoyaoReply, bool) {
	library, lists := a.Game.Pictures.Xiaoyao, a.Game.Data.Xiaoyao
	if !xiaoyaoAtlasWords.MatchString(original) || !strings.Contains(msg, "#") || !a.Artwork.Ready(library.Source) || lists == nil {
		return xiaoyaoReply{}, false
	}
	if xiaoyaoCardWords.MatchString(msg) {
		word := xiaoyaoRoleName.ReplaceAllString(msg, "")
		name := ""
		if entry, ok := a.miaoCharacter(word, aliases()); ok {
			if slices.Contains(travelerIDs, entry.ID) && !slices.Contains(travelerElements, word) {
				return xiaoyaoReply{text: "请选择：风主图鉴、岩主图鉴、雷主图鉴、草主图鉴、水主图鉴、火主图鉴"}, true
			}
			name = entry.Name
			if slices.Contains(travelerIDs, entry.ID) {
				name = word
			}
		} else {
			name, _ = xiaoyaoName(lists.Enemies, word)
		}
		if reply, ok := a.xiaoyaoFile(library.Role, name, msg); ok {
			return reply, true
		}
	}
	if xiaoyaoEventWords.MatchString(msg) {
		word := xiaoyaoEventName.ReplaceAllString(msg, "")
		name, ok := lists.card(word)
		if !ok {
			name, _ = xiaoyaoName(lists.Weapons, word)
		}
		if reply, ok := a.xiaoyaoFile(library.Event, name, msg); ok {
			return reply, true
		}
	}
	return xiaoyaoReply{}, false
}

// xiaoyaoFile is send_Msg's reply of a card: its picture, played as the video
// beside it for 动态 or 幻影. Upstream sends the video path whether or not
// the video exists, which fails without a reply; the picture is sent then,
// as upstream falls back to it when the video cannot be made.
func (a *App) xiaoyaoFile(pattern, name, msg string) (xiaoyaoReply, bool) {
	if name == "" {
		return xiaoyaoReply{}, false
	}
	source := a.Game.Pictures.Xiaoyao.Source
	file := strings.ReplaceAll(pattern, "{name}", name)
	if _, found := a.Artwork.File(source, file); !found {
		return xiaoyaoReply{}, false
	}
	reply := xiaoyaoReply{picture: artworkFile{source, file}}
	video := strings.TrimSuffix(file, path.Ext(file)) + ".mp4"
	if _, found := a.Artwork.File(source, video); found {
		reply.video = video
		reply.play = xiaoyaoVideoWords.MatchString(msg)
	}
	return reply, true
}

// xiaoyaoCardTTL is how long send_Msg means to remember a card's video by
// the message that sent it; upstream passes the three hours where the Redis
// client reads no expiry, so its entries never expire.
const xiaoyaoCardTTL = 3 * time.Hour

func xiaoyaoCardKey(event *rayleabot.EventContext, messageID string) string {
	return "xiaoyao-basic:" + event.Event.SourceAdapter + ":" + messageID
}

// sendXiaoyaoCard sends a card reply and remembers the video of a sent card
// by the message's ID.
func (a *App) sendXiaoyaoCard(ctx context.Context, event *rayleabot.EventContext, reply xiaoyaoReply) error {
	if reply.text != "" {
		_, err := post(ctx, event, rayleabot.Text(reply.text))
		return err
	}
	segment := a.artworkReply(event, reply.picture)
	if video, ok := a.Artwork.URL(reply.picture.Source, reply.video); reply.play && ok {
		segment = rayleabot.Passthrough("video", map[string]any{"file": video})
	}
	id, err := post(ctx, event, segment)
	if err != nil || id == "" || reply.video == "" {
		return err
	}
	_, err = event.Actions().KVSetWithOptions(ctx, xiaoyaoCardKey(event, id), artworkFile{reply.picture.Source, reply.video}, rayleabot.KVSetOptions{TTL: xiaoyaoCardTTL})
	return err
}

// xiaoyaoQuote is getBasicVoide for a message with 动态 or 幻影 that quotes a
// picture the bot sent. It sends nothing for other messages, which upstream
// stops silently.
func (a *App) xiaoyaoQuote(ctx context.Context, event *rayleabot.EventContext) (bool, error) {
	id := quotedMessage(event)
	if id == "" {
		return false, nil
	}
	quoted, err := event.Actions().MessageGet(ctx, id)
	if err != nil {
		return false, nil
	}
	sent, ok := quotedPicture(quoted, event.Bot.ID)
	if !ok {
		return false, nil
	}
	var video artworkFile
	if remembered, err := event.Actions().KVGet(ctx, xiaoyaoCardKey(event, id)); err != nil || remembered["exists"] != true || decodeObject(remembered["value"], &video) != nil {
		video = artworkFile{}
	}
	_, err = post(ctx, event, a.quoteReply(video, sent, time.Now()))
	return true, err
}

// quotedPicture tells whether a quoted message, as the chat platform returns
// it, is one picture the bot sent, and when it was sent.
func quotedPicture(quoted map[string]any, bot string) (time.Time, bool) {
	segments := asList(quoted["message"])
	if asText(asObject(quoted["sender"])["user_id"]) != bot || len(segments) != 1 || asText(asObject(segments[0])["type"]) != "image" {
		return time.Time{}, false
	}
	return time.Unix(int64(number(quoted["time"])), 0), true
}

// quoteReply is getBasicVoide's reply to a quoted picture: the card video
// remembered for it, else miao's what.jpg when it was sent within the hour,
// else that it was too long ago.
func (a *App) quoteReply(video artworkFile, sent, now time.Time) rayleabot.Segment {
	if url, ok := a.Artwork.URL(video.Source, video.Path); ok {
		return rayleabot.Passthrough("video", map[string]any{"file": url})
	}
	if now.Sub(sent) < time.Hour {
		return a.staticReply(a.Game.Pictures.Xiaoyao.Wrong)
	}
	return rayleabot.Text("消息太过久远了，俺也忘了动态是啥了，下次早点来吧~")
}
