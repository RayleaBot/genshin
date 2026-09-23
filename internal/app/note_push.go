package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// xiaoyao's 群体力推送 (apps/Note.js noteTask and DailyNoteTask,
// model/note.js). In a group, 开启体力推送 enters the sender in the group's
// push and 关闭体力推送 takes them out; 体力设置群推送开启/关闭 and
// 体力设置群阈值N (20–160, 120 by default) set the group. Every ten minutes a
// sender whose resin reached a group's threshold is pushed to that group
// with an @ and the 体力 page, at most once in twelve hours. Each sender has
// one task reading the role in use with a delegation of 90 days, the longest
// the accounts plugin grants; 开启体力推送 renews it.

const (
	notePushKind     = "note_push"
	notePushCooldown = 12 * time.Hour
)

// NotePush is a group's 体力推送 settings: Off turns the group's pushes off,
// and Resin is the resin that pushes, 120 when unset.
type NotePush struct {
	Off   bool `json:"off,omitempty"`
	Resin int  `json:"resin,omitempty"`
}

func (p NotePush) threshold() float64 {
	if p.Resin == 0 {
		return 120
	}
	return float64(p.Resin)
}

// set applies 体力设置群推送开启/关闭, or 体力设置群阈值N with N kept to 20–160
// and 120 without one, as upstream.
func (p *NotePush) set(args []string) {
	if len(args) > 0 && (args[0] == "开启" || args[0] == "关闭") {
		p.Off = args[0] == "关闭"
		return
	}
	resin := 120
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n != 0 {
			resin = n
		}
	}
	p.Resin = min(160, max(20, resin))
}

// pushGroup is the first of the groups, in the order entered, whose push is
// on and whose threshold the resin reached.
func (s *GroupStore) pushGroup(groups []GroupScope, resin float64) (GroupScope, bool) {
	for _, scope := range groups {
		group, err := s.Read(scope)
		if err == nil && !group.NotePush.Off && (group.Config.Enabled == nil || *group.Config.Enabled) && resin >= group.NotePush.threshold() {
			return scope, true
		}
	}
	return GroupScope{}, false
}

// XiaoyaoSettings are the xiaoyao-cvs-plugin settings (defSet/config) that
// apply here, under its names and with its defaults.
type XiaoyaoSettings struct {
	// NoteTask is isNoteTask: whether 体力推送 pushes at all.
	NoteTask bool `json:"is_note_task"`
	// NoteSetAuth is noteSetAuth, who may change a group's 体力推送: 1 group
	// administrators, 2 super administrators.
	NoteSetAuth int `json:"note_set_auth"`
}

func notePushRef(game string, owner Subject) string {
	sum := sha256.Sum256([]byte(owner.SourceProtocol + "\x00" + owner.SourceAdapter + "\x00" + owner.BotID + "\x00" + owner.ActorID))
	return "game.reminder.push." + game + "." + hex.EncodeToString(sum[:])
}

// notePushCommand answers xiaoyao's 体力推送 words, which it takes only in
// groups.
func (a *App) notePushCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if event.Event.Target.Type != "group" {
		return event.Result(map[string]any{"handled": true})
	}
	scope := groupScope(event)
	if command == "note-push-group" {
		switch {
		case settings(event).Xiaoyao.NoteSetAuth == 1 && !groupAdministrator(event):
			return event.SendText("只有管理员才能操作。")
		case settings(event).Xiaoyao.NoteSetAuth != 1 && !slices.Contains(event.SuperAdmins, event.Event.Actor.ID):
			return event.SendText("只有主人才能操作。")
		}
		err := a.Groups.Update(scope, func(g *GroupData) error {
			g.NotePush.set(args)
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(strings.Replace(event.Event.Message.PlainText, "#", "", 1) + "操作成功~")
	}
	group, err := a.Groups.Read(scope)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	owner := chatOwner(event)
	ref := notePushRef(a.Game.ID, owner)
	if len(args) > 0 && args[0] == "关闭" {
		if group.NotePush.Off {
			return event.SendText("群体力推送关闭了~\n请联系管理员开启功能~")
		}
		// Upstream splices at indexOf, so a sender not entered takes the
		// last one out; here only the sender leaves.
		var task Reminder
		left := false
		err = a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
			if i < 0 || !slices.Contains((*items)[i].Groups, scope) {
				return nil
			}
			left = true
			(*items)[i].Groups = slices.DeleteFunc((*items)[i].Groups, func(g GroupScope) bool { return g == scope })
			if task = (*items)[i]; len(task.Groups) == 0 {
				*items = slices.Delete(*items, i, i+1)
			}
			return nil
		})
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if !left {
			return event.Result(map[string]any{"handled": true})
		}
		if len(task.Groups) == 0 {
			_, _ = event.Actions().SchedulerDelete(ctx, ref)
			client := a.accountClient(event)
			client.Provider = task.Provider
			_ = client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": task.DelegationRef}, nil)
		}
		return event.SendText("体力推送关闭成功~\n后续将不会为您推送体力")
	}
	player, _ := a.panelOwner(ctx, event, "")
	if !player.Owned {
		return event.SendText("请先" + a.Game.Prefix + "绑定ck\n发送【体力帮助】获取教程")
	}
	if group.NotePush.Off {
		return event.SendText("群体力推送关闭了~\n请联系管理员开启功能~")
	}
	client := a.accountClient(event)
	var grant struct {
		Delegation struct {
			Ref         string `json:"ref"`
			ExpiresAtMS int64  `json:"expires_at_ms"`
		} `json:"delegation"`
	}
	if err = client.call(ctx, "delegation.create", map[string]any{"account_ref": player.Choice.AccountRef, "role_ref": player.Choice.RoleRef, "task_id": ref, "operation": a.Game.ID + ".note", "days": 90}, &grant); err != nil {
		return event.SendText(friendlyError(err))
	}
	err = a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
		task := Reminder{Ref: ref, Kind: notePushKind, Owner: owner}
		if i >= 0 {
			task = (*items)[i]
		} else if len(*items) >= 256 {
			return gameError("reminder_limit", "提醒数量已达上限。")
		}
		task.Selection, task.Role, task.Provider = player.Choice, player.Role, client.Provider
		task.DelegationRef, task.ExpiresAtMS, task.Enabled = grant.Delegation.Ref, grant.Delegation.ExpiresAtMS, true
		if !slices.Contains(task.Groups, scope) {
			task.Groups = append(task.Groups, scope)
		}
		if i >= 0 {
			(*items)[i] = task
		} else {
			*items = append(*items, task)
		}
		return nil
	})
	if err == nil {
		_, err = event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: ref, Cron: "*/10 * * * *", LogLabel: a.Game.Name + "体力推送", Payload: map[string]any{"kind": notePushKind}})
	}
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("体力推送开启成功~\n后续每天会为您推送体力")
}

