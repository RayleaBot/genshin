package app

import (
	"encoding/json"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestAccountIdentityIgnoresAdapter(t *testing.T) {
	event := &rayleabot.EventContext{Event: rayleabot.Event{SourceProtocol: "onebot11", SourceAdapter: "first", BotID: "123", Actor: rayleabot.Actor{ID: "456"}, Target: rayleabot.Target{Type: "group", ID: "group"}}}
	first := chatOwner(event)
	scope := groupScope(event)
	event.Event.SourceAdapter = "replacement"
	second := chatOwner(event)
	replaced := groupScope(event)
	if !first.same(second) || interactionRef(first) != interactionRef(second) || notePushRef("genshin", first) != notePushRef("genshin", second) {
		t.Fatal("adapter still participates in subject identity")
	}
	other := second
	other.BotID = "789"
	if first.same(other) {
		t.Fatal("different accounts share an identity")
	}
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
