package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/localdata"
)

// Address migration runs before the first event, and again when the host's
// bot snapshot changes so an unresolved legacy adapter can be recovered.
// Store locks serialize migration with tasks already running in the background.
type addressMigration struct {
	mu   sync.Mutex
	bots string
	done bool
}

func (a *App) migrateAddresses(ctx context.Context, event *rayleabot.EventContext) error {
	m := &a.addressMigration
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, _ := json.Marshal(event.Bots)
	if m.done && m.bots == string(raw) {
		return nil
	}
	pending := 0
	steps := []func() (int, error){
		func() (int, error) { return a.Groups.migrate(event.Bots) },
		func() (int, error) { return a.Reminders.migrate(event.Bots) },
		func() (int, error) { return a.SyncTasks.migrate(event.Bots) },
		func() (int, error) { return a.Subscriptions.migrate(event.Bots) },
		func() (int, error) { return a.Interactions.migrate(event.Bots) },
		func() (int, error) { return a.PayLogs.migrate(event.Bots) },
	}
	for _, step := range steps {
		n, err := step()
		if err != nil {
			_, _ = event.Actions().LoggerWrite(ctx, rayleabot.LoggerWriteRequest{Level: "error", Message: "机器人账号迁移未完成，旧记录已保留；请检查插件数据目录，修复后重试。", Fields: map[string]any{"error": err.Error()}})
			return gameError("address_migration_failed", "机器人账号迁移未完成，请检查插件日志。")
		}
		pending += n
	}
	if pending > 0 {
		_, _ = event.Actions().LoggerWrite(ctx, rayleabot.LoggerWriteRequest{Level: "warn", Message: "旧记录缺少机器人账号，已保留，暂不投递；请恢复原连接身份或重新选择机器人账号。", Fields: map[string]any{"records": pending}})
	}
	legacySimulation, err := a.Simulation.legacyFiles()
	if err != nil {
		return err
	}
	if legacySimulation > 0 {
		_, _ = event.Actions().LoggerWrite(ctx, rayleabot.LoggerWriteRequest{Level: "warn", Message: "旧模拟抽卡档案将在账号访问时迁移；仅有哈希且原连接已丢失的档案无法自动认领，原文件保留。", Fields: map[string]any{"files": legacySimulation}})
	}
	m.bots, m.done = string(raw), true
	return nil
}

func (s *PayLogStore) migrate(bots []rayleabot.Bot) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, err := migrateAddressFile(s.Path, bots)
	if err != nil {
		return pending, err
	}
	var items []payLogOwner
	if err = localdata.Read(s.Path, &items); err != nil {
		return pending, err
	}
	merged := []payLogOwner{}
	for _, item := range items {
		i := slices.IndexFunc(merged, func(v payLogOwner) bool { return item.Owner.BotID != "" && v.Owner.same(item.Owner) })
		if i < 0 {
			merged = append(merged, item)
			continue
		}
		merged[i].Logs = mergeAddressEntries(merged[i].Logs, item.Logs, func(v PayLog) string { return v.UID }, func(v PayLog) int64 { return int64(v.Crystal) })
	}
	if len(merged) != len(items) {
		err = writeAddressMerge(s.Path, merged)
	}
	return pending, err
}

// recoverAddresses works on JSON before decoding Subject/GroupScope. Unresolved
// records keep their original adapter as a recovery hint; it is never a route.
func recoverAddresses(value any, bots []rayleabot.Bot) (changed bool, pending int) {
	switch v := value.(type) {
	case []any:
		for _, child := range v {
			c, n := recoverAddresses(child, bots)
			changed, pending = changed || c, pending+n
		}
	case map[string]any:
		protocol, _ := v["source_protocol"].(string)
		_, actor := v["actor_id"]
		_, group := v["group_id"]
		if protocol != "" && (actor || group) {
			id, _ := v["bot_id"].(string)
			adapter, _ := v["source_adapter"].(string)
			if id == "" && adapter != "" {
				for _, bot := range bots {
					if bot.SourceProtocol == protocol && bot.SourceAdapter == adapter && bot.ID != "" {
						id = bot.ID
						v["bot_id"], changed = id, true
						break
					}
				}
			}
			if id == "" {
				pending++
			} else if _, exists := v["source_adapter"]; exists {
				delete(v, "source_adapter")
				changed = true
			}
		}
		for _, child := range v {
			c, n := recoverAddresses(child, bots)
			changed, pending = changed || c, pending+n
		}
	}
	return changed, pending
}

