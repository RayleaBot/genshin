package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/localdata"
)

func TestAccountIdentityIgnoresAdapter(t *testing.T) {
	first := Subject{SourceProtocol: "onebot11", SourceAdapter: "deleted", BotID: "123", ActorID: "456"}
	second := first
	second.SourceAdapter = "replacement"
	if !first.same(second) || interactionRef(first) != interactionRef(second) || notePushRef("genshin", first) != notePushRef("genshin", second) {
		t.Fatal("adapter still participates in subject identity")
	}
	other := second
	other.BotID = "789"
	if first.same(other) {
		t.Fatal("different accounts share an identity")
	}
	scope := GroupScope{Protocol: first.SourceProtocol, Adapter: first.SourceAdapter, BotID: first.BotID, GroupID: "group"}
	replaced := scope
	replaced.Adapter = second.SourceAdapter
	s := &GroupStore{Directory: t.TempDir()}
	if !scope.same(replaced) || s.file(scope) != s.file(replaced) {
		t.Fatal("adapter still participates in group identity")
	}
	if err := s.Update(scope, func(d *GroupData) error { d.Config.Aliases["alias"] = "id"; return nil }); err != nil {
		t.Fatal(err)
	}
	if config, err := s.Config(replaced); err != nil || config.Aliases["alias"] != "id" {
		t.Fatal(config, err)
	}
	raw, err := json.Marshal(map[string]any{"subject": first, "scope": scope})
	if err != nil || strings.Contains(string(raw), "source_adapter") {
		t.Fatal(string(raw), err)
	}
	bots := []rayleabot.Bot{{SourceProtocol: "qqofficial", ID: "123"}, {SourceProtocol: "onebot11", SourceAdapter: "new", ID: "123"}, {SourceProtocol: "onebot11", SourceAdapter: "second", ID: "123"}}
	if !accountOnline(bots, "onebot11", "123") || accountOnline(bots[:1], "onebot11", "123") || accountOnline(bots, "onebot11", "") {
		t.Fatal("incorrect account availability")
	}
}

