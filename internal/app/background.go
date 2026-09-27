package app

import (
	"context"
	"errors"

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
