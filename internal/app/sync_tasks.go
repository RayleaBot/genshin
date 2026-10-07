package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/gacha"
)

// A background sync reads a role's official wish history through the
// accounts plugin in one event moved to the background, each page a second
// after the previous one: xiaoyao's 更新抽卡记录 in chat and the management
// page's single sync read on demand as the user or administrator who asked;
// a daily sync's trigger reads with the task's delegation. Rounds are listed
// in memory only, one at a time for each role.

// BackgroundSync is a role's latest round. State is running, completed,
// failed or canceled; LastCode is why it did not complete. Notify tells
// Owner, the account's chat user, of a completed round.
type BackgroundSync struct {
	Ref        string         `json:"ref"`
	Role       Role           `json:"role"`
	Owner      Subject        `json:"owner"`
	Full       bool           `json:"full"`
	Notify     bool           `json:"notify"`
	Daily      bool           `json:"daily"`
	State      string         `json:"state"`
	LastCode   string         `json:"last_code,omitempty"`
	Progress   gacha.SyncInfo `json:"progress"`
	StartedMS  int64          `json:"started_ms"`
	FinishedMS int64          `json:"finished_ms,omitempty"`
}

// backgroundSyncs are the rounds by ref; cancel ends a running one's reads,
// and canceled is the code it was canceled with.
type backgroundSyncs struct {
	mu    sync.Mutex
	items map[string]*backgroundSync
}

type backgroundSync struct {
	BackgroundSync
	cancel   context.CancelFunc
	canceled string
}

func syncTaskID(game, provider string, choice Selection) string {
	sum := sha256.Sum256([]byte(provider + "\x00" + choice.AccountRef + "\x00" + choice.RoleRef))
	return "game.sync." + game + "." + hex.EncodeToString(sum[:])
}

// begin starts a round under its ref at now; the returned context ends when
// the round is canceled. A role whose round still runs is refused.
func (s *backgroundSyncs) begin(ctx context.Context, round BackgroundSync, now time.Time) (context.Context, BackgroundSync, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.items[round.Ref]; item != nil && item.State == "running" {
		return nil, BackgroundSync{}, gameError("sync_task_running", "此角色的抽卡记录正在后台读取，请稍后再试。")
	}
	if s.items == nil {
		s.items = map[string]*backgroundSync{}
	}
	ctx, cancel := context.WithCancel(ctx)
	round.State, round.LastCode, round.Progress, round.StartedMS, round.FinishedMS = "running", "", gacha.SyncInfo{}, now.UnixMilli(), 0
	s.items[round.Ref] = &backgroundSync{BackgroundSync: round, cancel: cancel}
	return ctx, round, nil
}

// drop forgets a round that never started reading, as one whose event the
// host would not move to the background.
func (s *backgroundSyncs) drop(ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.items[ref]; item != nil {
		item.cancel()
		delete(s.items, ref)
	}
}

func (s *backgroundSyncs) progress(ref string, info gacha.SyncInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.items[ref]; item != nil {
		item.Progress = info
	}
}

// finish records at now how a round's reads ended.
func (s *backgroundSyncs) finish(ref string, err error, now time.Time) BackgroundSync {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.items[ref]
	item.cancel()
	item.FinishedMS = now.UnixMilli()
	switch {
	case item.canceled != "":
		item.State, item.LastCode = "canceled", item.canceled
	case err != nil:
		item.State, item.LastCode = "failed", syncTaskError(err)
	default:
		item.State, item.LastCode = "completed", "sync_completed"
	}
	return item.BackgroundSync
}

// stop cancels with code the running rounds match picks; false when none
// ran.
func (s *backgroundSyncs) stop(code string, match func(BackgroundSync) bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	stopped := false
	for _, item := range s.items {
		if item.State == "running" && item.canceled == "" && match(item.BackgroundSync) {
			item.canceled = code
			item.cancel()
			stopped = true
		}
	}
	return stopped
}

// list is every role's latest round, the newest first.
func (s *backgroundSyncs) list() []BackgroundSync {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []BackgroundSync{}
	for _, item := range s.items {
		items = append(items, item.BackgroundSync)
	}
	slices.SortFunc(items, func(a, b BackgroundSync) int { return cmp.Compare(b.StartedMS, a.StartedMS) })
	return items
}

// syncTaskError is the code of a round's failure.
func syncTaskError(err error) string {
	switch {
	case errors.Is(err, gacha.ErrConflict):
		return "plugin.game_sync_conflict"
	case errors.Is(err, gacha.ErrInvalid):
		return "plugin.game_sync_invalid"
	case errors.Is(err, gacha.ErrSync):
		return "plugin.game_sync_busy"
	case errors.Is(err, context.DeadlineExceeded):
		return "plugin.event_timeout"
	}
	return PublicError(err).Code
}

