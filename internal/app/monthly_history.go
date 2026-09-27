package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/localdata"
)

type MonthlySnapshot struct {
	Month         string          `json:"month"`
	YearEstimated bool            `json:"year_estimated"`
	SavedMS       int64           `json:"saved_ms"`
	Data          json.RawMessage `json:"data"`
}
type MonthlyArchive struct {
	Revision uint64            `json:"revision"`
	Items    []MonthlySnapshot `json:"items"`
}
type MonthlyStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *MonthlyStore) file(provider string, choice Selection) string {
	raw, _ := json.Marshal([]string{provider, choice.AccountRef, choice.RoleRef})
	sum := sha256.Sum256(raw)
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func (s *MonthlyStore) Read(provider string, choice Selection) (MonthlyArchive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := MonthlyArchive{Items: []MonthlySnapshot{}}
	err := localdata.Read(s.file(provider, choice), &out)
	return out, err
}
func (s *MonthlyStore) Update(provider string, choice Selection, revision uint64, fn func(*MonthlyArchive) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := MonthlyArchive{Items: []MonthlySnapshot{}}
	file := s.file(provider, choice)
	if err := localdata.Read(file, &out); err != nil {
		return err
	}
	if out.Revision != revision {
		return gameError("history_changed", "月报档案已变化，请刷新后重试。")
	}
	if err := fn(&out); err != nil {
		return err
	}
	out.Revision++
	return localdata.Write(file, out)
}
func monthlyKey(data map[string]any, now time.Time) (string, bool, error) {
	raw := asText(data["data_month"])
	// The ledger names only the month; its year is this year's or the last.
	month, err := strconv.Atoi(raw)
	if err == nil && month >= 1 && month <= 12 {
		china := now.In(time.FixedZone("UTC+8", 28800))
		year := china.Year()
		if month > int(china.Month()) {
			year--
		}
		return fmt.Sprintf("%04d-%02d", year, month), true, nil
	}
	return "", false, gameError("monthly_invalid", "官方月报未提供可辨识的月份，未保存。")
}
func decodeMonthly(raw json.RawMessage) (map[string]any, error) {
	var data map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	err := decoder.Decode(&data)
	return data, err
}
func monthlyAmounts(data map[string]any) map[string]float64 {
	out := map[string]float64{}
	keys := map[string]string{"current_primogems": "原石", "current_mora": "摩拉"}
	month := asObject(data["month_data"])
	for key, label := range keys {
		if v, ok := challengeNumber(month[key]); ok {
			out[label] = v
		}
	}
	return out
}
func monthlySummaries(archive MonthlyArchive) map[string]any {
	items := []map[string]any{}
	totals := map[string]float64{}
	coverage := map[string]int{}
	slices.SortFunc(archive.Items, func(a, b MonthlySnapshot) int { return strings.Compare(b.Month, a.Month) })
	for _, v := range archive.Items {
		data, err := decodeMonthly(v.Data)
		if err != nil {
			continue
		}
		amounts := monthlyAmounts(data)
		for k, n := range amounts {
			totals[k] += n
			coverage[k]++
		}
		items = append(items, map[string]any{"month": v.Month, "year_estimated": v.YearEstimated, "saved_ms": v.SavedMS, "amounts": amounts})
	}
	return map[string]any{"items": items, "totals": totals, "coverage": coverage, "revision": archive.Revision}
}
func (a *App) monthlyAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	var r struct {
		Roles []Role `json:"roles"`
	}
	if err := client.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": a.Game.ID}, &r); err != nil {
		return nil, err
	}
	i := slices.IndexFunc(r.Roles, func(role Role) bool { return role.Ref == choice.RoleRef && role.Game == a.Game.ID })
	if i < 0 {
		return nil, gameError("role_missing", "所选角色授权不可用。")
	}
	role := r.Roles[i]
	archive, err := a.Monthly.Read(client.Provider, choice)
	if err != nil {
		return nil, err
	}
	if action == "monthly.list" {
		return monthlySummaries(archive), nil
	}
	if action == "monthly.fetch" {
		args := map[string]any{}
		if month, exists := input["month"]; exists && asText(month) != "" {
			args["month"] = month
		}
		result, err := client.Execute(ctx, choice, a.Game.ID+".monthly", args)
		if err != nil {
			return nil, err
		}
		key, err := a.Monthly.Save(client.Provider, choice, archive.Revision, result.Data, time.Now())
		if err != nil {
			return nil, err
		}
		available := []string{}
		for _, value := range asList(result.Data["optional_month"]) {
			month, err := strconv.Atoi(asText(value))
			if err == nil && (month >= 1 && month <= 12) {
				available = append(available, strconv.Itoa(month))
			}
		}
		return map[string]any{"month": key, "available_months": available, "saved": true}, nil
	}
	month := asText(input["month"])
	i = slices.IndexFunc(archive.Items, func(v MonthlySnapshot) bool { return v.Month == month })
	if i < 0 {
		return nil, gameError("history_missing", "没有保存此月份。")
	}
	if action == "monthly.remove" {
		var q struct {
			Revision uint64 `json:"revision"`
		}
		if decodeObject(input, &q) != nil {
			return nil, gameError("input_invalid", "月报档案版本无效。")
		}
		err = a.Monthly.Update(client.Provider, choice, q.Revision, func(v *MonthlyArchive) error {
			v.Items = slices.DeleteFunc(v.Items, func(item MonthlySnapshot) bool { return item.Month == month })
			return nil
		})
		return map[string]any{"removed": err == nil}, err
	}
	if action != "monthly.get" {
		return nil, gameError("operation_denied", "月报操作不存在。")
	}
	data, err := decodeMonthly(archive.Items[i].Data)
	if err != nil {
		return nil, err
	}
	result := QueryResult{Operation: a.Game.ID + ".monthly", Role: role, Data: data, FetchedAtMS: archive.Items[i].SavedMS}
	operation, _ := a.operation(result.Operation)
	view := BusinessView(a.Game, operation, result, a.Catalog)
	view.Note = "已保存月报 · " + month + " · 仅代表保存时的数据"
	return map[string]any{"view": view}, nil
}

