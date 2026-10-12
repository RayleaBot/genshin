package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/localdata"
)

type historicalSource struct {
	kind string
	path string
	mu   *sync.Mutex
	list bool
}

type historicalRecord struct {
	Kind           string `json:"kind"`
	Ref            string `json:"ref"`
	Revision       string `json:"revision"`
	SourceProtocol string `json:"source_protocol"`
	BotID          string `json:"bot_id"`
	Label          string `json:"label"`
	Assignable     bool   `json:"assignable"`
}

func (a *App) historicalSources() []historicalSource {
	return []historicalSource{
		{"group", a.Groups.Directory, &a.Groups.mu, false},
		{"reminder", a.Reminders.Directory, &a.Reminders.mu, false},
		{"sync_task", a.SyncTasks.Directory, &a.SyncTasks.mu, false},
		{"subscription", a.Subscriptions.Path, &a.Subscriptions.mu, true},
		{"interaction", a.Interactions.Path, &a.Interactions.mu, true},
		{"pay_log", a.PayLogs.Path, &a.PayLogs.mu, true},
		{"simulation", a.Simulation.Directory, &a.Simulation.mu, false},
		{"panel_history", a.PanelHistory.Directory, &a.PanelHistory.mu, false},
		{"monthly", a.Monthly.Directory, &a.Monthly.mu, false},
		{"billing", a.Billing.Directory, &a.Billing.mu, false},
		{"cloud_archive", a.CloudArchive.Directory, &a.CloudArchive.mu, false},
	}
}

