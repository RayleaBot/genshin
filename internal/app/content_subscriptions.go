package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/localdata"
)

// Yunzai's 米游社推送 (mysNews): a group administrator turns on 公告 and 资讯
// pushes and the 到期活动 warning for the group. Every five minutes a group
// gets at most one new post of the last two hours whose subject has none of
// Yunzai's banned words, drawn as mysNews after a line naming it, each post
// once in ten hours; from ten o'clock, once a day, it gets the activities
// ending within a day. 推送公告 runs every group's check at once.

// pushZone is the server time the pushes run on.
var pushZone = time.FixedZone("UTC+8", 8*3600)

// pushBanWords are Yunzai's pushNews banWord for Genshin.
var pushBanWords = regexp.MustCompile(`冒险助力礼包|纪行|预下载|脚本外挂|集中反馈|已开奖|云·原神|魔神任务|传说任务|线下赛|晋级赛|战绩更新|海选赛|邀请赛|积分赛|战绩工具|交流平台|首日赛|线上赛|社区内容|个人专访|全民赛|决赛|总决赛|半决赛|淘汰赛|作品展示|同人|大别野`)

// pushKinds are the pushes a group can turn on, with Yunzai's names; the
// news kinds with their getNewsList type.
var pushKinds = []struct {
	kind, name string
	newsType   int
}{{"announce", "公告", 1}, {"info", "资讯", 3}, {"expiry", "活动到期预警", 0}}

// ContentSubscription is a group's pushes, checked by its scheduled task.
type ContentSubscription struct {
	Ref     string   `json:"ref"`
	Owner   Subject  `json:"owner"`
	GroupID string   `json:"group_id"`
	Kinds   []string `json:"kinds"`
	// Sent are the posts pushed in the last ten hours, as upstream's redis
	// keys, and WarnedOn the day the activities were last warned of.
	Sent     map[string]int64 `json:"sent"`
	WarnedOn string           `json:"warned_on,omitempty"`
	LastCode string           `json:"last_code,omitempty"`
}

type ContentSubscriptions struct {
	mu   sync.Mutex
	Path string
}

func (s *ContentSubscriptions) read() ([]ContentSubscription, error) {
	items := []ContentSubscription{}
	err := localdata.Read(s.Path, &items)
	return items, err
}

func (s *ContentSubscriptions) List() ([]ContentSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *ContentSubscriptions) edit(ref string, fn func(*[]ContentSubscription, int) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(v ContentSubscription) bool { return v.Ref == ref })
	if err = fn(&items, i); err != nil {
		return err
	}
	return localdata.Write(s.Path, items)
}

// pushPost is a post of getNewsList with the push it belongs to.
type pushPost struct {
	id, subject, kind string
	created           int64
}

// postToPush is the post a group gets now: the first by ID of those created
// in the last two hours, free of banned words, of a kind the group has on
// and not pushed to it in ten hours.
func postToPush(posts []pushPost, sub ContentSubscription, now time.Time) (pushPost, bool) {
	slices.SortStableFunc(posts, func(a, b pushPost) int {
		if len(a.id) != len(b.id) {
			return len(a.id) - len(b.id)
		}
		return strings.Compare(a.id, b.id)
	})
	for _, post := range posts {
		sent, seen := sub.Sent[post.id]
		if slices.Contains(sub.Kinds, post.kind) && now.Unix()-post.created <= 7200 && !pushBanWords.MatchString(post.subject) && (!seen || now.UnixMilli()-sent >= int64(10*time.Hour/time.Millisecond)) {
			return post, true
		}
	}
	return pushPost{}, false
}

// expiringActivity is an activity announcement ending soon.
type expiringActivity struct {
	title, subtitle, banner, end string
	left                         time.Duration
}

// expiringActivities are Yunzai's getGsActivity: the second announcement
// group's activities, not 传说任务 or 游戏公告, with a day or less left.
func expiringActivities(groups []any, now time.Time) []expiringActivity {
	if len(groups) < 2 {
		return nil
	}
	out := []expiringActivity{}
	for _, raw := range asList(asObject(groups[1])["list"]) {
		item := asObject(raw)
		title := asText(item["title"])
		if !strings.Contains(asText(item["tag_label"]), "活动") || strings.Contains(title, "传说任务") || strings.Contains(title, "游戏公告") {
			continue
		}
		end, err := time.ParseInLocation("2006-01-02 15:04:05", asText(item["end_time"]), pushZone)
		if err != nil {
			continue
		}
		left := end.Sub(now)
		if math.Floor(left.Hours()/24) <= 1 {
			out = append(out, expiringActivity{title, asText(item["subtitle"]), asText(item["banner"]), asText(item["end_time"]), left})
		}
	}
	return out
}

// pushLists hold the news lists and announcements a round of checks reads,
// for a minute, so groups checked together share them.
type pushLists struct {
	mu   sync.Mutex
	at   time.Time
	data map[string]any
}

func (p *pushLists) get(key string, read func() (any, error)) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.at) > time.Minute {
		p.at, p.data = time.Now(), map[string]any{}
	}
	if value, ok := p.data[key]; ok {
		return value, nil
	}
	value, err := read()
	if err == nil {
		p.data[key] = value
	}
	return value, err
}