func (a *App) monthlyCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	uid := ""
	if len(args) > 1 {
		return event.SendText("请提供至多一个 UID。")
	}
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, role, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	input := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef}
	// Upstream saves the months the official report still offers before it
	// counts.
	if err := a.refreshMonthly(ctx, client, choice); err != nil {
		return event.SendText(friendlyError(err))
	}
	result, err := a.monthlyAction(ctx, client, "monthly.list", input)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := View{Title: a.Game.Name + "已保存月报累计", Subtitle: role.UID, Rows: []Row{}, Note: "统计前会保存官方仍提供的月份；只累计本地保存的月份，更早的缺失月份和未提供项目不补算。"}
	totals := result["totals"].(map[string]float64)
	coverage := result["coverage"].(map[string]int)
	keys := []string{}
	for key := range totals {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		view.Rows = append(view.Rows, Row{Label: key, Value: fmt.Sprintf("%.0f · 覆盖 %d 个月", totals[key], coverage[key])})
	}
	if a.monthlyStats != nil {
		archive, err := a.Monthly.Read(client.Provider, choice)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		months := []SavedMonth{}
		for _, item := range archive.Items {
			if data, err := decodeMonthly(item.Data); err == nil {
				months = append(months, SavedMonth{Month: item.Month, Data: data})
			}
		}
		slices.SortFunc(months, func(a, b SavedMonth) int { return strings.Compare(a.Month, b.Month) })
		if drawn, ok := a.monthlyStats(a.imageContext(ctx), MonthlyStats{Role: role, Word: event.Event.Command(), Months: months}); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// gluedMonth reports whether the first argument is the month written with
// the command word, as 星琼7 or 星琼100000001: Yunzai's ledger rule reads any
// digits there as the month, and a number no month has as this month, never
// as a UID.
func gluedMonth(word string, args []string) bool {
	return len(args) > 0 && strings.HasSuffix(strings.TrimRight(word, "月"), args[0])
}

// ledgerMonths are the Chinese month words Yunzai's getMonth reads.
var ledgerMonths = []string{"一", "二", "三", "四", "五", "六", "七", "八", "九", "十", "十一", "十二"}

// ledgerMonth reads the month of 原石7月 or 札记七月 the way Yunzai's getMonth
// does: digits or a Chinese numeral, anything else this month; only this
// month and the two before it can be read. A month after this one is last
// year's, which the official report reads by its number. Upstream sends such
// a month (November or December in January, December in February) to this
// month instead, because its check for later months tests month <= 9 +
// month, which always holds.
func ledgerMonth(word string, now time.Time) (int, error) {
	current := int(now.Month())
	month, err := strconv.Atoi(word)
	if err != nil {
		month = slices.Index(ledgerMonths, word) + 1
	}
	if month < 1 || month > 12 {
		month = current
	}
	if (current-month+12)%12 > 2 {
		return 0, gameError("ledger_month", "札记仅支持查询最近三个月的数据")
	}
	return month, nil
}

// monthlyTask is Yunzai's ledgerTask for a super administrator's 原石任务:
// every account's report is saved as refreshMonthly saves one, as upstream
// saves each bound CK's half a second apart. Upstream reckons 0.7 seconds an
// account, so the event moves to the background before the first.
func (a *App) monthlyTask(ctx context.Context, event *rayleabot.EventContext) error {
	client := a.accountClient(event)
	choices := []Selection{}
	for page := 0; ; {
		var listed Accounts
		if err := client.call(ctx, "list", map[string]any{"all": true, "page": page}, &listed); err != nil {
			return event.SendText(friendlyError(err))
		}
		for _, item := range listed.Items {
			for _, role := range item.Roles {
				if role.Game == a.Game.ID {
					choices = append(choices, Selection{AccountRef: item.Ref, RoleRef: role.Ref})
				}
			}
		}
		if listed.NextPage == nil {
			break
		}
		page = *listed.NextPage
	}
	if err := detach(ctx, event, nil); err != nil {
		return event.SendText(friendlyError(err))
	}
	cost := time.Duration(len(choices)) * 700 * time.Millisecond
	took := ""
	for _, part := range []struct {
		value int
		unit  string
	}{{int(cost.Hours()) % 24, "小时"}, {int(cost.Minutes()) % 60, "分钟"}, {int(cost.Seconds()) % 60, "秒"}} {
		if part.value > 0 {
			took += strconv.Itoa(part.value) + part.unit
		}
	}
	start := time.Now()
	notice(ctx, event, "开始任务：保存原石数据，完成前请勿重复执行")
	notice(ctx, event, "札记ck："+strconv.Itoa(len(choices))+"个\n预计需要："+took+"\n完成时间："+start.Add(cost).In(time.FixedZone("UTC+8", 28800)).Format("01-02 15:04:05"))
	for _, choice := range choices {
		// An account that cannot be read is left as saved, as upstream.
		_ = a.refreshMonthly(ctx, client, choice)
		_ = a.sleep(ctx, 500*time.Millisecond)
	}
	return event.SendText("原石任务完成")
}

// refreshMonthly reads this month's report and saves it as keepMonthly does.
func (a *App) refreshMonthly(ctx context.Context, client AccountsClient, choice Selection) error {
	current, err := client.Execute(ctx, choice, a.Game.ID+".monthly", map[string]any{})
	if err != nil {
		return err
	}
	return a.keepMonthly(ctx, client, choice, current)
}

// keepMonthly saves a month's report that was read and each other month the
// official report still offers that was not saved after it ended, as
// Yunzai's saveLedger does; a month that fails to load is left as saved.
func (a *App) keepMonthly(ctx context.Context, client AccountsClient, choice Selection, read QueryResult) error {
	operation := a.Game.ID + ".monthly"
	now := time.Now()
	if err := a.Monthly.Keep(client.Provider, choice, read.Data, now); err != nil {
		return err
	}
	archive, err := a.Monthly.Read(client.Provider, choice)
	if err != nil {
		return err
	}
	for _, value := range asList(read.Data["optional_month"]) {
		month := asText(value)
		if month == asText(read.Data["data_month"]) {
			continue
		}
		key, _, err := monthlyKey(map[string]any{"data_month": month}, now)
		if err != nil || monthlyFinal(archive, key) {
			continue
		}
		result, err := client.Execute(ctx, choice, operation, map[string]any{"month": month})
		if err != nil {
			continue
		}
		_ = a.Monthly.Keep(client.Provider, choice, result.Data, now)
	}
	return nil
}

// monthlyFinal reports whether the archive holds the month as saved after it
// ended, which no later read changes.
func monthlyFinal(archive MonthlyArchive, key string) bool {
	start, err := time.ParseInLocation("2006-01", key, time.FixedZone("UTC+8", 28800))
	if err != nil {
		return false
	}
	end := start.AddDate(0, 1, 0).UnixMilli()
	return slices.ContainsFunc(archive.Items, func(item MonthlySnapshot) bool { return item.Month == key && item.SavedMS >= end })
}

// Keep saves a month over whatever archive is current, for reads the user did
// not start from the archive page.
func (s *MonthlyStore) Keep(provider string, choice Selection, data map[string]any, now time.Time) error {
	archive, err := s.Read(provider, choice)
	if err != nil {
		return err
	}
	_, err = s.Save(provider, choice, archive.Revision, data, now)
	return err
}

func (s *MonthlyStore) Save(provider string, choice Selection, revision uint64, data map[string]any, now time.Time) (string, error) {
	if asObject(data["month_data"]) == nil {
		return "", gameError("monthly_invalid", "官方月报没有月度内容，未保存。")
	}
	key, estimated, err := monthlyKey(data, now)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(data)
	if err != nil || len(raw) > 64*1024 {
		return "", gameError("history_limit", "单月月报超出保存上限。")
	}
	snapshot := MonthlySnapshot{key, estimated, now.UnixMilli(), raw}
	err = s.Update(provider, choice, revision, func(v *MonthlyArchive) error {
		i := slices.IndexFunc(v.Items, func(item MonthlySnapshot) bool { return item.Month == key })
		if i >= 0 {
			v.Items[i] = snapshot
		} else {
			if len(v.Items) >= 120 {
				return gameError("history_limit", "已保存 120 个月，请先移除旧记录。")
			}
			v.Items = append(v.Items, snapshot)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *ReminderStore) tickMonthly(task *Reminder, now int64, query func(Reminder) (QueryResult, error), send func(Reminder, string) error, game Game) error {
	task.NextCheckMS = nextChallengeCheck(now, task.Hour, task.Minute, 0)
	task.LastCheckedMS = now
	result, err := query(*task)
	if errors.Is(err, errStepUnfinished) {
		return nil
	}
	if err != nil {
		task.LastCode = PublicError(err).Code
		switch task.LastCode {
		case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.upstream_auth_invalid":
			task.Enabled = false
		case "plugin.upstream_device_required", "plugin.upstream_challenge_required":
			task.NextCheckMS = max(task.NextCheckMS, now+int64(24*time.Hour/time.Millisecond))
		}
		return s.save(*task)
	}
	task.LastCode = "collected"
	if task.Notify {
		task.LastAttemptMS = now
	}
	if err = s.save(*task); err != nil {
		return err
	}
	if task.Notify {
		if err = send(*task, game.Name+"默认月份月报已保存 · "+task.Role.UID+" · "+asText(result.Data["data_month"])); err != nil {
			task.LastCode = "collected.notification_failed"
			return s.save(*task)
		}
	}
	return nil
}