func readAddressJSON(path string) (any, error) {
	var raw json.RawMessage
	if err := localdata.Read(path, &raw); err != nil || len(raw) == 0 {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	err := decoder.Decode(&value)
	return value, err
}

func migrateAddressFile(path string, bots []rayleabot.Bot) (int, error) {
	value, err := readAddressJSON(path)
	if err != nil || value == nil {
		return 0, err
	}
	changed, pending := recoverAddresses(value, bots)
	if changed {
		err = localdata.Write(path, value)
	}
	return pending, err
}

// Keep the original list when several old subjects collapse into one account;
// conflicting scalar preferences remain available in this archive.
func writeAddressMerge(path string, value any) error {
	archive := path + ".migrated"
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		var raw json.RawMessage
		if err := localdata.Read(path, &raw); err != nil {
			return err
		}
		if err := localdata.Write(archive, raw); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return localdata.Write(path, value)
}

func addressFiles(directory string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return slices.DeleteFunc(files, func(f os.DirEntry) bool {
		return !f.Type().IsRegular() || strings.HasPrefix(f.Name(), ".") || filepath.Ext(f.Name()) != ".json"
	}), err
}

func (s *taskFiles[T]) migrate(bots []rayleabot.Bot) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := addressFiles(s.Directory)
	if err != nil {
		return 0, err
	}
	pending := 0
	for _, file := range files {
		n, err := migrateAddressFile(filepath.Join(s.Directory, file.Name()), bots)
		if err != nil {
			return pending, err
		}
		pending += n
	}
	return pending, nil
}

func (s *ReminderStore) migrate(bots []rayleabot.Bot) (int, error) {
	pending, err := s.taskFiles.migrate(bots)
	if err != nil {
		return pending, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return pending, err
	}
	byOwner := map[Subject][]Reminder{}
	for _, task := range items {
		if task.Kind == notePushKind && task.Owner.BotID != "" {
			byOwner[task.Owner.account()] = append(byOwner[task.Owner.account()], task)
		}
	}
	for _, tasks := range byOwner {
		if len(tasks) < 2 {
			continue
		}
		// Keep the longest-lived usable delegation and its scheduler handle.
		slices.SortStableFunc(tasks, func(a, b Reminder) int {
			if a.Enabled != b.Enabled {
				if a.Enabled {
					return -1
				}
				return 1
			}
			if a.ExpiresAtMS > b.ExpiresAtMS {
				return -1
			}
			if a.ExpiresAtMS < b.ExpiresAtMS {
				return 1
			}
			return strings.Compare(a.Ref, b.Ref)
		})
		kept := tasks[0]
		for _, task := range tasks[1:] {
			for _, scope := range task.Groups {
				if !slices.ContainsFunc(kept.Groups, scope.same) {
					kept.Groups = append(kept.Groups, scope)
				}
			}
			kept.LastAttemptMS = max(kept.LastAttemptMS, task.LastAttemptMS)
			kept.LastCheckedMS = max(kept.LastCheckedMS, task.LastCheckedMS)
			kept.NextCheckMS = max(kept.NextCheckMS, task.NextCheckMS)
		}
		if err = localdata.Write(s.file(kept.Ref), kept); err != nil {
			return pending, err
		}
		for _, task := range tasks[1:] {
			path := s.file(task.Ref)
			if err = os.Rename(path, path+".migrated"); err != nil {
				return pending, err
			}
		}
	}
	return pending, nil
}