// pushGroup runs a group's check: the post it gets and the activity
// warnings, sent through send.
func (a *App) pushGroup(ctx context.Context, event *rayleabot.EventContext, sub *ContentSubscription, now time.Time, send func(...rayleabot.Segment) error) error {
	posts := []pushPost{}
	for _, kind := range pushKinds {
		if kind.newsType == 0 || !slices.Contains(sub.Kinds, kind.kind) {
			continue
		}
		raw, err := a.pushLists.get(kind.kind, func() (any, error) {
			return a.Content.get(ctx, "https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?"+url.Values{"gids": {bbsGID}, "page_size": {"10"}, "type": {strconv.Itoa(kind.newsType)}}.Encode(), nil)
		})
		if err != nil {
			return err
		}
		for _, item := range asList(asObject(raw)["list"]) {
			post := asObject(asObject(item)["post"])
			created, _ := strconv.ParseInt(asText(post["created_at"]), 10, 64)
			posts = append(posts, pushPost{asText(post["post_id"]), asText(post["subject"]), kind.kind, created})
		}
	}
	for id, at := range sub.Sent {
		if now.UnixMilli()-at >= int64(10*time.Hour/time.Millisecond) {
			delete(sub.Sent, id)
		}
	}
	if post, ok := postToPush(posts, *sub, now); ok {
		if sub.Sent == nil {
			sub.Sent = map[string]int64{}
		}
		sub.Sent[post.id] = now.UnixMilli()
		name := "公告"
		if post.kind == "info" {
			name = "资讯"
		}
		if err := a.pushPost(ctx, event, post.id, newsGameName+name+"推送："+post.subject, send); err != nil {
			if ctx.Err() != nil {
				// The event ended before the post was sent; the next
				// trigger pushes it.
				delete(sub.Sent, post.id)
			}
			return err
		}
	}
	today := now.In(pushZone).Format("2006-01-02")
	if slices.Contains(sub.Kinds, "expiry") && now.In(pushZone).Hour() >= 10 && sub.WarnedOn != today {
		warned := sub.WarnedOn
		sub.WarnedOn = today
		raw, err := a.pushLists.get("announcements", func() (any, error) { return a.Content.announcements(ctx) })
		if err != nil {
			if ctx.Err() != nil {
				// The event ended before the activities were read; the
				// next trigger warns.
				sub.WarnedOn = warned
			}
			return err
		}
		for _, activity := range expiringActivities(raw.(Announcements).Groups, now) {
			left := activity.left.Milliseconds()
			floor := func(value, unit int64) int64 { return int64(math.Floor(float64(value) / float64(unit))) }
			remaining := fmt.Sprintf("%d天%d小时%d分钟%d秒", floor(left, 86400000), floor(left%86400000, 3600000), floor(left%3600000, 60000), floor(left%60000, 1000))
			segments := []rayleabot.Segment{rayleabot.Text("【" + newsGameName + "活动即将结束通知】\n活动:" + activity.subtitle)}
			if banner := publicImage(activity.banner); banner != "" {
				segments = append(segments, rayleabot.Image(banner))
			}
			segments = append(segments, rayleabot.Text("描述:"+activity.title+"\n活动剩余时间:"+remaining+"\n活动结束时间:"+activity.end))
			if err := send(segments...); err != nil {
				return err
			}
		}
	}
	return nil
}

// pushPost draws a post after its line, or sends the line and the link when
// it cannot be drawn.
func (a *App) pushPost(ctx context.Context, event *rayleabot.EventContext, id, title string, send func(...rayleabot.Segment) error) error {
	data, err := a.Content.get(ctx, "https://bbs-api.miyoushe.com/post/wapi/getPostFull?"+url.Values{"gids": {bbsGID}, "read": {"1"}, "post_id": {id}}.Encode(), nil)
	if err != nil {
		return err
	}
	full := asObject(data["post"])
	if asText(asObject(full["post"])["post_id"]) != id {
		return gameError("public_invalid", "帖子编号与请求不一致。")
	}
	if settings(event).ImageReplies {
		image := a.newsPostImage(ctx, full)
		result, err := event.Actions().RenderImage(ctx, rayleabot.RenderImageRequest{Template: image.Template, Output: "jpeg", FallbackText: title, Data: image.Data, Resources: image.Resources})
		if path := asText(result["image_path"]); err == nil && path != "" {
			return send(rayleabot.Text(title), rayleabot.Image(path))
		}
	}
	return send(rayleabot.Text(title + "\nhttps://www.miyoushe.com/" + bbsPath + "/article/" + id))
}

