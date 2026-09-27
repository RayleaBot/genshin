package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
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

// Yunzai's 充值记录 (payLog): a customer service link sent in private reads
// the Genesis Crystals bought and the 680 primogems of each Gnostic Hymn with
// its authkey, and draws them by month and pack. The result is kept for the
// sender; the authkey only in memory, for a day, so 更新充值记录 can read
// again, as upstream keeps it a day in redis. Without a link 更新充值记录
// reads the same logs with the account's SToken, as xiaoyao's 刷新充值记录
// hands Yunzai a customer service authkey; the accounts plugin reads them
// and the authkey stays there. A history too long for one event is finished
// as a chat task, dropped if the plugin restarts.

const (
	payLogURL  = "https://hk4e-api.mihoyo.com/common/hk4e_self_help_query/User/"
	payLogTask = "game.pay."
)

var (
	// payLinkServices mark a customer service link, as in Yunzai's rule.
	payLinkServices = regexp.MustCompile(`user-game-search|bill-record-user|customer-claim|player-log|user\.mihoyo\.com`)
	payLinkKey      = regexp.MustCompile(`&authkey=([^&\s\p{Han}]+)`)
	// payPrices are the crystals of Yunzai's packs in its order: the Gnostic
	// Hymn's 680 primogems, the Blessing of the Welkin Moon, then 648 down
	// to 6 yuan; payDoubles the first top-up's doubled crystals.
	payPrices  = [8]int{680, 300, 8080, 3880, 2240, 1090, 330, 60}
	payDoubles = [8]int{0, 0, 12960, 6560, 3960, 1960, 600, 120}
)

// PayLog is Yunzai's filtrateData: the crystals bought and each month's
// packs, counted in payPrices' order.
type PayLog struct {
	UID     string     `json:"uid"`
	Crystal int        `json:"crystal"`
	Months  []PayMonth `json:"months"`
}

type PayMonth struct {
	Month  string `json:"month"`
	Counts [8]int `json:"counts"`
}

// PayLogImageBuilder draws a PayLog with the plugin's template.
type PayLogImageBuilder func(ImageContext, PayLog) (Image, bool)

// PayLogStore keeps each sender's results by UID, in the order first read.
type PayLogStore struct {
	Path string
	mu   sync.Mutex
}

type payLogOwner struct {
	Owner Subject  `json:"owner"`
	Logs  []PayLog `json:"logs"`
}

func (s *PayLogStore) Get(owner Subject) ([]PayLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []payLogOwner{}
	if err := localdata.Read(s.Path, &items); err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Owner == owner {
			return item.Logs, nil
		}
	}
	return nil, nil
}

// Keep replaces the sender's result for the UID, or adds it.
func (s *PayLogStore) Keep(owner Subject, log PayLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []payLogOwner{}
	if err := localdata.Read(s.Path, &items); err != nil {
		return err
	}
	index := slices.IndexFunc(items, func(item payLogOwner) bool { return item.Owner == owner })
	if index < 0 {
		items = append(items, payLogOwner{Owner: owner})
		index = len(items) - 1
	}
	logs := items[index].Logs
	if at := slices.IndexFunc(logs, func(kept PayLog) bool { return kept.UID == log.UID }); at >= 0 {
		logs[at] = log
	} else {
		logs = append(logs, log)
	}
	items[index].Logs = logs
	return localdata.Write(s.Path, items)
}

// payKeys hold each sender's latest link authkey and its UID for a day.
type payKeys struct {
	mu   sync.Mutex
	keys map[Subject]payKey
}

type payKey struct {
	uid, key string
	expires  time.Time
}

func (k *payKeys) put(owner Subject, uid, key string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.keys == nil {
		k.keys = map[Subject]payKey{}
	}
	k.keys[owner] = payKey{uid, key, time.Now().Add(24 * time.Hour)}
}

func (k *payKeys) get(owner Subject) (payKey, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	kept, ok := k.keys[owner]
	if ok && time.Now().After(kept.expires) {
		delete(k.keys, owner)
		return payKey{}, false
	}
	return kept, ok
}

// payRecord is a crystal or primogem gain.
type payRecord struct {
	id, at string
	add    int
}

// payLogJob is a link or account whose logs are still being read, as a chat
// task: the crystal log, then the primogem log for 680s. An account read has
// no key: choice and provider pick the role the accounts plugin reads, and
// grant is the delegation its scheduled task reads with, a page a second.
// owner is the sender the result is kept for, and images whether it is
// drawn.
type payLogJob struct {
	key, uid string
	choice   Selection
	provider string
	grant    string
	last     time.Time
	api      string
	cursor   string
	pages    int
	records  []payRecord
	owner    Subject
	images   bool
}