func (s *GroupStore) migrate(bots []rayleabot.Bot) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := addressFiles(s.Directory)
	if err != nil {
		return 0, err
	}
	type source struct {
		path     string
		data     GroupData
		modified int64
	}
	groups := map[GroupScope][]source{}
	pending := 0
	for _, file := range files {
		path := filepath.Join(s.Directory, file.Name())
		value, err := readAddressJSON(path)
		if err != nil {
			return pending, err
		}
		_, n := recoverAddresses(value, bots)
		pending += n
		if n > 0 {
			continue
		}
		var data GroupData
		if err = decodeObject(value, &data); err != nil {
			return pending, err
		}
		if !data.Scope.valid() {
			return pending, fmt.Errorf("invalid group scope in %s", file.Name())
		}
		info, err := file.Info()
		if err != nil {
			return pending, err
		}
		groups[data.Scope] = append(groups[data.Scope], source{path, data, info.ModTime().UnixNano()})
	}
	for scope, sources := range groups {
		target := s.file(scope)
		// A canonical file wins after a partially completed migration; otherwise
		// the most recently saved settings win. Conflicting originals are archived.
		slices.SortStableFunc(sources, func(a, b source) int {
			if a.path == target {
				return 1
			}
			if b.path == target {
				return -1
			}
			if a.modified < b.modified {
				return -1
			}
			if a.modified > b.modified {
				return 1
			}
			return strings.Compare(a.path, b.path)
		})
		data := sources[0].data
		for _, old := range sources[1:] {
			data = mergeGroupData(data, old.data)
		}
		if len(sources) == 1 && sources[0].path == target {
			if _, err = migrateAddressFile(target, bots); err != nil {
				return pending, err
			}
			continue
		}
		if err = localdata.Write(target, data); err != nil {
			return pending, err
		}
		delete(s.configs, scope)
		for _, source := range sources {
			if source.path != target {
				if err = os.Rename(source.path, source.path+".migrated"); err != nil {
					return pending, err
				}
			}
		}
	}
	return pending, nil
}

// recover reuses the management page's account selector for an unresolved
// legacy group. Its original file remains until the complete group is saved.
func (s *GroupStore) recover(ref string, scope GroupScope, update func(*GroupData) error) error {
	key, err := hex.DecodeString(ref)
	if err != nil || len(key) != 32 {
		return gameError("input_invalid", "旧群记录无效。")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.Directory, ref+".json")
	var old GroupData
	if err = localdata.Read(path, &old); err != nil {
		return err
	}
	if old.Scope.BotID != "" || old.Scope.Protocol != scope.Protocol || old.Scope.GroupID != scope.GroupID {
		return gameError("source_invalid", "旧群记录与所选账号不匹配。")
	}
	if err = update(&old); err != nil {
		return err
	}
	current, err := s.read(scope)
	if err != nil {
		return err
	}
	old.Scope = scope.account()
	merged := mergeGroupData(current, old)
	if err = localdata.Write(s.file(scope), merged); err != nil {
		return err
	}
	s.remember(scope.account(), merged.Config)
	return os.Rename(path, path+".migrated")
}