// readAccountGacha reads every pool of role with client for the round ref,
// each page a second after the previous one to keep under 米游社's rate
// limits, and merges the records once all are read. delegation, when set,
// is the daily task's delegation the pages are read with.
func (a *App) readAccountGacha(ctx context.Context, client AccountsClient, choice Selection, role Role, full bool, delegation, ref string) error {
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, role.UID, role.Region, full)
	if err != nil {
		return err
	}
	_, err = a.readSync(ctx, info, time.Second, func(ctx context.Context, pool, end string, page int) (gacha.RemotePage, error) {
		params := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "operation": a.Game.ID + ".gacha", "input": map[string]any{"gacha_type": pool, "end_id": end, "page": page}}
		if delegation != "" {
			params["delegation_ref"] = delegation
		}
		var response QueryResult
		if err := client.call(ctx, "execute", params, &response); err != nil {
			return gacha.RemotePage{}, err
		}
		if response.Role.UID != role.UID || response.Role.Region != role.Region || response.Role.Ref != role.Ref {
			return gacha.RemotePage{}, gacha.ErrInvalid
		}
		return gacha.ParsePage(response.Role.UID, response.Role.Region, pool, end, response.Data)
	}, func(info gacha.SyncInfo) { a.BackgroundSyncs.progress(ref, info) })
	return err
}

// notifySync tells a completed round's account user, through the bot the
// account belongs to, what the round added.
func (a *App) notifySync(ctx context.Context, event *rayleabot.EventContext, round BackgroundSync) error {
	owner, result := round.Owner, round.Progress.Result
	if result == nil || !slices.ContainsFunc(event.Bots, func(bot rayleabot.Bot) bool {
		return bot.ID == owner.BotID && bot.SourceProtocol == owner.SourceProtocol && bot.SourceAdapter == owner.SourceAdapter
	}) {
		return gameError("bot_missing", "所属机器人暂不可用。")
	}
	text := fmt.Sprintf("抽卡后台同步完成\n%s · %s\n新增 %d 条，档案共 %d 条。", round.Role.Nickname, round.Role.UID, result.Added, result.Total)
	_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: owner.SourceProtocol, SourceAdapter: owner.SourceAdapter, TargetType: "private", TargetID: owner.ActorID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
	return err
}

// syncTaskAction answers the management page's background and daily syncs.
func (a *App) syncTaskAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "gacha.task.list":
		tasks, err := a.SyncTasks.List()
		return map[string]any{"items": a.BackgroundSyncs.list(), "tasks": tasks}, err
	case "gacha.task.cancel":
		ref := asText(input["ref"])
		if !a.BackgroundSyncs.stop("canceled", func(s BackgroundSync) bool { return s.Ref == ref }) {
			return nil, gameError("sync_task_missing", "后台同步不存在或已结束。")
		}
		return map[string]any{"canceled": true}, nil
	case "gacha.task.remove":
		return a.removeSyncTask(ctx, event, asText(input["ref"]))
	}
	if input["confirm"] != true {
		return nil, gameError("input_invalid", "请明确开启或重新运行后台同步。")
	}
	switch action {
	case "gacha.task.start":
		return a.startSync(ctx, event, input)
	case "gacha.task.create":
		return a.createSyncTask(ctx, event, input)
	case "gacha.task.run":
		return a.rerunSyncTask(ctx, event, asText(input["ref"]))
	}
	return nil, gameError("operation_denied", "操作不存在。")
}

// startSync is the page's single sync: after the role is checked the action
// moves to the background, answering the page with the round, and reads the
// role's records in the same event, telling the account's user when asked.
func (a *App) startSync(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Selection
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if decodeObject(input, &q) != nil {
		return nil, gameError("input_invalid", "请选择角色并确认开始后台同步。")
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配后台同步。")
	}
	ref := syncTaskID(a.Game.ID, client.Provider, q.Selection)
	work, round, err := a.BackgroundSyncs.begin(ctx, BackgroundSync{Ref: ref, Role: role, Owner: account.Owner, Full: q.Full, Notify: q.Notify}, a.now())
	if err != nil {
		return nil, err
	}
	if err := detach(ctx, event, map[string]any{"task": round}); err != nil {
		a.BackgroundSyncs.drop(ref)
		return nil, err
	}
	err = a.readAccountGacha(work, client, q.Selection, role, q.Full, "", ref)
	round = a.BackgroundSyncs.finish(ref, err, a.now())
	if round.State == "completed" && round.Notify {
		_ = a.notifySync(ctx, event, round)
	}
	return map[string]any{"task": round}, nil
}