// payLogRequest reads one of Yunzai's self-help queries with an authkey; the
// failures get Yunzai's replies.
func (a *App) payLogRequest(ctx context.Context, key, api, endID string) (map[string]any, error) {
	query := url.Values{"selfquery_type": {"1"}, "sign_type": {"2"}, "auth_appid": {"csc"}, "authkey_ver": {"1"}, "game_biz": {"hk4e_cn"}, "win_direction": {"portrait"}, "bbs_auth_required": {"true"},
		"bbs_game_role_required": {"hk4e_cn"}, "app_client": {"bbs"}, "lang": {"zh-cn"}, "csc_authkey_required": {"true"}, "authkey": {key}, "page_id": {"1"}}
	if api != "GetUserInfo" {
		query.Set("add_type", "produce")
		query.Set("size", "20")
		query.Set("end_id", endID)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, payLogURL+api+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json, text/plain, */*")
	request.Header.Set("Referer", "https://webstatic.mihoyo.com/")
	client := a.LinkHTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, gameError("pay_log_unavailable", "获取失败，请稍后再试")
	}
	defer response.Body.Close()
	var envelope struct {
		RetCode int            `json:"retcode"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil || response.StatusCode != http.StatusOK || json.Unmarshal(body, &envelope) != nil {
		return nil, gameError("pay_log_unavailable", "获取失败，请稍后再试")
	}
	switch {
	case envelope.RetCode == -101:
		return nil, gameError("pay_log_expired", "您的链接过期，请重新获取")
	case envelope.RetCode == -100:
		return nil, gameError("pay_log_invalid", "链接不正确，请重新获取")
	case strings.Contains(envelope.Message, "unknown auth appid"):
		return nil, gameError("pay_log_link", "抽卡或其他链接现已无法获取充值记录，请发送客服页面的链接！")
	case api == "GetUserInfo" && asText(envelope.Data["uid"]) == "":
		return nil, gameError("pay_log_invalid", "获取失败，可能是链接已过期或不正确")
	case api != "GetUserInfo" && envelope.Data["list"] == nil:
		return nil, gameError("pay_log_unavailable", "获取失败，错误码："+strconv.Itoa(envelope.RetCode))
	}
	return envelope.Data, nil
}

// payLogPage reads a page of the log being read, with the link's authkey or
// through the accounts plugin, and the cursor of the next page.
func (a *App) payLogPage(ctx context.Context, client AccountsClient, job *payLogJob) ([]any, string, error) {
	if job.key != "" {
		data, err := a.payLogRequest(ctx, job.key, job.api, job.cursor)
		if err != nil {
			return nil, "", err
		}
		list := asList(data["list"])
		if len(list) == 20 {
			return list, asText(asObject(list[19])["id"]), nil
		}
		return list, "", nil
	}
	end := job.cursor
	if end == "" {
		end = "0"
	}
	params := map[string]any{"account_ref": job.choice.AccountRef, "role_ref": job.choice.RoleRef, "operation": a.Game.ID + ".billing",
		"input": map[string]any{"category": map[string]string{"GetCrystalLog": "crystal", "GetPrimogemLog": "primogem"}[job.api], "direction": "produce", "end_id": end}}
	if job.grant != "" {
		params["delegation_ref"] = job.grant
		if err := a.sleep(ctx, job.last.Add(time.Second).Sub(a.now())); err != nil {
			return nil, "", err
		}
		job.last = a.now()
	}
	client.Provider = job.provider
	var result QueryResult
	if err := client.call(ctx, "execute", params, &result); err != nil {
		return nil, "", err
	}
	if result.Role.UID != job.uid {
		return nil, "", gameError("pay_log_unavailable", "获取失败，请稍后再试")
	}
	next := ""
	if result.Data["has_more"] == true {
		next = asText(result.Data["next_end_id"])
	}
	return asList(result.Data["items"]), next, nil
}

// stepPayLog reads pages until both logs are read or stop passes; done is
// false while pages remain. A page open when ctx ends is read again by the
// next trigger.
func (a *App) stepPayLog(ctx context.Context, client AccountsClient, job *payLogJob, stop time.Time) (bool, error) {
	for job.api != "" {
		if job.pages >= 2000 {
			return false, gameError("pay_log_limit", "记录过多，未能全部获取")
		}
		list, next, err := a.payLogPage(ctx, client, job)
		switch {
		case err != nil && ctx.Err() != nil:
			return false, nil
		case err != nil:
			return false, err
		}
		job.pages++
		for _, raw := range list {
			item := asObject(raw)
			add, _ := strconv.Atoi(asText(item["add_num"]))
			// Of the primogems only the Gnostic Hymn's count.
			if job.api == "GetCrystalLog" || add == 680 {
				job.records = append(job.records, payRecord{asText(item["id"]), asText(item["datetime"]), add})
			}
		}
		switch {
		case next != "" && next != job.cursor:
			job.cursor = next
		case job.api == "GetCrystalLog":
			job.api, job.cursor = "GetPrimogemLog", ""
		default:
			job.api = ""
		}
		if job.api != "" && !a.now().Before(stop) {
			return false, nil
		}
	}
	return true, nil
}

// payLogData is Yunzai's filtrateData over the records, oldest first: each
// month's packs, a month starting where the month changes, and the crystals
// bought, the Gnostic Hymn aside. Upstream clears the first month's counts
// at each of its records but the last; every record counts here.
func payLogData(uid string, records []payRecord) PayLog {
	records = slices.Clone(records)
	slices.SortFunc(records, func(a, b payRecord) int {
		if len(a.id) != len(b.id) {
			return len(a.id) - len(b.id)
		}
		switch {
		case a.id < b.id:
			return -1
		case a.id > b.id:
			return 1
		}
		return 0
	})
	log := PayLog{UID: uid, Months: []PayMonth{}}
	month := 0
	for _, record := range records {
		if record.add <= 0 || len(record.at) < 7 {
			continue
		}
		if current, _ := strconv.Atoi(record.at[5:7]); current != month {
			month = current
			log.Months = append(log.Months, PayMonth{Month: strconv.Itoa(current) + "月"})
		}
		for index := range payPrices {
			if record.add == payPrices[index] || record.add == payDoubles[index] {
				log.Months[len(log.Months)-1].Counts[index]++
				if record.add != 680 {
					log.Crystal += record.add
				}
				break
			}
		}
	}
	return log
}

// payLinkMessage answers a customer service link sent in private; handled is
// false for any other message, and in a group, where upstream ignores it.
func (a *App) payLinkMessage(ctx context.Context, event *rayleabot.EventContext) (bool, error) {
	start := a.now()
	text := event.Event.Message.PlainText
	if event.Event.Target.Type != "private" || !payLinkServices.MatchString(text) {
		return false, nil
	}
	match := payLinkKey.FindStringSubmatch(text)
	if match == nil {
		return true, event.SendText("链接无效,请重新发送")
	}
	key, err := url.QueryUnescape(match[1])
	if err != nil || len(key) > 16384 {
		return true, event.SendText("链接无效,请重新发送")
	}
	notice(ctx, event, "正在获取消费数据,可能需要30s~~")
	return true, a.startPayLog(ctx, event, &payLogJob{key: key}, start)
}

// startPayLog reads the logs of a link or account in an event that started
// at start and answers in the chat, as a chat task when they take longer than
// the event.
func (a *App) startPayLog(ctx context.Context, event *rayleabot.EventContext, job *payLogJob, start time.Time) error {
	ctx, cancel := a.eventWork(ctx, start)
	defer cancel()
	if job.key != "" {
		user, err := a.payLogRequest(ctx, job.key, "GetUserInfo", "")
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		job.uid = asText(user["uid"])
		// As upstream, the link's UID becomes the sender's.
		a.useLinkUID(ctx, event, job.uid)
	}
	job.api, job.owner, job.images = "GetCrystalLog", chatOwner(event), settings(event).ImageReplies
	task := a.beginChatTask(event, payLogTask+rand.Text(), a.Game.Name+"充值记录", "pay_log", 15*time.Minute, job)
	reply, done, err := a.stepChatTask(ctx, event.Actions(), task, start.Add(chatTaskBudget))
	switch {
	case err != nil && job.key != "":
		return event.SendText("记录较多，本次未能全部获取，请稍后重新发送链接。")
	case err != nil:
		return event.SendText("记录较多，本次未能全部获取，请稍后再试。")
	case !done:
		return event.SendText("记录较多，将在后台继续获取，完成后在此回复。")
	}
	return answerChat(ctx, event, reply)
}

// step reads the logs until stop and answers, once both are read, with the
// result drawn as Yunzai does, or with the failure. Drawing and keeping the
// result has the rest of an event: when the reads end at stop, the next
// trigger does it.
func (job *payLogJob) step(ctx context.Context, a *App, host taskHost, stop time.Time) ([][]rayleabot.Segment, bool) {
	done, err := a.stepPayLog(ctx, AccountsClient{Caller: host, Game: a.Game.ID}, job, stop)
	switch {
	case err != nil:
		return job.fail(a, err), true
	case !done || !a.now().Before(stop):
		return nil, false
	}
	return a.payLogReply(ctx, host, job), true
}

// fail replies with the failure.
func (job *payLogJob) fail(_ *App, err error) [][]rayleabot.Segment {
	return textReply(friendlyError(err))
}

// handover grants an account read the delegation its scheduled task reads
// with, which only a chat can grant; it lapses after a day. A link needs
// none.
func (job *payLogJob) handover(ctx context.Context, a *App, host taskHost, ref string) error {
	if job.key != "" {
		return nil
	}
	var grant struct {
		Delegation struct {
			Ref string `json:"ref"`
		} `json:"delegation"`
	}
	client := AccountsClient{Caller: host, Provider: job.provider, Game: a.Game.ID}
	if err := client.call(ctx, "delegation.create", map[string]any{"account_ref": job.choice.AccountRef, "role_ref": job.choice.RoleRef, "task_id": ref, "operation": a.Game.ID + ".billing", "days": 1}, &grant); err != nil {
		return err
	}
	job.grant = grant.Delegation.Ref
	return nil
}

// payLogReply draws the logs read, keeps the result for the sender and a
// link's authkey for a day.
func (a *App) payLogReply(ctx context.Context, host imageRenderer, job *payLogJob) [][]rayleabot.Segment {
	if len(job.records) == 0 {
		return textReply("未获取到您的任何充值数据")
	}
	log := payLogData(job.uid, job.records)
	reply := [][]rayleabot.Segment{a.payLogMessage(ctx, host, job.images, log)}
	if job.key != "" {
		a.PayKeys.put(job.owner, job.uid, job.key)
	}
	if err := a.PayLogs.Keep(job.owner, log); err != nil {
		reply = append(reply, []rayleabot.Segment{rayleabot.Text(friendlyError(err))})
	}
	return reply
}

// payLogMessage draws a result, or writes it out when images are off.
func (a *App) payLogMessage(ctx context.Context, host imageRenderer, images bool, log PayLog) []rayleabot.Segment {
	view := View{Title: a.Game.Name + "充值统计", Subtitle: "UID " + log.UID, Rows: []Row{{Label: "总结晶", Value: strconv.Itoa(log.Crystal)}}}
	for _, month := range log.Months {
		counts := []string{}
		for index, count := range month.Counts {
			if count > 0 {
				counts = append(counts, PayPackNames[index]+"×"+strconv.Itoa(count))
			}
		}
		if len(counts) == 0 {
			counts = append(counts, "无")
		}
		view.Rows = append(view.Rows, Row{Label: month.Month, Value: strings.Join(counts, "、")})
	}
	if a.payLogImage != nil {
		if drawn, ok := a.payLogImage(a.imageContext(ctx), log); ok {
			view.Image = &drawn
		}
	}
	return viewMessage(ctx, host, images, view)
}

// PayPackNames name the packs in payPrices' order, as Yunzai's chart does.
var PayPackNames = [8]string{"大月卡", "小月卡", "648", "328", "198", "98", "30", "6"}

// payLogCommand answers 充值记录 and 更新充值记录: the kept result of the
// sender's current UID, else the first kept, and a new read with the
// authkey kept for that UID, else with the account of the UID.
func (a *App) payLogCommand(ctx context.Context, event *rayleabot.EventContext, command string) error {
	start := a.now()
	owner := chatOwner(event)
	player, _ := a.panelOwner(ctx, event, "")
	current := player.UID
	if command == "pay-log" {
		logs, err := a.PayLogs.Get(owner)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		for _, log := range logs {
			if current == "" || log.UID == current {
				return event.Send(event.Event.Target.Type, event.Event.Target.ID, a.payLogMessage(ctx, event.Actions(), settings(event).ImageReplies, log)...)
			}
		}
		if len(logs) > 0 {
			return event.SendText("当前绑定的uid未获取数据，请私聊获取")
		}
	}
	job := &payLogJob{}
	if kept, ok := a.PayKeys.get(owner); ok && (current == "" || kept.uid == current) {
		job.key = kept.key
	} else if player.Owned {
		job.choice, job.provider, job.uid = player.Choice, a.accountClient(event).Provider, player.UID
	} else {
		return event.SendText("请私聊发送米游社链接，可以发送【" + a.Game.Prefix + "充值统计帮助】查看链接教程")
	}
	notice(ctx, event, "正在获取数据,可能需要30s")
	return a.startPayLog(ctx, event, job, start)
}