// mergeGroupData preserves each member's latest records, all distinct aliases
// and seasons. Scalar settings follow the newer source chosen above.
func mergeGroupData(old, current GroupData) GroupData {
	current.Revision = max(old.Revision, current.Revision) + 1
	if current.Config.Enabled == nil {
		current.Config.Enabled = old.Config.Enabled
	}
	if current.Config.ImageReplies == nil {
		current.Config.ImageReplies = old.Config.ImageReplies
	}
	if current.Config.Aliases == nil {
		current.Config.Aliases = map[string]string{}
	}
	for key, value := range old.Config.Aliases {
		if _, ok := current.Config.Aliases[key]; !ok {
			current.Config.Aliases[key] = value
		}
	}
	current.Rank = mergeAddressEntries(old.Rank, current.Rank, func(v RankEntry) string { return v.UID + "\x00" + v.CharacterID }, func(v RankEntry) int64 { return v.UpdatedAtMS })
	current.Challenges = mergeAddressEntries(old.Challenges, current.Challenges, func(v ChallengeEntry) string {
		return strings.Join([]string{v.Kind, v.Season, v.Region, v.ActorID}, "\x00")
	}, func(v ChallengeEntry) int64 { return v.UpdatedMS })
	current.Cards = mergeAddressEntries(old.Cards, current.Cards, func(v RoleCardOffer) string { return v.ActorID }, func(v RoleCardOffer) int64 { return v.SavedMS })
	if current.Stygian == nil {
		current.Stygian = map[string][]StygianEntry{}
	}
	for season, entries := range old.Stygian {
		current.Stygian[season] = mergeAddressEntries(entries, current.Stygian[season], func(v StygianEntry) string { return v.UID }, func(v StygianEntry) int64 { return 0 })
	}
	return current
}

func mergeAddressEntries[T any](old, current []T, key func(T) string, updated func(T) int64) []T {
	for _, entry := range old {
		i := slices.IndexFunc(current, func(v T) bool { return key(v) == key(entry) })
		if i < 0 {
			current = append(current, entry)
		} else if updated(entry) > updated(current[i]) {
			current[i] = entry
		}
	}
	return current
}

func (s *ContentSubscriptions) migrate(bots []rayleabot.Bot) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, err := migrateAddressFile(s.Path, bots)
	if err != nil {
		return pending, err
	}
	items, err := s.read()
	if err != nil {
		return pending, err
	}
	merged := []ContentSubscription{}
	for _, item := range items {
		i := slices.IndexFunc(merged, func(v ContentSubscription) bool {
			return item.Owner.BotID != "" && v.Owner.SourceProtocol == item.Owner.SourceProtocol && v.Owner.BotID == item.Owner.BotID && v.GroupID == item.GroupID
		})
		if i < 0 {
			merged = append(merged, item)
			continue
		}
		kept := &merged[i]
		for _, kind := range item.Kinds {
			if !slices.Contains(kept.Kinds, kind) {
				kept.Kinds = append(kept.Kinds, kind)
			}
		}
		if kept.Sent == nil {
			kept.Sent = map[string]int64{}
		}
		for post, at := range item.Sent {
			kept.Sent[post] = max(kept.Sent[post], at)
		}
		kept.WarnedOn = max(kept.WarnedOn, item.WarnedOn)
	}
	if len(merged) != len(items) {
		err = writeAddressMerge(s.Path, merged)
	}
	return pending, err
}

func (s *InteractionStore) migrate(bots []rayleabot.Bot) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, err := migrateAddressFile(s.Path, bots)
	if err != nil {
		return pending, err
	}
	items, err := s.read()
	if err != nil {
		return pending, err
	}
	merged := []InteractionProfile{}
	changed := false
	for _, item := range items {
		if item.Owner.BotID == "" {
			merged = append(merged, item)
			continue
		}
		ref := interactionRef(item.Owner)
		changed = changed || item.Ref != ref
		item.Ref = ref
		i := slices.IndexFunc(merged, func(v InteractionProfile) bool { return v.Owner.same(item.Owner) })
		if i < 0 {
			merged = append(merged, item)
			continue
		}
		changed = true
		old := merged[i]
		if old.LastImageMS > item.LastImageMS {
			old, item = item, old
		}
		if item.Lists == nil {
			item.Lists = map[string][]string{}
		}
		for kind, ids := range old.Lists {
			for _, id := range ids {
				if !slices.Contains(item.Lists[kind], id) {
					item.Lists[kind] = append(item.Lists[kind], id)
				}
			}
		}
		item.LastPokeMS = max(old.LastPokeMS, item.LastPokeMS)
		if item.EnemyLevel == 0 {
			item.EnemyLevel = old.EnemyLevel
		}
		merged[i] = item
	}
	if changed {
		err = writeAddressMerge(s.Path, merged)
	}
	return pending, err
}
