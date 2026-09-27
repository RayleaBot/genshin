package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

// A gacha link pasted in chat is read as Yunzai's gcLog reads it: its
// authkey fetches the records from the official wish history page, the UID
// comes from the records and becomes the sender's current UID, and in a group
// the sender is asked to recall the link. No account is needed. The authkey
// is held in memory only while the records are fetched; a history too long
// for one event is finished as a chat task, which is dropped if the plugin
// restarts.

const (
	gachaLinkCN = "https://public-operation-hk4e.mihoyo.com/gacha_info/api/getGachaLog"
	gachaLinkOS = "https://public-operation-hk4e-sg.hoyoverse.com/gacha_info/api/getGachaLog"
	// gachaLinkTask prefixes the scheduled tasks that finish long histories.
	gachaLinkTask = "game.link."
)

// gachaLinkPools are the pools Yunzai reports after a link, in its order;
// 400 is merged into 301.
var gachaLinkPools = [][2]string{{"301", "角色"}, {"302", "武器"}, {"500", "集录"}, {"200", "常驻"}, {"100", "新手"}}

type gachaLink struct {
	key, region string
}

// gachaLinkJob is a link whose records are still being fetched, as a chat
// task: full reads the whole history, before counts the records each pool
// held, owner is the sender and images whether the reply draws the record.
type gachaLinkJob struct {
	link     gachaLink
	full     bool
	uid      string
	sync     string
	sequence int
	before   map[string]int
	owner    Subject
	images   bool
	// result is the finished sync, kept while its reply waits for the next
	// trigger.
	result *gacha.ImportResult
}

// parseGachaLink reads a wish history link as Yunzai's dealUrl does: 〈= is a
// mangled &lang=, only the query after getGachaLog? or index.html? counts
// and the page fragment (#/log) is cut off. A Star Rail or Zenless link
// is left to its own plugin and a customer service link to 充值记录, as
// Yunzai's payLog takes those first; ok is false when the text holds no
// Genshin wish history link.
func parseGachaLink(text string) (link gachaLink, ok bool, err error) {
	if !strings.Contains(text, "authkey=") || strings.Contains(text, "hkrpg") || strings.Contains(text, "/nap/") || strings.Contains(text, "nap_") {
		return gachaLink{}, false, nil
	}
	for _, service := range []string{"user-game-search", "bill-record-user", "customer-claim", "player-log", "user.mihoyo.com"} {
		if strings.Contains(text, service) {
			return gachaLink{}, false, nil
		}
	}
	text = strings.TrimSpace(strings.ReplaceAll(text, "〈=", "&"))
	for _, marker := range []string{"getGachaLog?", "index.html?"} {
		if _, after, found := strings.Cut(text, marker); found {
			text = after
		}
	}
	if _, after, found := strings.Cut(text, "?"); found && !strings.HasPrefix(text, "authkey") {
		text = after
	}
	text, _, _ = strings.Cut(text, "#")
	params, _ := url.ParseQuery(text)
	key := params.Get("authkey")
	if len(key) < 8 || len(key) > 16384 || strings.ContainsAny(key, " \r\n\t") {
		return gachaLink{}, true, gameError("gacha_link_invalid", "链接复制错误")
	}
	return gachaLink{key: key, region: params.Get("region")}, true, nil
}

