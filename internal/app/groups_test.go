package app

import (
	"maps"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestGroupSettingsDoNotChangeGlobalOrOtherBots(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "onebot11", BotID: "bot", GroupID: "group"}
	off := false
	if err := store.Update(scope, func(d *GroupData) error {
		d.Config.ImageReplies = &off
		d.Config.Aliases = map[string]string{"name": "1001"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	other := scope
	other.BotID = "other"
	value, err := store.Read(other)
	if err != nil || value.Config.ImageReplies != nil || len(value.Config.Aliases) != 0 {
		t.Fatal("cross-bot group settings")
	}
	global := map[string]any{"image_replies": true, "custom_aliases": map[string]string{"global": "1002", "name": "1003"}}
	event := &rayleabot.EventContext{Config: global}
	saved, _ := store.Read(scope)
	applyGroupConfig(event, saved.Config)
	if global["image_replies"] != true || global["custom_aliases"].(map[string]string)["name"] != "1003" {
		t.Fatal("global snapshot mutated")
	}
	settings := settings(event)
	if settings.ImageReplies || settings.CustomAliases["global"] != "1002" || settings.CustomAliases["name"] != "1001" {
		t.Fatal(settings)
	}
}
func TestGroupAdministratorMatchesHostRoles(t *testing.T) {
	for _, role := range []string{"member", "admin", "owner"} {
		event := &rayleabot.EventContext{Event: rayleabot.Event{Actor: rayleabot.Actor{ID: "u", Role: role}}}
		if groupAdministrator(event) != (role != "member") {
			t.Fatal(role)
		}
	}
}

func TestGroupManagementRejectsStaleRevision(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "p", BotID: "b", GroupID: "g"}
	app := &App{Groups: store}
	input := map[string]any{"scope": scope, "revision": 0, "config": map[string]any{"enabled": false}}
	if _, err := app.manageGroups("groups.set", input); err != nil {
		t.Fatal(err)
	}
	if _, err := app.manageGroups("groups.clear", input); err == nil || PublicError(err).Code != "plugin.game_group_changed" {
		t.Fatal("stale clear changed group", err)
	}
	data, _ := store.Read(scope)
	if data.Config.Enabled == nil || *data.Config.Enabled {
		t.Fatal("failed clear changed settings")
	}
}

func TestGroupConfigFollowsEveryWrite(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{"onebot11", "bot", "group"}
	if config, err := s.Config(scope); err != nil || config.Enabled != nil {
		t.Fatal(config, err)
	}
	off := false
	if err := s.Update(scope, func(data *GroupData) error { data.Config.Enabled = &off; return nil }); err != nil {
		t.Fatal(err)
	}
	if config, _ := s.Config(scope); config.Enabled == nil || *config.Enabled {
		t.Fatal("the kept settings missed a write", config)
	}
	// Another process start reads the file again.
	if config, _ := (&GroupStore{Directory: s.Directory}).Config(scope); config.Enabled == nil || *config.Enabled {
		t.Fatal(config)
	}
}

func TestAliasChangesFollowYunzaiPermission(t *testing.T) {
	event := func(target, role string, permission int, super bool) *rayleabot.EventContext {
		e := &rayleabot.EventContext{Config: map[string]any{"alias_permission": permission}, Event: rayleabot.Event{Actor: rayleabot.Actor{ID: "u", Role: role}, Target: rayleabot.Target{Type: target, ID: "g"}}}
		if super {
			e.SuperAdmins = []string{"u"}
		}
		return e
	}
	for _, c := range []struct {
		target, role string
		permission   int
		super, ok    bool
	}{
		{"group", "member", 0, false, true},
		{"private", "member", 0, false, false},
		{"private", "member", 2, true, true},
		{"group", "member", 1, false, false},
		{"group", "admin", 1, false, true},
		{"group", "owner", 2, false, false},
		{"group", "member", 2, true, true},
	} {
		if denied := aliasDenied(event(c.target, c.role, c.permission, c.super)); (denied == "") != c.ok {
			t.Errorf("%+v: %q", c, denied)
		}
	}
}

func TestGroupAliasTargetResolvesByName(t *testing.T) {
	a := pluginApp(t)
	host := newSDKHost(t, a, nil, nil)
	for _, tc := range []struct{ alias, target, want string }{
		{"老婆", "胡桃", "已指向胡桃"}, {"老婆", "往生堂堂主", "已指向胡桃"},
		{"专武", "护摩之杖", "已指向护摩之杖"}, {"甲", "不存在的名字", "未找到角色或武器"},
	} {
		end, _ := host.groupMessage("#群设置 别名 "+tc.alias+" "+tc.target, "群设置", "别名", tc.alias, tc.target)
		if text := terminalText(end); !strings.Contains(text, tc.want) {
			t.Fatal(text)
		}
	}
	data, err := a.Groups.Read(GroupScope{"onebot11", "bot", "g"})
	if err != nil || !maps.Equal(data.Config.Aliases, map[string]string{"老婆": "10000046", "专武": "13501"}) {
		t.Fatal(data.Config.Aliases, err)
	}
}