func historicalRevision(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func historicalIdentity(kind string, value any) (protocol, bot, label string) {
	if kind == "panel_history" {
		for _, item := range asList(value) {
			owner := asObject(asObject(item)["owner"])
			if id := asText(owner["bot_id"]); id != "" {
				return asText(owner["source_protocol"]), id, asText(asObject(asObject(item)["panel"])["name"])
			}
		}
		return "", "", "旧面板历史（身份未保存）"
	}
	data := asObject(value)
	identity := asObject(data["owner"])
	if kind == "group" {
		identity = asObject(data["scope"])
	}
	if kind == "simulation" {
		var scope []string
		if json.Unmarshal([]byte(asText(data["scope"])), &scope) == nil && len(scope) == 5 {
			return scope[0], scope[1], scope[4] + " · " + scope[3]
		}
		return "", "", "旧模拟档案（身份未保存）"
	}
	label = asText(identity["actor_id"])
	if group := asText(identity["group_id"]); group != "" {
		label = "群 " + group
	}
	if group := asText(data["group_id"]); group != "" {
		label += " · 群 " + group
	}
	if uid := asText(asObject(data["role"])["uid"]); uid != "" {
		label += " · " + uid
	}
	if uid := asText(data["uid"]); uid != "" {
		label += " · " + uid
	}
	return asText(identity["source_protocol"]), asText(identity["bot_id"]), label
}

// The browser classifies complete accounts with the directory's single-item
// endpoint. The plugin supplies its own record counts, including offline data.
func (source historicalSource) records() ([]historicalRecord, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	out := []historicalRecord{}
	appendRecord := func(ref string, value any) {
		management := sha256.Sum256([]byte("management"))
		if source.kind == "simulation" && (ref == "config" || ref == hex.EncodeToString(management[:]) || asText(asObject(value)["scope"]) == "management") {
			return
		}
		protocol, bot, label := historicalIdentity(source.kind, value)
		assignable := protocol == "onebot11" && bot == "" && slices.Contains([]string{"group", "reminder", "sync_task", "subscription", "interaction", "pay_log"}, source.kind)
		out = append(out, historicalRecord{Kind: source.kind, Ref: ref, Revision: historicalRevision(value), SourceProtocol: protocol, BotID: bot, Label: label, Assignable: assignable})
	}
	if source.list {
		value, err := readAddressJSON(source.path)
		if err != nil {
			return nil, err
		}
		items, _ := value.([]any)
		for _, item := range items {
			ref := asText(asObject(item)["ref"])
			if source.kind == "pay_log" {
				ref = historicalRevision(asObject(item)["owner"])
			}
			appendRecord(ref, item)
		}
		return out, nil
	}
	files, err := addressFiles(source.path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		value, err := readAddressJSON(filepath.Join(source.path, file.Name()))
		if err != nil {
			return nil, err
		}
		appendRecord(strings.TrimSuffix(file.Name(), ".json"), value)
	}
	return out, nil
}

func (a *App) manageHistoricalRecords(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	if event.Event.ManagementPageID() != "help" {
		return nil, managementInvalid("请在全局历史数据与归属中处理。")
	}
	if _, ok := event.Event.ManagementBot(); ok {
		return nil, managementInvalid("历史数据入口不接受账号页请求。")
	}
	sources := a.historicalSources()
	if action == "history.records.list" {
		var q struct {
			Offset int `json:"offset"`
		}
		if decodeObject(input, &q) != nil || q.Offset < 0 {
			return nil, managementInvalid("分页无效。")
		}
		items := []historicalRecord{}
		for _, source := range sources {
			rows, err := source.records()
			if err != nil {
				return nil, err
			}
			items = append(items, rows...)
		}
		start, end := min(q.Offset, len(items)), min(q.Offset+50, len(items))
		var next *int
		if end < len(items) {
			next = &end
		}
		return map[string]any{"items": items[start:end], "total": len(items), "next_offset": next}, nil
	}
	index := slices.IndexFunc(sources, func(s historicalSource) bool { return s.kind == asText(input["kind"]) })
	if index < 0 {
		return nil, managementInvalid("历史记录类型无效。")
	}
	if action == "history.records.get" {
		return sources[index].view(input)
	}
	if (action != "history.records.remove" && action != "history.records.assign") || input["confirm"] != true {
		return nil, managementInvalid("请确认历史记录操作。")
	}
	var bot rayleabot.BotAccount
	if action == "history.records.assign" {
		if decodeObject(input["bot"], &bot) != nil || bot.SourceProtocol != "onebot11" || bot.BotID == "" {
			return nil, managementInvalid("仅可归入 OneBot11 机器人账号。")
		}
	}
	source := sources[index]
	ref := asText(input["ref"])
	task, err := a.editHistoricalRecord(source, ref, asText(input["revision"]), bot)
	if err != nil {
		return nil, err
	}
	if task != "" {
		a.BackgroundSyncs.stop("canceled", func(s BackgroundSync) bool { return s.Ref == task })
		_, err = event.Actions().SchedulerDelete(ctx, task)
	}
	return map[string]any{"updated": true, "scheduler_removed": err == nil}, nil
}

func (source historicalSource) view(input map[string]any) (map[string]any, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	var q struct {
		Offset int `json:"offset"`
	}
	if decodeObject(input, &q) != nil || q.Offset < 0 {
		return nil, managementInvalid("记录分页无效。")
	}
	_, _, _, value, err := source.read(asText(input["ref"]))
	if err != nil {
		return nil, err
	}
	if value == nil || historicalRevision(value) != asText(input["revision"]) {
		return nil, gameError("history_changed", "记录已变化，请刷新后重试。")
	}
	if data := asObject(value); data != nil {
		delete(data, "delegation_ref")
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	content := []rune(string(raw))
	start, end := min(q.Offset, len(content)), min(q.Offset+12000, len(content))
	var next *int
	if end < len(content) {
		next = &end
	}
	return map[string]any{"text": string(content[start:end]), "offset": start, "total": len(content), "next_offset": next}, nil
}

// Called under the source store's lock. Only record IDs select files, never
// paths supplied by the management page.
func (source historicalSource) read(ref string) (path string, items []any, index int, value any, err error) {
	path, index = source.path, -1
	if source.list {
		var raw any
		raw, err = readAddressJSON(path)
		if err != nil {
			return
		}
		items, _ = raw.([]any)
		index = slices.IndexFunc(items, func(item any) bool {
			if source.kind == "pay_log" {
				return historicalRevision(asObject(item)["owner"]) == ref
			}
			return asText(asObject(item)["ref"]) == ref
		})
		if index >= 0 {
			value = items[index]
		}
	} else {
		var key []byte
		key, err = hex.DecodeString(ref)
		if err != nil || (len(key) != 16 && len(key) != 32) {
			err = managementInvalid("历史记录引用无效。")
			return
		}
		path = filepath.Join(path, ref+".json")
		value, err = readAddressJSON(path)
	}
	return
}

func (a *App) editHistoricalRecord(source historicalSource, ref, revision string, bot rayleabot.BotAccount) (string, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	path, items, index, value, err := source.read(ref)
	if err != nil {
		return "", err
	}
	if value == nil || historicalRevision(value) != revision {
		return "", gameError("history_changed", "记录已变化，请刷新后重试。")
	}
	protocol, oldBot, _ := historicalIdentity(source.kind, value)
	data := asObject(value)
	if source.kind == "simulation" && asText(data["scope"]) == "management" {
		return "", managementInvalid("此记录属于全局模拟档案。")
	}
	task := ""
	if source.kind == "reminder" || source.kind == "sync_task" || source.kind == "subscription" {
		task = asText(data["ref"])
	}
	if bot.BotID == "" {
		if source.list {
			items = slices.Delete(items, index, index+1)
			if err := localdata.Write(path, items); err != nil {
				return "", err
			}
		} else if err := os.Remove(path); err != nil {
			return "", err
		}
		if source.kind == "group" {
			a.Groups.configs = nil
		}
		return task, nil
	}
	if protocol != "onebot11" || oldBot != "" || !slices.Contains([]string{"group", "reminder", "sync_task", "subscription", "interaction", "pay_log"}, source.kind) {
		return "", managementInvalid("已有完整账号或非 OneBot11 记录不能改挂。")
	}
	if err := assignHistoricalIdentity(value, bot); err != nil {
		return "", err
	}
	if source.kind == "group" {
		var old GroupData
		if err := decodeObject(value, &old); err != nil {
			return "", err
		}
		current, err := a.Groups.read(old.Scope)
		if err != nil {
			return "", err
		}
		merged := mergeGroupData(current, old)
		if err = localdata.Write(a.Groups.file(old.Scope), merged); err != nil {
			return "", err
		}
		a.Groups.remember(old.Scope, merged.Config)
		return "", os.Rename(path, path+".migrated")
	}
	if source.kind == "reminder" {
		data["enabled"], data["last_code"] = false, "account_assigned"
	}
	if source.kind == "sync_task" {
		data["state"], data["last_code"] = "paused", "account_assigned"
	}
	if source.list {
		// Do not replace an existing subject's preferences or push subscription.
		for i, other := range items {
			if i == index {
				continue
			}
			p, id, _ := historicalIdentity(source.kind, other)
			otherData := asObject(other)
			if p == bot.SourceProtocol && id == bot.BotID && (source.kind == "subscription" && asText(otherData["group_id"]) == asText(data["group_id"]) || source.kind != "subscription" && asText(asObject(otherData["owner"])["actor_id"]) == asText(asObject(data["owner"])["actor_id"])) {
				return "", gameError("history_changed", "此账号已有对应记录，请先核对并清理旧记录。")
			}
		}
		items[index] = value
		return "", localdata.Write(path, items)
	}
	return task, localdata.Write(path, value)
}

func assignHistoricalIdentity(value any, bot rayleabot.BotAccount) error {
	switch data := value.(type) {
	case []any:
		for _, item := range data {
			if err := assignHistoricalIdentity(item, bot); err != nil {
				return err
			}
		}
	case map[string]any:
		_, actor := data["actor_id"]
		_, group := data["group_id"]
		if protocol := asText(data["source_protocol"]); protocol != "" && (actor || group) {
			if protocol != bot.SourceProtocol || asText(data["bot_id"]) != "" && asText(data["bot_id"]) != bot.BotID {
				return managementInvalid("记录中的账号归属冲突，不能改挂。")
			}
			data["bot_id"] = bot.BotID
			delete(data, "source_adapter")
		}
		for _, item := range data {
			if err := assignHistoricalIdentity(item, bot); err != nil {
				return err
			}
		}
	}
	return nil
}
