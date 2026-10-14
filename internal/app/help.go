package app

import (
	"context"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/pluginmeta"
)

// help lists the commands the requester may use, and the same list grouped
// for a plugin help image; query, the management page's search, keeps the
// commands whose name, description or usage holds it.
func (a *App) help(event *rayleabot.EventContext, query string) (map[string]any, HelpImage, error) {
	m := a.Manifest
	disabled := false
	if event.Event.Target.Type == "group" {
		config, err := a.Groups.Config(groupScope(event))
		if err != nil {
			return nil, HelpImage{}, err
		}
		disabled = config.Enabled != nil && !*config.Enabled
	}
	commands := []pluginmeta.Command{}
	for _, c := range m.Commands {
		if disabled && !slices.Contains([]string{"help", "version", "group-settings", "unsubscribe", "challenge-withdraw", "challenge-stop", "challenge-status", "cloud-game-stop", "cloud-game-progress"}, c.ID) {
			continue
		}
		if !mayRun(event, c) {
			continue
		}
		c.Usage = a.usage(c.Usage)
		if query != "" && !strings.Contains(c.Name+c.Description+c.Usage, query) {
			continue
		}
		commands = append(commands, c)
	}
	view := View{Title: a.Game.Name + "帮助", Subtitle: "插件 " + m.Version + " · " + m.License, Rows: []Row{}}
	if disabled {
		view.Note = "本群游戏功能已暂停，当前只列管理与停止操作。"
	}
	// One section per declared group, in manifest order; commands outside every
	// group come last.
	visible := map[string]pluginmeta.Command{}
	for _, c := range commands {
		visible[c.ID] = c
	}
	placed := map[string]bool{}
	image := HelpImage{Title: view.Title, Subtitle: view.Subtitle, Note: view.Note}
	add := func(id, title string, list []pluginmeta.Command) {
		section, group := Section{Title: title, Rows: []Row{}}, HelpGroup{ID: id, Title: title}
		for _, c := range list {
			section.Rows = append(section.Rows, Row{c.Usage, c.Description})
			group.Commands = append(group.Commands, HelpCommand{ID: c.ID, Name: c.Name, Usage: c.Usage, Description: c.Description})
		}
		if len(section.Rows) > 0 {
			view.Sections = append(view.Sections, section)
			image.Groups = append(image.Groups, group)
		}
	}
	for _, group := range m.Groups {
		list := []pluginmeta.Command{}
		for _, id := range group.Commands {
			if c, ok := visible[id]; ok && !placed[id] {
				list = append(list, c)
				placed[id] = true
			}
		}
		add(group.ID, group.Title, list)
	}
	rest := []pluginmeta.Command{}
	for _, c := range commands {
		if !placed[c.ID] {
			rest = append(rest, c)
		}
	}
	add("other", "其他", rest)
	return map[string]any{"plugin": m, "commands": commands, "catalog_version": a.Catalog.Version, "resource_version": resourceVersion(a.Game), "group_enabled": !disabled, "view": view}, image, nil
}

// mayRun follows the host's permission levels, then the settings that choose
// who may change aliases or exchange panel data. The management page lists
// every command; help does not query accounts to check panel data access.
func mayRun(event *rayleabot.EventContext, c pluginmeta.Command) bool {
	if event.Event.EventType == "management.action" {
		return true
	}
	super := event.Event.SourceProtocol == "onebot11" && slices.Contains(event.SuperAdmins, event.Event.Actor.ID)
	rank := 1
	if super {
		rank = 3
	} else if role := event.Event.Actor.Role; role == "owner" || role == "admin" {
		rank = 2
	}
	if rank < map[string]int{"group_admin": 2, "super_admin": 3}[c.Permission] {
		return false
	}
	switch c.ID {
	case "alias-set", "alias-add", "alias-remove":
		return aliasDenied(event) == ""
	case "note-push-group":
		return event.Event.Target.Type == "group" && (super || settings(event).Xiaoyao.NoteSetAuth == 1 && rank >= 2)
	case "panel-export":
		return settings(event).Ark.ExportPanelRequire <= 2
	case "cloud-export":
		return panelDataRefusal(settings(event).Ark.ExportPanelData, true, super) == ""
	case "cloud-import":
		return panelDataRefusal(settings(event).Ark.ImportPanelData, true, super) == ""
	}
	return true
}

func (a *App) helpCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	// Upstream's rules end at the word: with more after it, as in the help
	// another bot answers in the chat, the message is not the command.
	if len(args) > 0 {
		return event.Result(map[string]any{"handled": false})
	}
	if command == "version" {
		return a.sendView(ctx, event, View{Title: a.Game.Name + "版本", Rows: []Row{{"插件", a.Manifest.Version}, {"最低宿主", a.Manifest.MinCoreVersion}, {"图鉴", a.Catalog.Version}, {"材料与卡池", resourceVersion(a.Game)}, {"许可", a.Manifest.License}}, Note: "插件通过宿主的安装/更新入口替换正式安装包。公开资料查询获取最新官方结果，本地固定资料随插件版本更新。"})
	}
	out, image, err := a.help(event, "")
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := out["view"].(View)
	if a.helpImage != nil {
		if drawn, ok := a.helpImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// HelpImage is the help menu the requester may use, grouped as the manifest
// declares, for the plugin's help image.
type HelpImage struct {
	Title, Subtitle, Note string
	Groups                []HelpGroup
}

// HelpGroup is one declared command group; ID is the manifest's group id, or
// "other" for the commands outside every group.
type HelpGroup struct {
	ID, Title string
	Commands  []HelpCommand
}

// HelpCommand is one command with its name, and its usage written with the
// reply prefix.
type HelpCommand struct {
	ID, Name, Usage, Description string
}

// HelpImageBuilder draws the help menu with the plugin's template, or returns
// false to keep the generic summary card.
type HelpImageBuilder func(ImageContext, HelpImage) (Image, bool)
