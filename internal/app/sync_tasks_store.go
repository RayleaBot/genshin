package app

import (
	"path/filepath"
	"time"

	"github.com/RayleaBot/plugin-genshin/internal/gacha"
)

// SyncTask is a daily background sync: each day after Hour, Beijing time,
// its trigger moves to the background and reads the role's records with the
// task's delegation, which lasts until ExpiresAtMS. State is creating,
// waiting, running (due at the next trigger), paused or expired; LastCode
// is the last outcome, and Result what the last completed round added.
type SyncTask struct {
	Ref string `json:"ref"`
	Selection
	Owner              Subject             `json:"owner"`
	Role               Role                `json:"role"`
	Provider           string              `json:"provider"`
	DelegationRef      string              `json:"delegation_ref"`
	ExpiresAtMS        int64               `json:"expires_at_ms"`
	Hour               int                 `json:"hour"`
	Full               bool                `json:"full"`
	Notify             bool                `json:"notify"`
	State              string              `json:"state"`
	NextCheckMS        int64               `json:"next_check_ms"`
	LastCheckedMS      int64               `json:"last_checked_ms"`
	LastFinishedMS     int64               `json:"last_finished_ms"`
	LastNotificationMS int64               `json:"last_notification_ms"`
	RunDay             string              `json:"run_day"`
	Failures           int                 `json:"failures"`
	LastCode           string              `json:"last_code"`
	Result             *gacha.ImportResult `json:"result,omitempty"`
}

// SyncTaskStore keeps each daily sync in its own file.
type SyncTaskStore struct {
	taskFiles[SyncTask]
}

func syncTaskStore(directory string) *SyncTaskStore {
	return &SyncTaskStore{taskFiles[SyncTask]{Directory: filepath.Join(directory, "sync-tasks")}}
}

// syncTaskTime is the Beijing day of now, and the task hour of that day and
// of the next.
func syncTaskTime(now int64, hour int) (string, int64, int64) {
	china := time.UnixMilli(now).In(time.FixedZone("UTC+8", 8*3600))
	at := time.Date(china.Year(), china.Month(), china.Day(), hour, 0, 0, 0, china.Location())
	return china.Format("2006-01-02"), at.UnixMilli(), at.AddDate(0, 0, 1).UnixMilli()
}

// due decides whether a claimed task reads at this trigger, at now, and
// saves the schedule it moves otherwise: an expired task stops, and a
// waiting one waits for its hour and reads once a day.
func (s *SyncTaskStore) due(task *SyncTask, now int64) (bool, error) {
	if task.State != "running" && task.State != "waiting" || task.NextCheckMS > now {
		return false, nil
	}
	if task.ExpiresAtMS <= now {
		task.State, task.LastCode = "expired", "expired"
		return false, s.save(*task)
	}
	day, at, nextDay := syncTaskTime(now, task.Hour)
	if task.State == "waiting" {
		switch {
		case now < at:
			task.NextCheckMS = at
			return false, s.save(*task)
		case task.RunDay == day:
			task.NextCheckMS = nextDay
			return false, s.save(*task)
		}
		task.State, task.RunDay, task.Failures = "running", day, 0
	}
	task.LastCheckedMS = now
	return true, nil
}

// finish writes at now how a task's round ended: run is the round as the
// page listed it, err its failure. A completed round waits for the day after
// the one it began on and marks its notification before it is sent, so a
// restart does not send it again; notify is whether to send it.
func (s *SyncTaskStore) finish(task *SyncTask, run BackgroundSync, err error, now int64) (notify bool, _ error) {
	_, _, nextDay := syncTaskTime(run.StartedMS, task.Hour)
	switch {
	case run.State == "canceled":
		task.State, task.LastCode, task.NextCheckMS = "waiting", run.LastCode, nextDay
	case err != nil:
		syncTaskFailure(task, err, now)
	default:
		task.State, task.LastCode, task.NextCheckMS, task.LastFinishedMS, task.Failures = "waiting", "sync_completed", nextDay, now, 0
		task.Result = run.Progress.Result
		if task.Notify {
			task.LastNotificationMS = now
		}
	}
	if err := s.save(*task); err != nil {
		return false, err
	}
	return run.State == "completed" && task.Notify, nil
}

// syncTaskFailure backs a failed round off for five minutes; revoked or
// unverified accounts, record conflicts and invalid pages pause the task,
// and the third failure of a round leaves it to the next day.
func syncTaskFailure(task *SyncTask, err error, now int64) {
	task.LastCode = syncTaskError(err)
	task.Failures++
	task.NextCheckMS = now + 5*60*1000
	switch task.LastCode {
	case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.account_subject_denied", "plugin.upstream_auth_invalid", "plugin.upstream_device_required", "plugin.upstream_challenge_required", "plugin.upstream_gacha_unavailable", "plugin.game_sync_conflict", "plugin.game_sync_invalid":
		task.State = "paused"
	default:
		if task.Failures >= 3 {
			_, _, next := syncTaskTime(now, task.Hour)
			task.State, task.NextCheckMS = "waiting", next
		}
	}
}

// pauseArchive pauses the daily syncs of an archive being removed.
func (s *SyncTaskStore) pauseArchive(uid, region string) error {
	return s.edit("", func(items *[]SyncTask, _ int) error {
		for i := range *items {
			if t := &(*items)[i]; t.Role.UID == uid && t.Role.Region == region {
				t.State, t.LastCode = "paused", "archive_removed"
			}
		}
		return nil
	})
}