// createSyncTask saves a daily sync: the delegation it reads with, for days,
// and its job, which triggers each minute to find the day's round due.
func (a *App) createSyncTask(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Selection
		Hour   int  `json:"hour"`
		Days   int  `json:"days"`
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if decodeObject(input, &q) != nil || q.Days < 1 || q.Days > 90 || q.Hour < 0 || q.Hour > 23 {
		return nil, gameError("input_invalid", "请选择 1–90 天有效期和 0–23 时。")
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配后台同步。")
	}
	task := SyncTask{Ref: syncTaskID(a.Game.ID, client.Provider, q.Selection), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Hour: q.Hour, Full: q.Full, Notify: q.Notify, State: "creating"}
	// The task is kept as "creating" while the delegation and job are made
	// outside the store's lock, then enabled or dropped.
	err = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
		if len(*items) >= 256 {
			return gameError("sync_task_limit", "每日同步任务已达上限。")
		}
		if i >= 0 {
			return gameError("sync_task_exists", "此角色已有每日同步，请重新运行已有任务，或移除后调整设置。")
		}
		*items = append(*items, task)
		return nil
	})
	if err != nil {
		return nil, err
	}
	var grant struct {
		Delegation struct {
			Ref         string `json:"ref"`
			ExpiresAtMS int64  `json:"expires_at_ms"`
		} `json:"delegation"`
	}
	err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": a.Game.ID + ".gacha", "days": q.Days}, &grant)
	if err == nil {
		task.DelegationRef, task.ExpiresAtMS = grant.Delegation.Ref, grant.Delegation.ExpiresAtMS
		_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "* * * * *", LogLabel: a.Game.Name + "抽卡每日同步"})
	}
	if err == nil {
		task.State, task.LastCode = "waiting", "queued"
		err = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "同步任务创建已取消。")
			}
			(*items)[i] = task
			return nil
		})
	}
	if err != nil {
		_ = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
			if i >= 0 {
				*items = slices.Delete(*items, i, i+1)
			}
			return nil
		})
		if task.DelegationRef != "" {
			_ = client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": task.DelegationRef}, nil)
		}
		return nil, err
	}
	return map[string]any{"task": task}, nil
}

// rerunSyncTask makes a daily sync due at its next trigger, which reads the
// round and counts it as the day's.
func (a *App) rerunSyncTask(ctx context.Context, event *rayleabot.EventContext, ref string) (map[string]any, error) {
	items, err := a.SyncTasks.List()
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(items, func(t SyncTask) bool { return t.Ref == ref })
	if i < 0 {
		return nil, gameError("sync_task_missing", "同步任务不存在。")
	}
	// The account check runs before the task is edited, outside the store's
	// lock.
	client := AccountsClient{Caller: event.Actions(), Provider: items[i].Provider, Game: a.Game.ID}
	_, role, err := client.Authorize(ctx, items[i].Selection)
	if err != nil {
		return nil, err
	}
	err = a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
		if i < 0 {
			return gameError("sync_task_missing", "同步任务不存在。")
		}
		t := &(*items)[i]
		switch {
		case t.State == "running" || t.State == "creating":
			return gameError("sync_task_running", "此任务正在处理，请查看进度。")
		case t.ExpiresAtMS <= time.Now().UnixMilli() || t.DelegationRef == "":
			return gameError("sync_task_expired", "委托已过期，请移除后重新创建任务。")
		case role.UID != t.Role.UID || role.Region != t.Role.Region:
			return gameError("role_missing", "账号角色已变更，请重新创建任务。")
		}
		t.State, t.Failures, t.NextCheckMS, t.LastCode = "running", 0, 0, "queued"
		t.RunDay, _, _ = syncTaskTime(time.Now().UnixMilli(), t.Hour)
		return nil
	})
	return map[string]any{"queued": err == nil}, err
}

// removeSyncTask stops a daily sync: its round, its job and its delegation.
func (a *App) removeSyncTask(ctx context.Context, event *rayleabot.EventContext, ref string) (map[string]any, error) {
	var t SyncTask
	err := a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
		if i < 0 {
			return gameError("sync_task_missing", "同步任务不存在。")
		}
		t = (*items)[i]
		*items = slices.Delete(*items, i, i+1)
		return nil
	})
	if err != nil {
		return map[string]any{"removed": false, "delegation_revoked": false}, err
	}
	a.BackgroundSyncs.stop("canceled", func(s BackgroundSync) bool { return s.Ref == ref && s.Daily })
	_, _ = event.Actions().SchedulerDelete(ctx, t.Ref)
	client := AccountsClient{Caller: event.Actions(), Provider: t.Provider, Game: a.Game.ID}
	revoked := client.call(ctx, "delegation.revoke", map[string]any{"account_ref": t.AccountRef, "delegation_ref": t.DelegationRef}, nil) == nil
	return map[string]any{"removed": true, "delegation_revoked": revoked}, nil
}