func writeLegacyGroup(t *testing.T, s *GroupStore, adapter, bot string, data GroupData) string {
	t.Helper()
	scope := json.RawMessage(`{"source_protocol":"onebot11","source_adapter":"` + adapter + `","bot_id":"` + bot + `","group_id":"group"}`)
	sum := sha256.Sum256(scope)
	path := filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
	var value map[string]any
	if err := decodeObject(data, &value); err != nil {
		t.Fatal(err)
	}
	value["scope"] = scope
	if err := localdata.Write(path, value); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGroupMigrationMergesWithoutLosingSettingsOrRecords(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	off := false
	old := GroupData{Revision: 4, Config: GroupConfig{Enabled: &off, Aliases: map[string]string{"old": "1"}}, NotePush: NotePush{Resin: 150}, Rank: []RankEntry{{UID: "uid", CharacterID: "1", UpdatedAtMS: 10}}, Cards: []RoleCardOffer{{ActorID: "member", SavedMS: 10}}}
	oldPath := writeLegacyGroup(t, s, "deleted", "bot", old)
	scope := GroupScope{Protocol: "onebot11", BotID: "bot", GroupID: "group"}
	current := GroupData{Scope: scope, Revision: 8, Config: GroupConfig{Aliases: map[string]string{"new": "2"}}, NotePush: NotePush{Resin: 130}, Rank: []RankEntry{{UID: "uid", CharacterID: "1", UpdatedAtMS: 20}}, Stygian: map[string][]StygianEntry{"season": {{UID: "uid"}}}}
	if err := localdata.Write(s.file(scope), current); err != nil {
		t.Fatal(err)
	}
	if n, err := s.migrate(nil); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	got, err := s.Read(scope)
	if err != nil || got.Config.Enabled == nil || *got.Config.Enabled || len(got.Config.Aliases) != 2 || len(got.Rank) != 1 || got.Rank[0].UpdatedAtMS != 20 || len(got.Cards) != 1 || len(got.Stygian) != 1 || got.NotePush.Resin != 130 {
		t.Fatalf("migration lost data: %+v %v", got, err)
	}
	if _, err := os.Stat(oldPath + ".migrated"); err != nil {
		t.Fatal("original conflict record was not preserved", err)
	}
	before, _ := os.ReadFile(s.file(scope))
	if _, err := s.migrate(nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.file(scope))
	if string(before) != string(after) || strings.Contains(string(after), "source_adapter") {
		t.Fatal("migration was not idempotent")
	}
}

func TestGroupMigrationRecoversMissingBotAndRetriesWriteFailure(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	oldPath := writeLegacyGroup(t, s, "old", "", GroupData{Config: GroupConfig{Aliases: map[string]string{"saved": "1"}}})
	before, _ := os.ReadFile(oldPath)
	if n, err := s.migrate([]rayleabot.Bot{{SourceProtocol: "qqofficial", SourceAdapter: "old", ID: "wrong"}}); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	after, _ := os.ReadFile(oldPath)
	if string(before) != string(after) {
		t.Fatal("unresolved legacy record was changed")
	}
	bots := []rayleabot.Bot{{SourceProtocol: "onebot11", SourceAdapter: "old", ID: "bot"}}
	scope := GroupScope{Protocol: "onebot11", BotID: "bot", GroupID: "group"}
	// A directory at the target file makes its atomic replacement fail.
	if err := os.Mkdir(s.file(scope), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.migrate(bots); err == nil {
		t.Fatal("write failure was swallowed")
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatal("failed migration removed its source", err)
	}
	if err := os.Remove(s.file(scope)); err != nil {
		t.Fatal(err)
	}
	if n, err := s.migrate(bots); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if data, err := s.Read(scope); err != nil || data.Config.Aliases["saved"] != "1" {
		t.Fatal(data, err)
	}
}

func TestSubscriptionAndNoteMigrationPreserveDeduplication(t *testing.T) {
	directory := t.TempDir()
	s := &ContentSubscriptions{Path: filepath.Join(directory, "content-subscriptions.json")}
	owner := Subject{SourceProtocol: "onebot11", BotID: "bot", ActorID: "u"}
	items := []ContentSubscription{{Ref: "old-a", Owner: owner, GroupID: "g", Kinds: []string{"announce"}, Sent: map[string]int64{"post": 10}, WarnedOn: "2026-10-10"}, {Ref: "old-b", Owner: owner, GroupID: "g", Kinds: []string{"info"}, Sent: map[string]int64{"post": 20, "another": 15}, WarnedOn: "2026-10-11"}}
	if err := localdata.Write(s.Path, items); err != nil {
		t.Fatal(err)
	}
	if _, err := s.migrate(nil); err != nil {
		t.Fatal(err)
	}
	got, _ := s.List()
	if len(got) != 1 || len(got[0].Kinds) != 2 || got[0].Sent["post"] != 20 || got[0].Sent["another"] != 15 || got[0].WarnedOn != "2026-10-11" {
		t.Fatal(got)
	}
	r := reminderStore(directory)
	tasks := []Reminder{{Ref: "old-a", Kind: notePushKind, Owner: owner, Enabled: true, ExpiresAtMS: 100, LastAttemptMS: 90, Groups: []GroupScope{{Protocol: "onebot11", BotID: "bot", GroupID: "a"}}}, {Ref: "old-b", Kind: notePushKind, Owner: owner, Enabled: true, ExpiresAtMS: 200, DelegationRef: "bound-to-old-b", LastAttemptMS: 80, Groups: []GroupScope{{Protocol: "onebot11", BotID: "bot", GroupID: "b"}}}}
	for _, task := range tasks {
		if err := localdata.Write(r.file(task.Ref), task); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.migrate(nil); err != nil {
		t.Fatal(err)
	}
	kept, _ := r.List()
	if len(kept) != 1 || kept[0].Ref != "old-b" || kept[0].DelegationRef != "bound-to-old-b" || kept[0].LastAttemptMS != 90 || len(kept[0].Groups) != 2 {
		t.Fatal(kept)
	}
	if _, err := r.migrate(nil); err != nil {
		t.Fatal(err)
	}
	again, _ := r.List()
	if !sameTask(kept, again) {
		t.Fatal("repeat migration changed deduplication state")
	}
}

func TestSelectingAccountRecoversUnresolvedGroupRecords(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	path := writeLegacyGroup(t, s, "removed", "", GroupData{Revision: 3, Rank: []RankEntry{{UID: "old", CharacterID: "1"}}, NotePush: NotePush{Resin: 150}})
	scope := GroupScope{Protocol: "onebot11", BotID: "chosen", GroupID: "group"}
	if err := s.Update(scope, func(d *GroupData) error { d.Cards = []RoleCardOffer{{ActorID: "member"}}; return nil }); err != nil {
		t.Fatal(err)
	}
	a := &App{Groups: s}
	input := map[string]any{"scope": scope, "legacy_ref": strings.TrimSuffix(filepath.Base(path), ".json"), "revision": 3, "config": map[string]any{"enabled": false}}
	if _, err := a.manageGroups("groups.set", input); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(scope)
	if err != nil || len(got.Rank) != 1 || len(got.Cards) != 1 || got.NotePush.Resin != 150 || got.Config.Enabled == nil || *got.Config.Enabled {
		t.Fatal(got, err)
	}
	if _, err := os.Stat(path + ".migrated"); err != nil {
		t.Fatal(err)
	}
}

func TestSimulationMigrationKeepsOldStateAcrossKnownAdapters(t *testing.T) {
	s := &SimulationStore{Directory: t.TempDir()}
	event := &rayleabot.EventContext{Event: rayleabot.Event{SourceProtocol: "onebot11", SourceAdapter: "new", BotID: "bot", Actor: rayleabot.Actor{ID: "user"}, Target: rayleabot.Target{Type: "group", ID: "group"}}, Bots: []rayleabot.Bot{{SourceProtocol: "onebot11", SourceAdapter: "old", ID: "bot"}}}
	oldScope := `["onebot11","old","bot","group","group","user"]`
	state := SimulationState{Version: 1, Total: 100, Used: 20, Owned: map[string]int64{"character": 1}, Requests: []SimulationBatch{{RequestRef: "sent"}}}
	if err := localdata.Write(s.filename(oldScope), state); err != nil {
		t.Fatal(err)
	}
	scope, err := simulationScope(event)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.migrateScope(scope, event); err != nil {
		t.Fatal(err)
	}
	var got SimulationState
	if err = localdata.Read(s.filename(scope), &got); err != nil || got.Total != 100 || got.Used != 20 || got.Owned["character"] != 1 || len(got.Requests) != 1 || got.Scope != scope {
		t.Fatal(got, err)
	}
	if _, err = os.Stat(s.filename(oldScope) + ".migrated"); err != nil {
		t.Fatal(err)
	}
	event.Event.SourceAdapter = "another"
	if replaced, _ := simulationScope(event); scope != replaced {
		t.Fatal("simulator identity still uses the adapter")
	}
}

func TestLegacyReminderSendsToAccountOnReplacementAdapter(t *testing.T) {
	a := pluginApp(t)
	ref := "game.reminder.genshin.legacy"
	task := Reminder{Ref: ref, Owner: Subject{SourceProtocol: "onebot11", BotID: "bot", ActorID: "u"}, Provider: "raylea.mihoyo-accounts", Selection: Selection{AccountRef: "account", RoleRef: "role"}, DelegationRef: "grant:" + ref, Enabled: true, Armed: true, Threshold: 50, ExpiresAtMS: farFuture}
	var value map[string]any
	if err := decodeObject(task, &value); err != nil {
		t.Fatal(err)
	}
	value["owner"] = map[string]any{"source_protocol": "onebot11", "source_adapter": "deleted", "bot_id": "bot", "actor_id": "u"}
	if err := localdata.Write(a.Reminders.file(ref), value); err != nil {
		t.Fatal(err)
	}
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t, data: map[string]map[string]any{"genshin.note": {"current_resin": 160, "max_resin": 200}}}).answer)
	host.jobs[ref] = a.reminderJob(task)
	end, actions := host.trigger(ref)
	if end["type"] != "result" || len(host.sent) != 1 {
		t.Fatal(end, host.sent)
	}
	for _, action := range actions {
		if action.Name != "message.send" {
			continue
		}
		if _, exists := action.Data["source_adapter"]; exists {
			t.Fatal("adapter was sent", action.Data)
		}
		if action.Data["source_protocol"] != "onebot11" || action.Data["bot_id"] != "bot" || action.Data["target_type"] != "private" || action.Data["target_id"] != "u" {
			t.Fatal(action.Data)
		}
	}
	// The persisted pre-send marker still prevents another delivery.
	dueNow(t, a, ref)
	host.trigger(ref)
	if len(host.sent) != 1 {
		t.Fatal("migration caused duplicate delivery")
	}
}
