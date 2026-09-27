package app

import (
	"context"
	"errors"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Work that may take longer than the host's event deadline, as reading every
// page of a gacha history, moves its event to the background first: the host
// completes the delivery, frees the chat's queue, and the event keeps its
// origin until the background deadline, so the flow runs as sequential code
// and answers in its chat.

// errBackgroundBusy answers a flow the host would not move to the
// background while the plugin already holds its limit of background events.
var errBackgroundBusy = gameError("background_busy", "正在后台处理的任务较多，请稍后再试。")

// errStepUnfinished is a scheduled task's step that its event's deadline cut
// off, or that the host would not move to the background; the next trigger
// does the step again.
var errStepUnfinished = gameError("task_unfinished", "本次触发未能在时限内完成，下次触发时重试。")

// detach moves event to the background, unless it is there already; result is
// what a management page receives. A refusal leaves the event in the
// foreground.
func detach(ctx context.Context, event *rayleabot.EventContext, result any) error {
	if event.Detached() {
		return nil
	}
	if _, err := event.Detach(ctx, result); err != nil {
		var refused *rayleabot.ActionError
		if errors.As(err, &refused) && refused.Code == "platform.rate_limited" {
			return errBackgroundBusy
		}
		return err
	}
	return nil
}

// answerChat answers a chat event with a reply: every message but the last is
// sent while the event goes on, and the last ends it.
func answerChat(ctx context.Context, event *rayleabot.EventContext, reply [][]rayleabot.Segment) error {
	if len(reply) == 0 {
		return event.Result(map[string]any{"handled": true})
	}
	for _, message := range reply[:len(reply)-1] {
		_, _ = post(ctx, event, message...)
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, reply[len(reply)-1]...)
}

// textReply is a reply of one text message.
func textReply(text string) [][]rayleabot.Segment {
	return [][]rayleabot.Segment{{rayleabot.Text(text)}}
}

// clock is the time flows count and wait by, as the pause between two pages;
// tests set their own.
type clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

func (a *App) now() time.Time {
	if a.clock != nil {
		return a.clock.Now()
	}
	return time.Now()
}

// sleep waits for d or until ctx ends.
func (a *App) sleep(ctx context.Context, d time.Duration) error {
	if a.clock != nil {
		return a.clock.Sleep(ctx, d)
	}
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