// checkPushes runs the checks of the given subscriptions, or of all when ref
// is empty, sending to each group through its own bot.
func (a *App) checkPushes(ctx context.Context, event *rayleabot.EventContext, ref string) error {
	items, err := a.Subscriptions.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if ref != "" && item.Ref != ref {
			continue
		}
		code, online := "checked", false
		for _, bot := range event.Bots {
			online = online || bot.ID == item.Owner.BotID && bot.SourceAdapter == item.Owner.SourceAdapter && bot.SourceProtocol == item.Owner.SourceProtocol
		}
		if !online {
			code = "bot_missing"
		} else if config, err := a.Groups.Config(GroupScope{item.Owner.SourceProtocol, item.Owner.SourceAdapter, item.Owner.BotID, item.GroupID}); err != nil || config.Enabled != nil && !*config.Enabled {
			code = "group_disabled"
		} else {
			send := func(segments ...rayleabot.Segment) error {
				_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: item.Owner.SourceProtocol, SourceAdapter: item.Owner.SourceAdapter, TargetType: "group", TargetID: item.GroupID, Message: rayleabot.MessageOut{Segments: segments}})
				return err
			}
			if err := a.pushGroup(ctx, event, &item, time.Now(), send); err != nil {
				if ctx.Err() != nil {
					err = errStepUnfinished
				}
				code = PublicError(err).Code
			}
		}
		err := a.Subscriptions.edit(item.Ref, func(items *[]ContentSubscription, i int) error {
			if i >= 0 {
				(*items)[i].Sent, (*items)[i].WarnedOn, (*items)[i].LastCode = item.Sent, item.WarnedOn, code
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) subscriptionManage(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if action == "content.subscription.list" {
		items, err := a.Subscriptions.List()
		return map[string]any{"items": items}, err
	}
	if action != "content.subscription.remove" || input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认移除此推送。")
	}
	ref := asText(input["ref"])
	err := a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
		if i < 0 {
			return gameError("subscription_missing", "推送已不存在。")
		}
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
	if err == nil {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
	}
	return map[string]any{"removed": err == nil}, err
}

// subscriptionCommand is Yunzai's setPush and setActivityPush: a group
// administrator turns a push on or off for the group.
func (a *App) subscriptionCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if command == "content-push" {
		// As upstream, the pushes are the only answer.
		if err := a.checkPushes(ctx, event, ""); err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.Result(map[string]any{"handled": true})
	}
	if event.Event.EventType != "message.group" {
		return event.SendText("推送请在群聊中设置")
	}
	if !groupAdministrator(event) {
		return event.SendText("暂无权限，只有管理员才能操作")
	}
	kind := pushKinds[2]
	if len(args) > 0 && args[0] == "资讯" {
		kind = pushKinds[1]
	} else if len(args) > 0 && args[0] == "公告" {
		kind = pushKinds[0]
	}
	on := command == "subscribe"
	owner := Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
	identity := strings.Join([]string{owner.SourceProtocol, owner.SourceAdapter, owner.BotID, event.Event.Target.ID}, "\x00")
	sum := sha256.Sum256([]byte(identity))
	ref := "game.content." + a.Game.ID + "." + hex.EncodeToString(sum[:16])
	created, emptied := false, false
	err := a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
		if i < 0 {
			if !on {
				return nil
			}
			*items = append(*items, ContentSubscription{Ref: ref, Owner: owner, GroupID: event.Event.Target.ID, Sent: map[string]int64{}})
			i, created = len(*items)-1, true
		}
		item := &(*items)[i]
		item.Kinds = slices.DeleteFunc(item.Kinds, func(k string) bool { return k == kind.kind })
		if on {
			item.Kinds = append(item.Kinds, kind.kind)
		} else if len(item.Kinds) == 0 {
			*items, emptied = slices.Delete(*items, i, i+1), true
		}
		return nil
	})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if created {
		if _, err := event.Actions().SchedulerCreate(ctx, a.contentJob(ref)); err != nil {
			_ = a.Subscriptions.edit(ref, func(items *[]ContentSubscription, i int) error {
				if i >= 0 {
					*items = slices.Delete(*items, i, i+1)
				}
				return nil
			})
			return event.SendText(friendlyError(err))
		}
	}
	if emptied {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
	}
	if kind.kind == "expiry" {
		if on {
			return event.SendText(newsGameName + "活动到期预警推送已开启\n如有即将到期的活动将自动推送至此")
		}
		return event.SendText(newsGameName + "活动到期预警推送已关闭")
	}
	if on {
		return event.SendText(newsGameName + kind.name + "推送已开启\n如有最新" + kind.name + "将自动推送至此")
	}
	return event.SendText(newsGameName + kind.name + "推送已关闭")
}

// contentJob is the scheduler job of a group's push ref.
func (a *App) contentJob(ref string) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + "米游社推送"}
}

// runContentSubscription checks a group on its scheduled task; a task whose
// group turned every push off is removed.
func (a *App) runContentSubscription(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ctx, cancel := a.eventWork(ctx, a.now())
	defer cancel()
	ref := event.Event.TaskID()
	items, err := a.Subscriptions.List()
	if err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	if !slices.ContainsFunc(items, func(item ContentSubscription) bool { return item.Ref == ref }) {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
		return event.Result(map[string]any{"checked": false})
	}
	if err := a.checkPushes(ctx, event, ref); err != nil {
		return event.Fail(PublicError(err).Code, PublicError(err).Message)
	}
	return event.Result(map[string]any{"checked": true})
}
