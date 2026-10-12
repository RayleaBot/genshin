package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/gacha"
	"github.com/RayleaBot/genshin/internal/localdata"
)

func managementEvent(protocol, id string) *rayleabot.EventContext {
	return &rayleabot.EventContext{Event: rayleabot.Event{EventType: "management.action", SourceProtocol: "management", SourceAdapter: "management.ui", Payload: map[string]any{"page_id": "groups", "bot": map[string]any{"source_protocol": protocol, "bot_id": id}}}}
}

func TestManagementRequiresHostAccount(t *testing.T) {
	a := &App{}
	for _, action := range []string{"accounts.list", "accounts.select", "query", "groups.list", "groups.set", "challenge.clear", "cards.group.remove", "reminder.list", "reminder.create", "signin.task.remove", "monthly.list", "community.task.create", "cloudgame.status", "content.subscription.remove", "interaction.remove", "gacha.task.list", "gacha.sync.cancel", "panel.history.get", "billing.sync.cancel", "cloud.archive.export.close", "cloud.poll", "statistics.teams", "blueprint.compute", "growth.prepare"} {
		t.Run(action, func(t *testing.T) {
			event := managementEvent("onebot11", "")
			delete(event.Event.Payload, "bot")
			_, err := a.Manage(t.Context(), event, action, map[string]any{"bot": map[string]any{"source_protocol": "onebot11", "bot_id": "injected"}})
			if err == nil || PublicError(err).Code != "platform.invalid_request" {
				t.Fatalf("%s accepted an unscoped action: %v", action, err)
			}
		})
	}
}

