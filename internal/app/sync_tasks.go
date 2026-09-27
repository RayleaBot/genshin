package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

// A background sync reads a role's official wish history through the
// accounts plugin in one event moved to the background: xiaoyao's
// 更新抽卡记录 in chat, or the management page's background sync. The event
// keeps its origin, so every page is read as the user or administrator who
// asked, a second after the previous one. Syncs are held in memory only, one
// at a time for each role.

// BackgroundSync is a role's latest background sync. State is running,
// completed, failed or canceled; LastCode is why it did not complete.
type BackgroundSync struct {
	Ref        string         `json:"ref"`
	Role       Role           `json:"role"`
	Full       bool           `json:"full"`
	State      string         `json:"state"`
	LastCode   string         `json:"last_code,omitempty"`
	Progress   gacha.SyncInfo `json:"progress"`
	StartedMS  int64          `json:"started_ms"`
	FinishedMS int64          `json:"finished_ms,omitempty"`
}

// backgroundSyncs are the background syncs by ref; cancel ends a running
// one's reads, and canceled is the code it was canceled with.
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

// begin starts the sync of role under ref at now; the returned context ends
// when the sync is canceled. A role whose sync still runs is refused.
func (s *backgroundSyncs) begin(ctx context.Context, ref string, role Role, full bool, now time.Time) (context.Context, BackgroundSync, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.items[ref]; item != nil && item.State == "running" {
		return nil, BackgroundSync{}, gameError("sync_task_running", "此角色的抽卡记录正在后台读取，请稍后再试。")
	}
	if s.items == nil {
		s.items = map[string]*backgroundSync{}
	}
	ctx, cancel := context.WithCancel(ctx)
	item := &backgroundSync{BackgroundSync: BackgroundSync{Ref: ref, Role: role, Full: full, State: "running", StartedMS: now.UnixMilli()}, cancel: cancel}
	s.items[ref] = item
	return ctx, item.BackgroundSync, nil
}

// drop forgets a sync that never started reading, as one whose event the
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

// finish records at now how a sync's reads ended.
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

// stop cancels with code the running syncs match picks; false when none
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

// list is every role's latest sync, the newest first.
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

// syncTaskError is the code of a sync's failure.
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

// readAccountGacha reads every pool of role with client for the sync ref,
// each page a second after the previous one to keep under 米游社's rate
// limits, and merges the records once all are read.
func (a *App) readAccountGacha(ctx context.Context, client AccountsClient, choice Selection, role Role, full bool, ref string) error {
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, role.UID, role.Region, full)
	if err != nil {
		return err
	}
	_, err = a.readSync(ctx, info, time.Second, func(ctx context.Context, pool, end string, page int) (gacha.RemotePage, error) {
		response, err := client.Execute(ctx, choice, a.Game.ID+".gacha", map[string]any{"gacha_type": pool, "end_id": end, "page": page})
		if err != nil {
			return gacha.RemotePage{}, err
		}
		if response.Role.UID != role.UID || response.Role.Region != role.Region || response.Role.Ref != role.Ref {
			return gacha.RemotePage{}, gacha.ErrInvalid
		}
		return gacha.ParsePage(response.Role.UID, response.Role.Region, pool, end, response.Data)
	}, func(info gacha.SyncInfo) { a.BackgroundSyncs.progress(ref, info) })
	return err
}

// syncTaskAction answers the management page's background syncs: start
// moves the action to the background, answering the page with the sync, and
// reads the role's records in the same event.
func (a *App) syncTaskAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "gacha.task.list":
		return map[string]any{"items": a.BackgroundSyncs.list()}, nil
	case "gacha.task.cancel":
		ref := asText(input["ref"])
		if !a.BackgroundSyncs.stop("canceled", func(s BackgroundSync) bool { return s.Ref == ref }) {
			return nil, gameError("sync_task_missing", "后台同步不存在或已结束。")
		}
		return map[string]any{"canceled": true}, nil
	case "gacha.task.start":
	default:
		return nil, gameError("operation_denied", "操作不存在。")
	}
	var q struct {
		Selection
		Full bool `json:"full"`
	}
	if decodeObject(input, &q) != nil || input["confirm"] != true {
		return nil, gameError("input_invalid", "请选择角色并确认开始后台同步。")
	}
	client := a.accountClient(event)
	_, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配后台同步。")
	}
	ref := syncTaskID(a.Game.ID, client.Provider, q.Selection)
	work, started, err := a.BackgroundSyncs.begin(ctx, ref, role, q.Full, a.now())
	if err != nil {
		return nil, err
	}
	if err := detach(ctx, event, map[string]any{"task": started}); err != nil {
		a.BackgroundSyncs.drop(ref)
		return nil, err
	}
	err = a.readAccountGacha(work, client, q.Selection, role, q.Full, ref)
	return map[string]any{"task": a.BackgroundSyncs.finish(ref, err, a.now())}, nil
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
	work, _, err := a.BackgroundSyncs.begin(ctx, ref, role, full, a.now())
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if err := detach(ctx, event, nil); err != nil {
		a.BackgroundSyncs.drop(ref)
		return event.SendText(friendlyError(err))
	}
	notice(ctx, event, "抽卡记录获取中请稍等...")
	before := gachaPoolCounts(a.archiveOrEmpty(role.UID, role.Region))
	err = a.readAccountGacha(work, client, choice, role, full, ref)
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