// runNotePush is xiaoyao's DailyNoteTask for one sender: once twelve hours
// have passed since the last push, the resin is read and pushed to the first
// of the sender's groups whose threshold it reached.
func (a *App) runNotePush(ctx context.Context, event *rayleabot.EventContext) error {
	ref := asText(event.Event.Payload["task_id"])
	task, ok, err := a.Reminders.claim(ref)
	if err != nil || !ok {
		return event.Result(map[string]any{"checked": false})
	}
	defer a.Reminders.release(ref)
	now := time.Now()
	if !settings(event).Xiaoyao.NoteTask || !task.Enabled || now.Sub(time.UnixMilli(task.LastAttemptMS)) < notePushCooldown {
		return event.Result(map[string]any{"checked": false})
	}
	if task.ExpiresAtMS <= now.UnixMilli() {
		task.Enabled = false
		task.LastCode = "expired"
		return a.saveNotePush(event, task)
	}
	if !slices.ContainsFunc(event.Bots, func(bot rayleabot.Bot) bool {
		return bot.ID == task.Owner.BotID && bot.SourceProtocol == task.Owner.SourceProtocol && bot.SourceAdapter == task.Owner.SourceAdapter
	}) {
		return event.Result(map[string]any{"checked": false})
	}
	client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
	var result QueryResult
	task.LastCheckedMS = now.UnixMilli()
	if err = client.call(ctx, "execute", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "operation": a.Game.ID + ".note", "input": map[string]any{}, "delegation_ref": task.DelegationRef}, &result); err != nil {
		task.LastCode = PublicError(err).Code
		switch task.LastCode {
		case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid":
			task.Enabled = false
		}
		return a.saveNotePush(event, task)
	}
	current, _, ok := stamina(result.Data)
	if !ok {
		task.LastCode = "plugin.game_note_invalid"
		return a.saveNotePush(event, task)
	}
	task.LastCode = "checked"
	scope, ok := a.Groups.pushGroup(task.Groups, current)
	if !ok {
		return a.saveNotePush(event, task)
	}
	// The push is marked before it is sent, so a restart does not send it
	// again.
	task.LastAttemptMS = now.UnixMilli()
	task.LastCode = "notified"
	if err = a.Reminders.save(task); err != nil {
		return event.Result(map[string]any{"checked": false})
	}
	view := View{Title: a.Game.Name + "体力", Rows: []Row{}}
	if operation, ok := a.operation(a.Game.ID + ".note"); ok {
		view = BusinessView(a.Game, operation, result, a.Catalog)
	}
	view.Image = a.featureImage(ctx, client, task.Selection, a.Game.ID+".note", "体力", map[string]any{}, result)
	page := rayleabot.Text("\n" + view.Text())
	if image := a.renderView(ctx, event, view); image != "" {
		page = rayleabot.Image(image)
	}
	_, _ = event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: scope.Protocol, SourceAdapter: scope.Adapter, TargetType: "group", TargetID: scope.GroupID,
		Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.At(task.Owner.ActorID), rayleabot.Text("哥哥（姐姐）你的体力快满了哦~"), page}}})
	return event.Result(map[string]any{"checked": true})
}

func (a *App) saveNotePush(event *rayleabot.EventContext, task Reminder) error {
	_ = a.Reminders.save(task)
	return event.Result(map[string]any{"checked": true})
}