// runSyncTask is a trigger of a daily sync. Once the day's round is due, the
// trigger moves to the background and reads it with the task's delegation;
// a round the host would not move there, or whose role another round is
// reading, is left to the next trigger.
func (a *App) runSyncTask(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ref := event.Event.TaskID()
	task, ok, err := a.SyncTasks.claim(ref)
	if errors.Is(err, errTaskMissing) {
		// The task was removed; its job goes with it.
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
	}
	if err != nil || !ok {
		return event.Result(map[string]any{"checked": false})
	}
	round, notify, err := a.readDailyRound(ctx, event, &task)
	// The task is released before its user is told and the event ends: the
	// host delivers the next trigger as soon as it does.
	a.SyncTasks.release(ref)
	if notify && a.notifySync(ctx, event, round) != nil {
		// The notification was marked before it was sent and is not sent
		// again.
		_ = a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
			if i >= 0 && (*items)[i].LastNotificationMS == task.LastNotificationMS {
				(*items)[i].LastCode = "sync_completed.notification_failed"
			}
			return nil
		})
	}
	if err != nil && !errors.Is(err, errTaskChanged) {
		failure := PublicError(err)
		return event.Fail(failure.Code, failure.Message)
	}
	return event.Result(map[string]any{"checked": round.Ref != ""})
}

// readDailyRound reads a claimed daily task's round once it is due and
// writes how it ended; round is empty when no round was read, and notify
// tells whether its user is to be told.
func (a *App) readDailyRound(ctx context.Context, event *rayleabot.EventContext, task *SyncTask) (round BackgroundSync, notify bool, _ error) {
	start := a.now()
	if due, err := a.SyncTasks.due(task, start.UnixMilli()); err != nil || !due {
		return BackgroundSync{}, false, err
	}
	work, _, err := a.BackgroundSyncs.begin(ctx, BackgroundSync{Ref: task.Ref, Role: task.Role, Owner: task.Owner, Full: task.Full, Notify: task.Notify, Daily: true}, start)
	if err != nil {
		return BackgroundSync{}, false, nil
	}
	if err := detach(ctx, event, nil); err != nil {
		a.BackgroundSyncs.drop(task.Ref)
		return BackgroundSync{}, false, nil
	}
	client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
	err = a.readAccountGacha(work, client, task.Selection, task.Role, task.Full, task.DelegationRef, task.Ref)
	round = a.BackgroundSyncs.finish(task.Ref, err, a.now())
	notify, err = a.SyncTasks.finish(task, round, err, a.now().UnixMilli())
	return round, notify, err
}

// syncTaskCommand is xiaoyao's 更新抽卡记录, which hands Yunzai the account's
// authkey: the event moves to the background, the role's records are read
// and the chat gets Yunzai's report. 设置全量更新抽卡记录 makes the round read
// in full, then is turned off.
func (a *App) syncTaskCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 1 {
		return event.SendText("请指定一个 UID，或省略以使用默认角色。")
	}
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	accounts, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, role, err := Choose(accounts, a.Game.ID, uid)
	if err == nil && !syncRegionAllowed(role.Region) {
		err = gameError("region_unsupported", "此区服暂未适配后台同步。")
	}
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	owner := chatOwner(event)
	full := a.fullLinks.active(owner)
	ref := syncTaskID(a.Game.ID, client.Provider, choice)
	work, _, err := a.BackgroundSyncs.begin(ctx, BackgroundSync{Ref: ref, Role: role, Owner: owner, Full: full}, a.now())
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if err := detach(ctx, event, nil); err != nil {
		a.BackgroundSyncs.drop(ref)
		return event.SendText(friendlyError(err))
	}
	notice(ctx, event, "抽卡记录获取中请稍等...")
	before := gachaPoolCounts(a.archiveOrEmpty(role.UID, role.Region))
	err = a.readAccountGacha(work, client, choice, role, full, "", ref)
	if finished := a.BackgroundSyncs.finish(ref, err, a.now()); finished.State == "canceled" {
		return event.SendText("抽卡记录读取已取消。")
	}
	if err != nil {
		return event.SendText(friendlyError(syncError(err)))
	}
	archive, err := a.Gacha.Read(role.UID, role.Region)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return answerChat(ctx, event, a.gachaReport(owner, archive, before, full))
}
