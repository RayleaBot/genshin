package app

import (
	"context"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// miao's alias commands: 喵喵别名设置, 删除 and 列表 for the custom aliases.
// Custom aliases are the plugin's custom_aliases setting, the one the
// management page edits. They apply to every group, so only super
// administrators change them; 群设置 别名 keeps a group's own.

// customAliases serializes chat changes to custom_aliases. Events of several
// groups run at once, each with the configuration of its start, so the value
// last written, or announced by config.changed, is the base of a change.
type customAliases struct {
	mu     sync.Mutex
	latest map[string]string
}

func (c *customAliases) observe(aliases map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latest = maps.Clone(aliases)
	if c.latest == nil {
		c.latest = map[string]string{}
	}
}

// changeAliases applies edit to the newest custom aliases and writes them
// when edit says so, returning edit's reply.
func (a *App) changeAliases(ctx context.Context, event *rayleabot.EventContext, edit func(map[string]string) (string, bool)) (string, error) {
	a.aliases.mu.Lock()
	defer a.aliases.mu.Unlock()
	current := a.aliases.latest
	if current == nil {
		current = settings(event).CustomAliases
	}
	next := maps.Clone(current)
	if next == nil {
		next = map[string]string{}
	}
	reply, write := edit(next)
	if !write {
		return reply, nil
	}
	if len(next) > 256 {
		return "自定义别名最多 256 个。", nil
	}
	if _, err := event.Actions().ConfigWrite(ctx, map[string]any{"custom_aliases": next}); err != nil {
		return "", err
	}
	a.aliases.latest = next
	return reply, nil
}

// aliasOwner is the entry a word names exactly: by name, a built-in alias or
// a custom one; custom tells which.
func (a *App) aliasOwner(word string, aliases map[string]string) (entry Entry, custom string, found bool) {
	key := strings.ToLower(strings.TrimSpace(word))
	for alias, id := range aliases {
		if strings.ToLower(alias) == key {
			entry, found = a.Catalog.Get(id)
			return entry, alias, found
		}
	}
	for _, entry := range a.Catalog.Entries {
		if strings.ToLower(entry.Name) == key || slices.ContainsFunc(entry.Aliases, func(alias string) bool { return strings.ToLower(alias) == key }) {
			return entry, "", true
		}
	}
	return Entry{}, "", false
}

var miaoAliasInvalid = regexp.MustCompile(`[,，:：\s]`)

func validAlias(alias string) bool {
	return alias != "" && len(alias) <= 64 && !strings.ContainsAny(alias, "\r\n\t")
}

func (a *App) aliasCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	aliases := settings(event).CustomAliases
	reply := func(text string, err error) error {
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.SendText(text)
	}
	switch command {
	case "alias-list":
		names := map[string][]string{}
		order := []string{}
		for alias, id := range aliases {
			entry, ok := a.Catalog.Get(id)
			if !ok {
				continue
			}
			if names[entry.Name] == nil {
				order = append(order, entry.Name)
			}
			names[entry.Name] = append(names[entry.Name], alias)
		}
		if len(order) == 0 {
			return event.SendText("暂无原神自定义别名")
		}
		slices.Sort(order)
		lines := []string{"【原神自定义别名】"}
		for _, name := range order {
			slices.Sort(names[name])
			lines = append(lines, name+"："+strings.Join(names[name], "，"))
		}
		return event.SendText(strings.Join(lines, "\n"))
	case "alias-set":
		if len(args) != 2 {
			return event.SendText("命令格式：" + a.Game.Prefix + "喵喵别名设置 角色名 别名")
		}
		name, alias := args[0], args[1]
		if miaoAliasInvalid.MatchString(alias) || !validAlias(alias) {
			return event.SendText("别名不能为空，且不能包含逗号、冒号或空格")
		}
		if strings.Trim(alias, "0123456789") == "" {
			return event.SendText("别名不能为纯数字，避免与角色ID冲突")
		}
		return reply(a.changeAliases(ctx, event, func(aliases map[string]string) (string, bool) {
			entry, _, ok := a.aliasOwner(name, aliases)
			if !ok || entry.Kind != "character" {
				return "未找到该角色", false
			}
			if owner, _, taken := a.aliasOwner(alias, aliases); taken {
				if owner.ID == entry.ID {
					return "「" + entry.Name + "」已拥有别名「" + alias + "」，无需重复添加", false
				}
				return "别名「" + alias + "」已被角色「" + owner.Name + "」使用，请更换别名", false
			}
			aliases[alias] = entry.ID
			return entry.Name + "：" + alias + " 添加成功。", true
		}))
	case "alias-remove":
		if len(args) != 1 {
			return event.SendText("命令格式：" + a.Game.Prefix + "喵喵别名删除 别名")
		}
		alias := args[0]
		return reply(a.changeAliases(ctx, event, func(aliases map[string]string) (string, bool) {
			if _, custom, _ := a.aliasOwner(alias, aliases); custom != "" {
				delete(aliases, custom)
				return "别名「" + alias + "」删除成功", true
			}
			return "不存在该别名，或该别名为预设，不支持删除", false
		}))
	}
	return event.Result(map[string]any{"handled": false})
}
