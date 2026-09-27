package app

import (
	"context"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Jobs created before their payloads carried their task IDs trigger without
// one, and the host does not say which job fired. The first such trigger
// registers the job of every saved task again under the task's ID, which
// replaces the old job with one whose payload carries the ID; the tasks then
// run as they should. A job without a saved task, as the temporary job of a
// gacha or customer service link or the job of a removed task, cannot be
// found this way: its triggers do nothing.

// legacyJobs records whether the saved tasks' jobs were registered again in
// this process.
type legacyJobs struct {
	mu   sync.Mutex
	done bool
}

// runLegacyJob answers a trigger of a job from before task IDs were carried:
// the first registers the saved tasks' jobs again, and later ones retry
// until every job was registered.
func (a *App) runLegacyJob(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	// A trigger that finds another registering leaves it to that one.
	if a.legacy.mu.TryLock() {
		defer a.legacy.mu.Unlock()
		if !a.legacy.done {
			ctx, cancel := a.eventWork(ctx, a.now())
			defer cancel()
			a.legacy.done = a.reschedule(ctx, event.Actions()) == nil
		}
	}
	return event.Result(map[string]any{"handled": false})
}

// reschedule registers the job of every saved reminder, account task,
// background sync and group push again. A task whose creation stopped
// before its job was created has no delegation and is left out.
func (a *App) reschedule(ctx context.Context, host taskHost) error {
	jobs := []rayleabot.SchedulerCreateRequest{}
	reminders, err := a.Reminders.List()
	if err != nil {
		return err
	}
	for _, task := range reminders {
		if task.DelegationRef != "" {
			jobs = append(jobs, a.reminderJob(task))
		}
	}
	syncs, err := a.SyncTasks.List()
	if err != nil {
		return err
	}
	for _, task := range syncs {
		if task.DelegationRef != "" {
			jobs = append(jobs, a.syncJob(task.Ref))
		}
	}
	subscriptions, err := a.Subscriptions.List()
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		jobs = append(jobs, a.contentJob(subscription.Ref))
	}
	var failed error
	for _, job := range jobs {
		if _, err := host.SchedulerCreate(ctx, job); err != nil {
			failed = err
		}
	}
	return failed
}
