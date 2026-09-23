package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

type RoleCard struct {
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Unlocked bool   `json:"unlocked"`
	// Icon is the card's official picture; offers kept before it was
	// recorded have none.
	Icon string `json:"icon,omitempty"`
}
type RoleCardOffer struct {
	ActorID  string     `json:"actor_id"`
	Nickname string     `json:"nickname"`
	UID      string     `json:"uid"`
	Region   string     `json:"region"`
	Cards    []RoleCard `json:"cards"`
	SavedMS  int64      `json:"saved_ms"`
}

// RoleCardMatch is a group member to trade with: Give are the requester's
// spare cards the member lacks, Receive the member's spare cards the
// requester lacks, each as its owner holds it.
type RoleCardMatch struct {
	ActorID  string     `json:"actor_id"`
	Nickname string     `json:"nickname"`
	UID      string     `json:"uid"`
	Give     []RoleCard `json:"give"`
	Receive  []RoleCard `json:"receive"`
	Score    int        `json:"score"`
	SavedMS  int64      `json:"saved_ms"`
}

// RoleCardsImage is 月谕圣牌 as miao's stat/role-card draws it, or with
// Exchange 月谕圣牌交换 as stat/role-card-exchange does.
type RoleCardsImage struct {
	UID              string
	Collected, Total int
	Cards            []RoleCard
	Exchange         bool
	Group            string
	Matches          []RoleCardMatch
}

// RoleCardsImageBuilder draws RoleCardsImage.
type RoleCardsImageBuilder func(ImageContext, RoleCardsImage) (Image, bool)

// errNoRoleCards is miao's reply when the theater record has no collection.
var errNoRoleCards = gameError("cards_unavailable", "暂未获得「月谕圣牌」收藏数据...")

// parseRoleCards reads the collection in the official order, as miao lists
// it.
func parseRoleCards(data map[string]any) ([]RoleCard, error) {
	if data["tarot_card_state"] == nil {
		return nil, errNoRoleCards
	}
	state := asObject(data["tarot_card_state"])
	raw, ok := state["list"].([]any)
	if !ok || len(raw) > 200 {
		return nil, gameError("cards_unavailable", "官方尚未返回完整月谕圣牌收藏。")
	}
	cards := []RoleCard{}
	seen := map[string]bool{}
	for _, v := range raw {
		m := asObject(v)
		name := plainGameText(asText(m["name"]))
		unlocked, ok := m["is_unlock"].(bool)
		n, known := challengeNumber(m["unlock_num"])
		if name == "" || len([]rune(name)) > 64 || !ok || !known || !finiteRange(n, 0, 10000) || n != float64(int(n)) || seen[name] {
			return nil, gameError("cards_unavailable", "圣牌数据缺少数量、解锁状态或名称。")
		}
		seen[name] = true
		cards = append(cards, RoleCard{Name: name, Count: int(n), Unlocked: unlocked, Icon: asText(m["icon"])})
	}
	return cards, nil
}

// missing is whether miao counts a card as one to trade for.
func (c RoleCard) missing() bool { return !c.Unlocked || c.Count <= 0 }

// spareRoleCards are the unlocked cards held more than once, by name.
func spareRoleCards(cards []RoleCard) map[string]RoleCard {
	spare := map[string]RoleCard{}
	for _, c := range cards {
		if c.Unlocked && c.Count > 1 {
			spare[c.Name] = c
		}
	}
	return spare
}
func roleCardView(role Role, cards []RoleCard) View {
	view := View{Title: "月谕圣牌收藏", Subtitle: role.Nickname + " · " + role.UID, Rows: []Row{}, Note: "交换匹配需在群内主动提交；查询本身不加入群交换列表。"}
	for _, c := range cards {
		text := fmt.Sprintf("持有%d张", c.Count)
		if c.missing() {
			text += " · 缺少"
		} else if c.Count > 1 {
			text += fmt.Sprintf(" · 多余%d张", c.Count-1)
		}
		view.Rows = append(view.Rows, Row{c.Name, text})
	}
	return view
}