func TestManagementGroupsFilterBeforePaginationAndMergeAccountWrites(t *testing.T) {
	a := &App{Groups: &GroupStore{Directory: t.TempDir()}}
	for _, bot := range []string{"a", "b"} {
		for i := 0; i < 52; i++ {
			if err := a.Groups.Update(GroupScope{Protocol: "onebot11", BotID: bot, GroupID: fmt.Sprint(i)}, func(d *GroupData) error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
	}
	writeLegacyGroup(t, a.Groups, "gone", "", GroupData{})
	for page, want := range []int{50, 2} {
		out, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "groups.list", map[string]any{"page": page})
		if err != nil {
			t.Fatal(err)
		}
		items := out["items"].([]map[string]any)
		if len(items) != want {
			t.Fatalf("page %d: %d", page, len(items))
		}
		for _, item := range items {
			if item["scope"].(GroupScope).BotID != "a" {
				t.Fatal("another account leaked", item)
			}
		}
	}
	for _, input := range []map[string]any{
		{"scope": GroupScope{Protocol: "onebot11", BotID: "b", GroupID: "0"}},
		{"scope": map[string]any{"group_id": "0"}, "owner": Subject{SourceProtocol: "qqofficial", BotID: "a"}},
	} {
		if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "groups.clear", input); err == nil {
			t.Fatal("conflicting identity accepted")
		}
	}
	var wg sync.WaitGroup
	for _, bot := range []string{"a", "b"} {
		wg.Go(func() {
			_, err := a.Manage(t.Context(), managementEvent("onebot11", bot), "groups.set", map[string]any{"scope": map[string]any{"group_id": "new"}, "revision": 0, "config": map[string]any{"image_replies": bot == "a"}})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for _, bot := range []string{"a", "b"} {
		data, err := a.Groups.Read(GroupScope{Protocol: "onebot11", BotID: bot, GroupID: "new"})
		if err != nil || data.Config.ImageReplies == nil || *data.Config.ImageReplies != (bot == "a") {
			t.Fatal("account write lost", data, err)
		}
	}
}

func TestManagementOwnedListsAndMutations(t *testing.T) {
	a := pluginApp(t)
	owner := func(id string) Subject { return Subject{SourceProtocol: "onebot11", BotID: id, ActorID: "u"} }
	reminders := []Reminder{}
	for _, id := range []string{"a", "b", ""} {
		for _, kind := range []string{"", "signin", "monthly", "challenge", "community", "cloudgame"} {
			reminders = append(reminders, Reminder{Ref: id + ":" + kind, Kind: kind, Owner: owner(id)})
		}
	}
	if err := seedReminders(a.Reminders, reminders); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"reminder.list", "signin.task.list", "monthly.task.list", "challenge.reminder.list", "community.task.list", "cloudgame.task.list"} {
		out, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), action, map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		items := out["items"].([]Reminder)
		if len(items) != 1 || items[0].Owner.BotID != "a" {
			t.Fatal(action, items)
		}
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "reminder.remove", map[string]any{"ref": "b:"}); err == nil {
		t.Fatal("removed another account's reminder")
	}
	if err := localdata.Write(a.Subscriptions.Path, []ContentSubscription{{Ref: "a", Owner: owner("a")}, {Ref: "b", Owner: owner("b")}, {Ref: "legacy", Owner: owner("")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "content.subscription.remove", map[string]any{"ref": "b", "confirm": true}); err == nil {
		t.Fatal("removed another account's subscription")
	}
	profiles := []InteractionProfile{{Ref: "a", Owner: owner("a")}, {Ref: "b", Owner: owner("b")}, {Ref: "legacy", Owner: owner("")}}
	if err := localdata.Write(a.Interactions.Path, profiles); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "interaction.remove", map[string]any{"ref": "b", "confirm": true}); err == nil {
		t.Fatal("removed another account's preferences")
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "interaction.remove", map[string]any{"ref": "a", "confirm": true}); err != nil {
		t.Fatal(err)
	}
	kept, _ := a.Interactions.List()
	if len(kept) != 2 || kept[0].Ref != "b" || kept[1].Ref != "legacy" {
		t.Fatal("other preferences lost", kept)
	}
	if err := a.SyncTasks.edit("b", func(items *[]SyncTask, _ int) error {
		*items = append(*items, SyncTask{Ref: "b", Owner: owner("b")})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "gacha.task.remove", map[string]any{"ref": "b"}); err == nil {
		t.Fatal("removed another account's sync")
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "gacha.task.run", map[string]any{"ref": "b", "confirm": true}); err == nil {
		t.Fatal("reran another account's sync")
	}
	_, _, err := a.BackgroundSyncs.begin(t.Context(), BackgroundSync{Ref: "b", Owner: owner("b")}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "gacha.task.cancel", map[string]any{"ref": "b"}); err == nil {
		t.Fatal("canceled another account's round")
	}
	if a.BackgroundSyncs.list()[0].State != "running" {
		t.Fatal("round changed")
	}
}

func TestHistoricalRecordsKeepOtherAccountsAndRejectReassignment(t *testing.T) {
	a := pluginApp(t)
	path := writeLegacyGroup(t, a.Groups, "deleted", "", GroupData{Config: GroupConfig{Aliases: map[string]string{"old": "1001"}}})
	source := a.historicalSources()[0]
	rows, err := source.records()
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	row := rows[0]
	if !row.Assignable || row.BotID != "" {
		t.Fatal(row)
	}
	other := GroupScope{Protocol: "onebot11", BotID: "other", GroupID: "group"}
	if err := a.Groups.Update(other, func(g *GroupData) error { g.Config.Aliases["keep"] = "1002"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err = a.editHistoricalRecord(source, row.Ref, row.Revision, rayleabot.BotAccount{SourceProtocol: "onebot11", BotID: "a"}); err != nil {
		t.Fatal(err)
	}
	current, _ := a.Groups.Read(GroupScope{Protocol: "onebot11", BotID: "a", GroupID: "group"})
	untouched, _ := a.Groups.Read(other)
	if current.Config.Aliases["old"] != "1001" || untouched.Config.Aliases["keep"] != "1002" {
		t.Fatal(current, untouched)
	}
	if _, err = a.editHistoricalRecord(source, row.Ref, row.Revision, rayleabot.BotAccount{}); err == nil {
		t.Fatal("stale ref accepted")
	}
	rows, _ = source.records()
	for _, row := range rows {
		if row.Assignable {
			t.Fatal("complete account can be reassigned")
		}
		if _, err = a.editHistoricalRecord(source, row.Ref, row.Revision, rayleabot.BotAccount{SourceProtocol: "onebot11", BotID: "changed"}); err == nil {
			t.Fatal("reassigned a complete account")
		}
	}
	var old any
	if err := localdata.Read(path+".migrated", &old); err != nil {
		t.Fatal(err)
	}
	legacyQQ := map[string]any{"scope": map[string]any{"source_protocol": "qqofficial", "bot_id": "", "group_id": "opaque"}, "config": map[string]any{}}
	qqRef := historicalRevision(legacyQQ)
	if err := localdata.Write(filepath.Join(a.Groups.Directory, qqRef+".json"), legacyQQ); err != nil {
		t.Fatal(err)
	}
	if _, err = a.editHistoricalRecord(source, qqRef, historicalRevision(legacyQQ), rayleabot.BotAccount{SourceProtocol: "onebot11", BotID: "a"}); err == nil {
		t.Fatal("QQ official record was reassigned")
	}
	if _, err = a.editHistoricalRecord(source, qqRef, historicalRevision(legacyQQ), rayleabot.BotAccount{}); err != nil {
		t.Fatal(err)
	}
}

func TestCloudManagementCannotReadOrCancelAnotherAccountsResult(t *testing.T) {
	bot := rayleabot.BotAccount{SourceProtocol: "onebot11", BotID: "a"}
	client := CloudClient{jobs: map[string]*CloudJob{"job": {Ref: "job", bot: bot, State: "completed", CreatedAtMS: time.Now().UnixMilli(), cancel: func() { t.Error("canceled another account") }}}}
	other := rayleabot.BotAccount{SourceProtocol: "qqofficial", BotID: "a"}
	if _, err := client.poll("job", false, &other); err == nil {
		t.Fatal("cross-protocol read")
	}
	if _, err := client.poll("job", true, &other); err == nil {
		t.Fatal("cross-protocol cancel")
	}
	if _, err := client.poll("job", false, &bot); err != nil {
		t.Fatal(err)
	}
}

func TestManagementServiceDenialIsReturnedWithoutUnscopedRetry(t *testing.T) {
	a := pluginApp(t)
	methods := []string{}
	h := newSDKHost(t, a, nil, func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		methods = append(methods, request.Method)
		for _, key := range []string{"bot", "bot_id", "source_protocol", "subject", "owner"} {
			if _, supplied := request.Params[key]; supplied {
				t.Errorf("caller supplied source %s", key)
			}
		}
		return nil, "plugin.account_subject_denied"
	})
	result, _ := h.manage("accounts.list", map[string]any{})
	if result["code"] != "plugin.account_subject_denied" || !slices.Equal(methods, []string{"list"}) {
		t.Fatal(result, methods)
	}
	methods = nil
	result, _ = h.manageFor(nil, "accounts.list", map[string]any{})
	if result["code"] != "platform.invalid_request" || len(methods) != 0 {
		t.Fatal("unscoped service call", result, methods)
	}
	choice := Selection{AccountRef: "other-account", RoleRef: "role"}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, "100000001", "cn_gf01", false)
	if err != nil {
		t.Fatal(err)
	}
	result, _ = h.manage("gacha.sync.cancel", map[string]any{"ref": info.Ref})
	if result["code"] != "plugin.account_subject_denied" {
		t.Fatal(result)
	}
	if _, err = a.Syncs.Choice(info.Ref); err != nil {
		t.Fatal("denied cancellation changed sync", err)
	}
	ref, err := a.CloudTransfers.start(cloudTransfer{Selection: choice, Provider: "raylea.mihoyo-accounts"})
	if err != nil {
		t.Fatal(err)
	}
	result, _ = h.manage("cloud.archive.import.cancel", map[string]any{"ref": ref})
	if result["code"] != "plugin.account_subject_denied" || a.CloudTransfers.items[ref] == nil {
		t.Fatal("denied cancellation changed transfer", result)
	}
	a.BillingJobs.items = map[string]*BillingJob{"billing": {Selection: choice, Provider: "raylea.mihoyo-accounts", timer: time.NewTimer(time.Hour)}}
	result, _ = h.manage("billing.sync.cancel", map[string]any{"ref": "billing"})
	if result["code"] != "plugin.account_subject_denied" || a.BillingJobs.items["billing"] == nil {
		t.Fatal("denied cancellation changed billing job", result)
	}
}

func TestHistoricalAccountArchivesAndQQRecovery(t *testing.T) {
	a := pluginApp(t)
	owner := Subject{SourceProtocol: "qqofficial", BotID: "opaque-bot"}
	choice := Selection{AccountRef: "account", RoleRef: "role"}
	if _, err := a.PanelHistory.Save("provider", choice, "v1", CharacterPanel{ID: "1", Name: "character"}, owner); err != nil {
		t.Fatal(err)
	}
	if err := a.CloudArchive.Update("provider", choice, 0, Role{UID: "100000001", Region: "cn_gf01"}, nil, false, owner); err != nil {
		t.Fatal(err)
	}
	if err := a.Billing.Write("provider", choice, 0, BillingArchive{Category: "crystal", Direction: "all", Owner: owner}); err != nil {
		t.Fatal(err)
	}
	if err := localdata.Write(a.Monthly.file("provider", choice), MonthlyArchive{Owner: owner}); err != nil {
		t.Fatal(err)
	}
	for _, source := range a.historicalSources() {
		if !slices.Contains([]string{"panel_history", "monthly", "billing", "cloud_archive"}, source.kind) {
			continue
		}
		rows, err := source.records()
		if err != nil || len(rows) != 1 || rows[0].BotID != owner.BotID || rows[0].SourceProtocol != owner.SourceProtocol || rows[0].Assignable {
			t.Fatal(source.kind, rows, err)
		}
		row := rows[0]
		if _, err = source.view(map[string]any{"ref": row.Ref, "revision": row.Revision}); err != nil {
			t.Fatal(err)
		}
		if _, err = a.editHistoricalRecord(source, row.Ref, row.Revision, rayleabot.BotAccount{}); err != nil {
			t.Fatal(err)
		}
		if rows, err = source.records(); err != nil || len(rows) != 0 {
			t.Fatal(rows, err)
		}
	}
	legacy := map[string]any{"source_protocol": "qqofficial", "source_adapter": "relogged", "bot_id": "", "actor_id": "openid"}
	changed, pending := recoverAddresses(legacy, []rayleabot.Bot{{SourceProtocol: "qqofficial", SourceAdapter: "relogged", ID: "new-bot"}})
	if changed || pending != 1 || legacy["bot_id"] != "" {
		t.Fatal("QQ openid rebound by adapter", legacy)
	}
}
