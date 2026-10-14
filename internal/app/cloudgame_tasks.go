package app

import (
	"errors"
	"slices"
	"time"
)

func stopAccountRound(task *Reminder, now int64, code string) {
	task.LastCode = code
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
	if task.Once {
		task.Enabled = false
	}
	if slices.Contains([]string{"plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_subject_denied", "plugin.upstream_auth_invalid", "plugin.account_cloud_unavailable", "plugin.upstream_cloud_unavailable"}, code) {
		task.Enabled = false
	}
	if code == "plugin.upstream_device_required" || code == "plugin.upstream_challenge_required" {
		task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
	}
}
func (s *ReminderStore) tickCloudGame(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	today := time.UnixMilli(now).In(time.FixedZone("UTC+8", 28800)).Format("2006-01-02")
	if task.LastSignDay == today {
		task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
		if task.Once {
			task.Enabled = false
		}
		return s.save(*task)
	}
	previous := task.LastSignDay
	task.LastSignDay = today
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
	task.LastCheckedMS = now
	task.LastCode = "cloud_attempted"
	if err := s.save(*task); err != nil {
		return err
	}
	result, err := query(*task)
	if errors.Is(err, errStepUnfinished) {
		// The event ended before the query answered; the next trigger
		// queries again.
		task.LastSignDay, task.NextCheckMS, task.LastCode = previous, now, PublicError(err).Code
		return s.save(*task)
	}
	if err != nil {
		stopAccountRound(task, now, PublicError(err).Code)
		return s.save(*task)
	}
	stopAccountRound(task, now, "cloud_completed")
	if task.Notify {
		task.LastAttemptMS = now
	}
	if err = s.save(*task); err != nil {
		return err
	}
	if task.Notify {
		message := game.Name + "云游戏查询已完成\n账号 " + task.Role.Nickname + "\n免费时长 " + asText(result.Data["free_time"]) + " 分钟 · 本次发放 " + asText(result.Data["send_freetime"]) + " 分钟"
		if err = send(*task, message); err != nil {
			task.LastCode += ".notification_failed"
			return s.save(*task)
		}
	}
	return nil
}