// roleCardMatches pairs the requester with the members whose spare cards
// each lacks, as miao's RoleCardExchange: the cards the member can give in
// the requester's order, those the requester can give in the member's.
func roleCardMatches(self RoleCardOffer, others []RoleCardOffer, now int64) []RoleCardMatch {
	selfSpare := spareRoleCards(self.Cards)
	out := []RoleCardMatch{}
	for _, other := range others {
		if other.ActorID == self.ActorID || other.UID == self.UID || other.Region != self.Region || other.SavedMS <= now-int64(30*24*time.Hour/time.Millisecond) {
			continue
		}
		candidate := RoleCardMatch{ActorID: other.ActorID, Nickname: other.Nickname, UID: other.UID, SavedMS: other.SavedMS, Give: []RoleCard{}, Receive: []RoleCard{}}
		otherSpare := spareRoleCards(other.Cards)
		for _, c := range self.Cards {
			if spare, ok := otherSpare[c.Name]; ok && c.missing() {
				candidate.Receive = append(candidate.Receive, spare)
			}
		}
		for _, c := range other.Cards {
			if spare, ok := selfSpare[c.Name]; ok && c.missing() {
				candidate.Give = append(candidate.Give, spare)
			}
		}
		if len(candidate.Give) > 0 && len(candidate.Receive) > 0 {
			candidate.Score = len(candidate.Give) + len(candidate.Receive)
			out = append(out, candidate)
		}
	}
	slices.SortFunc(out, func(a, b RoleCardMatch) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		return strings.Compare(a.ActorID, b.ActorID)
	})
	return out[:min(8, len(out))]
}
func (a *App) roleCardsQuery(ctx context.Context, client AccountsClient, choice Selection) (QueryResult, []RoleCard, error) {
	result, err := client.Execute(ctx, choice, "genshin.theater", map[string]any{"need_detail": true})
	if err != nil {
		return result, nil, err
	}
	cards, err := parseRoleCards(result.Data)
	return result, cards, err
}
func (a *App) cardsAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if action == "cards.query" {
		result, cards, err := a.roleCardsQuery(ctx, a.accountClient(event), Selection{asText(input["account_ref"]), asText(input["role_ref"])})
		if err != nil {
			return nil, err
		}
		return map[string]any{"cards": cards, "view": roleCardView(result.Role, cards)}, nil
	}
	var q struct {
		Scope   GroupScope `json:"scope"`
		ActorID string     `json:"actor_id"`
		Confirm bool       `json:"confirm"`
	}
	if decodeObject(input, &q) != nil || !q.Scope.valid() {
		return nil, gameError("input_invalid", "请提供完整群身份。")
	}
	if action == "cards.group.remove" {
		if !q.Confirm || q.ActorID == "" {
			return nil, gameError("input_invalid", "请确认移除此成员的交换记录。")
		}
		err := a.Groups.Update(q.Scope, func(g *GroupData) error {
			g.Cards = slices.DeleteFunc(g.Cards, func(v RoleCardOffer) bool { return v.ActorID == q.ActorID })
			return nil
		})
		return map[string]any{"removed": err == nil}, err
	}
	data, err := a.Groups.Read(q.Scope)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	items := slices.DeleteFunc(slices.Clone(data.Cards), func(v RoleCardOffer) bool { return v.SavedMS <= now-int64(30*24*time.Hour/time.Millisecond) })
	return map[string]any{"items": items}, nil
}
func (a *App) cardsCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	group := event.Event.Target.Type == "group"
	if command == "role-cards-exchange" && !group {
		return event.SendText("「月谕圣牌交换」仅支持在群聊中查询。")
	}
	if len(args) > 1 {
		return event.SendText("可提供本人UID选择角色。")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	result, cards, err := a.roleCardsQuery(ctx, client, choice)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	now := time.Now().UnixMilli()
	self := RoleCardOffer{ActorID: event.Event.Actor.ID, Nickname: plainGameText(event.Event.Actor.Nickname), UID: result.Role.UID, Region: result.Role.Region, Cards: cards, SavedMS: now}
	scope := groupScope(event)
	// Viewing or exchanging in a group enters the caller into that group's
	// exchange matching for 30 days, as upstream does.
	if group {
		err = a.Groups.Update(scope, func(g *GroupData) error {
			g.Cards = slices.DeleteFunc(g.Cards, func(v RoleCardOffer) bool {
				return v.ActorID == self.ActorID || v.SavedMS <= now-int64(30*24*time.Hour/time.Millisecond)
			})
			if len(g.Cards) >= 2000 {
				return gameError("cards_limit", "本群交换记录达到2000人上限。")
			}
			g.Cards = append(g.Cards, self)
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
	}
	if command == "role-cards" {
		view := roleCardView(result.Role, cards)
		if group {
			view.Note += "\n已记入本群月谕圣牌交换匹配，有效 30 天，再次查看时刷新。"
		}
		state := asObject(result.Data["tarot_card_state"])
		view.Image = a.roleCardsDrawn(ctx, RoleCardsImage{UID: result.Role.UID, Collected: Int(state["curr_num"]), Total: Int(state["total_num"]), Cards: cards})
		return a.sendView(ctx, event, view)
	}
	// miao answers these before looking for members to trade with.
	if !slices.ContainsFunc(cards, RoleCard.missing) {
		return event.SendText("你的「月谕圣牌」已全部收集，无需交换。")
	}
	if len(spareRoleCards(cards)) == 0 {
		return event.SendText("你当前没有可用于交换的重复「月谕圣牌」。")
	}
	data, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	matches := roleCardMatches(self, data.Cards, now)
	if len(matches) == 0 {
		return event.SendText("本群暂无可互相交换的「月谕圣牌」用户。\n提示：需要群友先查询过 " + a.Game.Prefix + "月谕圣牌，才会进入本群交换匹配范围。")
	}
	view := View{Title: "月谕圣牌交换匹配", Subtitle: result.Role.UID, Note: "仅列双方互补且同区服的记录；数据最多保留30天。请自行与对方确认，本插件不执行赠送。"}
	for _, m := range matches {
		view.Sections = append(view.Sections, Section{Title: m.Nickname + " · " + m.ActorID + " · " + m.UID, Rows: []Row{{"你可给对方", roleCardNames(m.Give)}, {"对方可给你", roleCardNames(m.Receive)}, {"对方更新时间", calendarMS(m.SavedMS)}}})
	}
	view.Image = a.roleCardsDrawn(ctx, RoleCardsImage{UID: result.Role.UID, Exchange: true, Group: event.Event.Target.ID, Matches: matches})
	return a.sendView(ctx, event, view)
}

func roleCardNames(cards []RoleCard) string {
	names := []string{}
	for _, c := range cards {
		names = append(names, c.Name)
	}
	return strings.Join(names, "、")
}

// roleCardsDrawn draws a role card page with the plugin's template, if any.
func (a *App) roleCardsDrawn(ctx context.Context, page RoleCardsImage) *Image {
	if a.roleCardsImage == nil {
		return nil
	}
	drawn, ok := a.roleCardsImage(a.imageContext(ctx), page)
	if !ok {
		return nil
	}
	return &drawn
}