// gachaLinkPage reads one page of a pool with the link's authkey. The
// official retcodes get Yunzai's replies.
func (a *App) gachaLinkPage(ctx context.Context, link gachaLink, pool, endID string, page, size int) (map[string]any, error) {
	endpoint := gachaLinkCN
	if !slices.Contains([]string{"cn_gf01", "cn_qd01"}, link.region) {
		endpoint = gachaLinkOS
	}
	query := url.Values{"authkey_ver": {"1"}, "lang": {"zh-cn"}, "authkey": {link.key}, "region": {link.region}, "gacha_type": {pool}, "page": {strconv.Itoa(page)}, "size": {strconv.Itoa(size)}, "end_id": {endID}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := a.LinkHTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, gameError("gacha_link_unavailable", "获取数据错误")
	}
	defer response.Body.Close()
	var envelope struct {
		RetCode int            `json:"retcode"`
		Data    map[string]any `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil || response.StatusCode != http.StatusOK || json.Unmarshal(body, &envelope) != nil {
		return nil, gameError("gacha_link_unavailable", "获取数据错误")
	}
	switch envelope.RetCode {
	case 0:
		return envelope.Data, nil
	case -109:
		return nil, gameError("gacha_link_feedback", "2.3版本后，反馈的链接已无法查询！请用安卓方式获取链接")
	case -101:
		return nil, gameError("gacha_link_expired", "该链接已失效，请重新进入游戏，重新复制链接")
	case -100:
		return nil, gameError("gacha_link_incomplete", "链接不完整，请长按全选复制全部内容（可能输入法复制限制），或者复制的不是历史记录页面链接")
	default:
		return nil, gameError("gacha_link_invalid", "链接复制错误")
	}
}

// checkGachaLink finds the link's region when it names none, as Yunzai tries
// the mainland server and then America, and takes the UID from the first
// record of any pool.
func (a *App) checkGachaLink(ctx context.Context, link *gachaLink, length int) (string, error) {
	regions := []string{link.region}
	if link.region == "" {
		regions = []string{"cn_gf01", "os_usa"}
	}
	var failure error
	for _, region := range regions {
		link.region = region
		for _, pool := range gachaLinkPools {
			data, err := a.gachaLinkPage(ctx, *link, pool[0], "0", 1, 6)
			if err != nil {
				var public *rayleabot.ActionError
				if errors.As(err, &public) && public.Code == "plugin.game_gacha_link_incomplete" && length == 1000 {
					return "", gameError("gacha_link_incomplete", "输入法限制，链接复制不完整，请更换输入法复制完整链接")
				}
				failure = err
				break
			}
			if region := asText(data["region"]); region != "" {
				link.region = region
			}
			if list := asList(data["list"]); len(list) > 0 {
				return asText(asObject(list[0])["uid"]), nil
			}
		}
	}
	if failure != nil {
		if len(regions) > 1 {
			return "", gameError("gacha_link_expired", "链接复制错误或已失效")
		}
		return "", failure
	}
	return "", gameError("gacha_link_empty", "暂无数据，请等待记录后再查询")
}

// gachaLinkMessage answers a message holding a gacha link; handled is false
// for any other message.
func (a *App) gachaLinkMessage(ctx context.Context, event *rayleabot.EventContext) (handled bool, err error) {
	text := event.Event.Message.PlainText
	link, ok, err := parseGachaLink(text)
	if !ok {
		return false, nil
	}
	group := event.Event.Target.Type == "group"
	if group {
		config, configErr := a.Groups.Config(groupScope(event))
		if configErr == nil && config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	if err != nil {
		return true, event.SendText(friendlyError(err))
	}
	start := a.now()
	ctx, cancel := a.eventWork(ctx, start)
	defer cancel()
	uid, err := a.checkGachaLink(ctx, &link, len([]rune(text)))
	if err != nil {
		return true, event.SendText(friendlyError(err))
	}
	notice(ctx, event, "链接发送成功，数据获取中……")
	a.useLinkUID(ctx, event, uid)
	owner := chatOwner(event)
	job := &gachaLinkJob{link: link, full: a.fullLinks.active(owner), uid: uid, before: gachaPoolCounts(a.archiveOrEmpty(uid, link.region)), owner: owner, images: settings(event).ImageReplies}
	if job.full {
		notice(ctx, event, "开始获取角色记录，全量更新获取数据较多，请耐心等待...")
	}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{Link: true}, uid, link.region, job.full)
	if err != nil {
		return true, event.SendText(friendlyError(syncError(err)))
	}
	job.sync = info.Ref
	task := a.beginChatTask(event, gachaLinkTask+rand.Text(), a.Game.Name+"抽卡链接记录", 15*time.Minute, job)
	reply, done, err := a.stepChatTask(ctx, event.Actions(), task, start.Add(chatTaskBudget))
	switch {
	case err != nil:
		reply = textReply("记录较多，本次未能全部获取，请稍后重新发送链接。")
	case !done:
		reply = textReply("记录较多，将在后台继续获取，完成后在此回复。")
	}
	if group {
		reply = append(reply, []rayleabot.Segment{rayleabot.Text("已收到链接，请撤回")})
	}
	return true, answerChat(ctx, event, reply)
}

// step fetches the link's pages until stop and answers, once all are read,
// as Yunzai does, or with the failure. Drawing the reply has the rest of an
// event: when the reads end at stop, the next trigger draws it.
func (job *gachaLinkJob) step(ctx context.Context, a *App, host taskHost, stop time.Time) ([][]rayleabot.Segment, bool) {
	if job.result == nil {
		result, err := a.stepGachaLink(ctx, job, stop)
		switch {
		case err != nil:
			return job.fail(a, err), true
		case result == nil:
			return nil, false
		}
		job.result = result
	}
	if !a.now().Before(stop) {
		return nil, false
	}
	return a.gachaLinkReply(ctx, host, job), true
}

// fail cancels the link's sync and replies with the failure.
func (job *gachaLinkJob) fail(a *App, err error) [][]rayleabot.Segment {
	_ = a.Syncs.Cancel(job.sync)
	return textReply(friendlyError(syncError(err)))
}

// stepGachaLink fetches pages until the sync completes or stop passes; the
// result is nil while pages remain. A page open when ctx ends is fetched
// again by the next trigger.
func (a *App) stepGachaLink(ctx context.Context, job *gachaLinkJob, stop time.Time) (*gacha.ImportResult, error) {
	for {
		info, err := a.Syncs.Step(ctx, a.Gacha, job.sync, job.sequence, func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
			data, err := a.gachaLinkPage(ctx, job.link, pool, endID, page, 20)
			if err != nil {
				return gacha.RemotePage{}, err
			}
			return gacha.ParsePage(job.uid, job.link.region, pool, endID, data)
		})
		switch {
		case err != nil && ctx.Err() != nil:
			return nil, nil
		case err != nil:
			return nil, err
		}
		job.sequence = info.Sequence
		if info.State == "completed" {
			return info.Result, nil
		}
		if !a.now().Before(stop) || a.sleep(ctx, 300*time.Millisecond) != nil {
			return nil, nil
		}
	}
}

// gachaLinkReply reports each pool as Yunzai does, then draws the character
// event wish record.
func (a *App) gachaLinkReply(ctx context.Context, host imageRenderer, job *gachaLinkJob) [][]rayleabot.Segment {
	archive, err := a.Gacha.Read(job.result.UID, job.result.Region)
	if err != nil {
		return textReply(friendlyError(err))
	}
	reply := textReply(gachaLinkSummary(a.Game.Prefix, job.before, gachaPoolCounts(archive)))
	// As upstream, a whole read turns the setting off and says so.
	if job.full {
		a.fullLinks.set(job.owner, false)
		reply = append(reply, []rayleabot.Segment{rayleabot.Text("已关闭全量更新抽卡记录")})
	}
	view := GachaView(a.Game, archive)
	if a.gacha != nil {
		if drawn, ok := a.gacha(a.imageContext(ctx), GachaImage{UID: job.result.UID, Role: Role{Game: a.Game.ID, UID: job.result.UID, Region: job.result.Region}, Word: "角色记录", Archive: archive}); ok {
			view.Image = &drawn
		}
	}
	return append(reply, viewMessage(ctx, host, job.images, view))
}

// fullLinks are Yunzai's 设置全量更新抽卡记录: for ten minutes the sender's
// links read the whole history again rather than only the new records, to
// mend records the official history still holds.
type fullLinks struct {
	mu    sync.Mutex
	until map[Subject]time.Time
}

func (f *fullLinks) set(owner Subject, on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.until == nil {
		f.until = map[Subject]time.Time{}
	}
	delete(f.until, owner)
	if on {
		f.until[owner] = time.Now().Add(10 * time.Minute)
	}
}

func (f *fullLinks) active(owner Subject) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return time.Now().Before(f.until[owner])
}

// fullLinkCommand turns 设置全量更新抽卡记录 on, or off with 关 or off.
func (a *App) fullLinkCommand(event *rayleabot.EventContext) error {
	word := event.Event.Command() + strings.Join(event.Event.Args(), "")
	on := !strings.Contains(word, "关") && !strings.Contains(word, "off")
	a.fullLinks.set(chatOwner(event), on)
	if on {
		return event.SendText("已开启全量更新抽卡记录，在10分钟内您的首次抽卡记录将全量更新，用于修复在官方记录有效期内可能发生的数据错误")
	}
	return event.SendText("已关闭全量更新抽卡记录")
}

// gachaLinkSummary is Yunzai's reply after a link: the records each pool
// gained, then the record commands to try.
func gachaLinkSummary(prefix string, before, after map[string]int) string {
	lines := []string{}
	for _, pool := range gachaLinkPools {
		if after[pool[0]] > 0 {
			lines = append(lines, "["+pool[1]+"]记录获取成功，更新"+strconv.Itoa(after[pool[0]]-before[pool[0]])+"条")
		}
	}
	lines = append(lines, "", "抽卡记录更新完成，您还可回复", "【"+prefix+"全部记录】统计全部抽卡数据", "【"+prefix+"武器记录】统计武器池数据", "【"+prefix+"角色统计】按卡池统计数据", "【"+prefix+"导出记录】导出记录数据")
	return strings.Join(lines, "\n")
}

// gachaHelpPort answers 电脑帮助, 苹果帮助 and 安卓帮助 as Yunzai's helpPort:
// the platform's guide image, or for Android the online guide.
func (a *App) gachaHelpPort(event *rayleabot.EventContext) error {
	switch strings.TrimSuffix(strings.ToLower(event.Event.Command()), "帮助") {
	case "安卓":
		return event.SendText("安卓抽卡记录获取教程：https://docs.qq.com/doc/DUWpYaXlvSklmVXlX")
	case "电脑", "pc":
		return a.staticCommand(event, StaticPicture{Source: "yunzai-genshin", Path: "resources/logHelp/记录帮助-电脑.png"})
	default:
		return a.staticCommand(event, StaticPicture{Source: "yunzai-genshin", Path: "resources/logHelp/记录帮助-苹果.png"})
	}
}

// gachaPoolCounts counts an archive's records by the pools Yunzai reports.
func gachaPoolCounts(archive gacha.Archive) map[string]int {
	counts := map[string]int{}
	for _, record := range archive.Records {
		pool := gacha.Pool(record.GachaType)
		if pool == "400" {
			pool = "301"
		}
		counts[pool]++
	}
	return counts
}

// chatArchive is the archive a record command reads: as upstream, the log of
// the UID named or of the UID in use, bound with or without an account.
func (a *App) chatArchive(ctx context.Context, event *rayleabot.EventContext, uid string) (gacha.Archive, Role, error) {
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return gacha.Archive{}, Role{}, err
	}
	role := owner.Role
	if !owner.Owned {
		role = Role{Game: a.Game.ID, UID: owner.UID}
		if summaries, err := a.Gacha.List(); err == nil {
			for _, summary := range summaries {
				if summary.UID == owner.UID {
					role.Region = summary.Region
					break
				}
			}
		}
	}
	archive, err := a.Gacha.Read(role.UID, role.Region)
	if err != nil {
		return gacha.Archive{}, role, a.noGachaRecords()
	}
	return archive, role, nil
}

// noGachaRecords is Yunzai's reply for a UID without wish records.
func (a *App) noGachaRecords() error {
	return gameError("archive_missing", "暂无抽卡记录\n"+a.Game.Prefix+"记录帮助，查看配置说明")
}

func (a *App) archiveOrEmpty(uid, region string) gacha.Archive {
	archive, _ := a.Gacha.Read(uid, region)
	return archive
}

// useLinkUID makes the link's UID the sender's current UID, as upstream sets
// it after a link; without the account plugin the link still works.
func (a *App) useLinkUID(ctx context.Context, event *rayleabot.EventContext, uid string) {
	client := a.accountClient(event)
	if listed, err := client.List(ctx, 0); err == nil {
		_ = useUID(ctx, client, listed, a.Game.ID, uid)
	}
}

// syncError gives the sync engine's failures their public codes.
func syncError(err error) error {
	switch {
	case errors.Is(err, gacha.ErrSync):
		return gameError("sync_expired", "同步任务已取消或过期，请重新开始。")
	case errors.Is(err, gacha.ErrConflict):
		return gameError("sync_conflict", "档案已变更或记录冲突，请重新开始同步；现有档案已保留。")
	case errors.Is(err, gacha.ErrInvalid):
		return gameError("sync_invalid", "官方记录结构或分页异常，现有档案已保留。")
	}
	return err
}
