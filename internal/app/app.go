package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/artwork"
	"github.com/RayleaBot/plugin-genshin/internal/gacha"
	"github.com/RayleaBot/plugin-genshin/internal/pluginmeta"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

type Operation struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Command string `json:"command"`
	Input   string `json:"input"`
}
type Game struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Prefix     string      `json:"prefix"`
	Region     string      `json:"region"`
	Operations []Operation `json:"operations"`
	// Hints are the replies of commands that upstream answers only by naming
	// the commands that query, by command ID; {prefix} is the reply prefix.
	Hints map[string]string `json:"hints"`
	// Panels are the 更新面板 rules and replies.
	Panels PanelSettings `json:"panels"`
	// Artwork lists the upstream image repositories an administrator can
	// download into the data directory.
	Artwork []artwork.Source `json:"artwork"`
	// Pictures are where 照片, 老婆 and 图鉴 find downloaded images.
	Pictures Pictures `json:"pictures"`
	// Calc runs this game's pinned upstream calculation scripts.
	Calc *reference.Engine `json:"-"`
	// Data is this game's fixed reference data.
	Data *GameData `json:"-"`
}

// Assets is the data a game plugin compiles in and hands to the shared library.
type Assets struct {
	Game     []byte // game.json
	Catalog  []byte // catalog.json
	Manifest []byte // info.json
	Calc     reference.Profile
	// Resources holds materials, banners and birthdays. The others are optional:
	// only games with that feature provide them.
	Resources   []byte
	Simulation  []byte
	CloudPanels []byte
	Enemies     []byte
	// Xiaoyao holds the alias lists of xiaoyao's 七圣召唤 card 图鉴.
	Xiaoyao []byte
	// Images holds the plugin's own image builders by operation name; Panel
	// draws single-character panels.
	Images map[string]ImageBuilder
	Panel  PanelImageBuilder
	Gacha  GachaImageBuilder
	Help   HelpImageBuilder
	// MonthlyStats draws 统计 from the saved months.
	MonthlyStats MonthlyStatsImageBuilder
	// Calendar draws 日历 from the official announcements.
	Calendar CalendarImageBuilder
	// Entry draws reference pages for a catalog entry, as 天赋 and 图鉴.
	Entry EntryImageBuilder
	// SimulationImage draws a 十连.
	SimulationImage SimulationImageBuilder
	// Rank draws a group's panel ranking; CloudRank draws ark's custom
	// ranking on the same page.
	Rank      RankImageBuilder
	CloudRank CloudRankImageBuilder
	// StygianRank draws ark's group 幽境危战 ranking.
	StygianRank StygianRankImageBuilder
	// Showcase is the public service 更新面板 reads without an account;
	// PanelList draws 面板列表.
	Showcase  ShowcaseSource
	PanelList PanelListImageBuilder
	// Characters draws 角色; Training draws 练度统计, 天赋统计 and
	// 剧诗练度统计.
	Characters CharactersImageBuilder
	Training   CharactersImageBuilder
	// ArtifactList draws 圣遗物列表; DailyMaterial draws
	// 今日素材.
	ArtifactList  ArtifactListImageBuilder
	DailyMaterial DailyMaterialImageBuilder
	// UIDList draws 我的uid; RankStats draws 排名统计.
	UIDList   UIDListImageBuilder
	RankStats RankStatsImageBuilder
	// Statistics draws miao's public statistics pages.
	Statistics StatisticsImageBuilder
	// CharacterCard draws miao's card for a bare character name.
	CharacterCard CharacterCardImageBuilder
	// Pools draws 卡池.
	Pools PoolImageBuilder
	// PayLog draws 充值记录.
	PayLog PayLogImageBuilder
	// RoleCards draws 月谕圣牌 and 月谕圣牌交换.
	RoleCards RoleCardsImageBuilder
	// AtlasIndex draws Atlas's numbered lists that cannot be forwarded.
	AtlasIndex AtlasIndexImageBuilder
	// Queries picks the official query, by operation name, for commands that
	// draw on another operation's data, as 最新深渊 runs whichever record
	// opened last.
	Queries map[string]func(now time.Time) string
}
type Settings struct {
	AccountProvider string            `json:"account_provider"`
	ImageReplies    bool              `json:"image_replies"`
	CustomAliases   map[string]string `json:"custom_aliases"`
	// PokeCard is miao's avatarPoke: a poke shows a character card.
	PokeCard bool `json:"poke_card"`
	// AliasPermission is who may change custom aliases, as Yunzai's
	// abbrSetAuth: 0 group members, 1 group administrators, 2 super
	// administrators.
	AliasPermission int `json:"alias_permission"`
	// Ark holds the ark-plugin settings (config/system/cfg_system.js) that
	// apply here.
	Ark ArkSettings `json:"ark"`
	// Xiaoyao holds the xiaoyao-cvs-plugin settings that apply here.
	Xiaoyao XiaoyaoSettings `json:"xiaoyao"`
}

// ArkSettings are ark-plugin's settings under its names, with its defaults.
type ArkSettings struct {
	// PanelRank is panelRank: a panel with a damage table shows ark's ranks.
	PanelRank bool `json:"panel_rank"`
	// QueryType is queryType, the panel's ranks: 0 damage, 1 artifacts, 2
	// both, 3 both as 排名统计 charts.
	QueryType int `json:"query_type"`
	// RankType is RankType, how a rank is written: 0 the place, 1 the
	// percent, 2 both.
	RankType int `json:"rank_type"`
	// LocalPanelRank is localPanelRank: ark ranks the panel itself rather
	// than the UID's panel it keeps.
	LocalPanelRank bool `json:"local_panel_rank"`
	// MarkRankType is markRankType: ranks of local data are marked (本地),
	// or (面板变换) for a changed panel.
	MarkRankType bool `json:"mark_rank_type"`
	// ProfileChangeDiff is profileChangeDiff: a changed panel shows how each
	// damage differs from the kept panel's.
	ProfileChangeDiff bool `json:"profile_change_diff"`
	// DealLongDmgTitle is DealLongDmgTitle, what a changed panel does with
	// damage titles past 20 widths: 0 nothing, 1 cut them, 2 wrap them.
	DealLongDmgTitle int `json:"deal_long_dmg_title"`
	// ProfileChangeOCR is profileChangeOCR: 面板换装 reads the attached or
	// quoted artifact screenshots with ark's OCR.
	ProfileChangeOCR bool `json:"profile_change_ocr"`
	// ExportPanelData and ImportPanelData are exportPanelData and
	// importPanelData, who may 导出面板数据 and 导入面板数据: 0 anyone, 1
	// users with an account and super administrators, 2 super administrators,
	// 3 no one.
	ExportPanelData int `json:"export_panel_data"`
	ImportPanelData int `json:"import_panel_data"`
	// ExportPanelRequire is exportPanelRequire, who may 导出面板: 0 anyone,
	// 1 users with an account or whose UID ark verified for their QQ, 2 users
	// with an account, 3 no one.
	ExportPanelRequire int `json:"export_panel_require"`
	// GroupRank is groupRank: a group ranking shows ark's global ranks.
	GroupRank bool `json:"group_rank"`
	// LocalGroupRank is localGroupRank: ark ranks the listed panels
	// themselves rather than the UIDs' panels it keeps.
	LocalGroupRank bool `json:"local_group_rank"`
	// StygianRank is stygianRank: 更新面板 in a group enters the sender's
	// UID in the group's 幽境危战 ranking.
	StygianRank bool `json:"stygian_rank"`
	// StygianDataFrom is stygianDataFrom, where 幽境危战排名 reads global
	// places: 0 ark, 1 akasha.cv, 2 both.
	StygianDataFrom int `json:"stygian_data_from"`
	// NewUserPanel is newUserPanel: 更新面板 of a UID without kept panels
	// first takes the panels ark keeps for it.
	NewUserPanel bool `json:"new_user_panel"`
}
type App struct {
	Manifest pluginmeta.Manifest
	Media    *MediaStore
	// PanelPictures are the 面板图 uploaded in chat.
	PanelPictures  *PanelPictureStore
	Artwork        *artwork.Store
	Interactions   *InteractionStore
	GuideSettings  *GuideSettings
	Subscriptions  *ContentSubscriptions
	pushLists      pushLists
	ContentJobs    ContentJobs
	Billing        *BillingStore
	BillingJobs    BillingJobs
	Content        PublicContentClient
	CloudArchive   *CloudArchiveStore
	CloudTransfers CloudTransfers
	Monthly        *MonthlyStore
	Game           Game
	Catalog        Catalog
	Gacha          *gacha.Store
	Transfers      gacha.Transfers
	Syncs          gacha.Syncs
	// fileImports are the senders 导入记录 is waiting on for a file.
	fileImports fileImports
	// fullLinks are the senders whose next link reads the whole history;
	// LinkHTTP reads the official wish history and customer service logs
	// (nil uses a default client).
	fullLinks fullLinks
	LinkHTTP  *http.Client
	// PayLogs keep each sender's 充值记录; PayKeys hold the customer service
	// links' authkeys in memory.
	PayLogs *PayLogStore
	PayKeys payKeys
	// BackgroundSyncs are the rounds of background syncs being or last read;
	// SyncTasks are the daily syncs.
	BackgroundSyncs backgroundSyncs
	SyncTasks       *SyncTaskStore
	Showcase        ShowcaseClient
	Profiles        *PanelStore
	Reminders       *ReminderStore
	PanelHistory    *PanelHistoryStore
	Groups          *GroupStore
	Simulation      *SimulationStore
	Cloud           CloudClient
	BuildPresets    *BuildPresetStore

	commands           commandSet
	images             map[string]ImageBuilder
	queries            map[string]func(now time.Time) string
	panel              PanelImageBuilder
	gacha              GachaImageBuilder
	helpImage          HelpImageBuilder
	monthlyStats       MonthlyStatsImageBuilder
	calendarImage      CalendarImageBuilder
	entryPage          EntryImageBuilder
	simulationImage    SimulationImageBuilder
	rankImage          RankImageBuilder
	cloudRankImage     CloudRankImageBuilder
	stygianRankImage   StygianRankImageBuilder
	showcase           ShowcaseSource
	panelList          PanelListImageBuilder
	charactersImage    CharactersImageBuilder
	trainingImage      CharactersImageBuilder
	artifactListImage  ArtifactListImageBuilder
	dailyMaterialImage DailyMaterialImageBuilder
	poolImage          PoolImageBuilder
	statisticsImage    StatisticsImageBuilder
	characterCardImage CharacterCardImageBuilder
	payLogImage        PayLogImageBuilder
	roleCardsImage     RoleCardsImageBuilder
	atlasIndexImage    AtlasIndexImageBuilder
	stats              *statisticsCache
	uidListImage       UIDListImageBuilder
	rankStatsImage     RankStatsImageBuilder
	atlases            atlasLibraries
	atlasMenus         atlasMenus
	guides             guideCache
	emoticons          newsEmoticons
	aliases            customAliases
	maps               mapImages
	usageOnce          sync.Once
	// clock is nil for the wall clock.
	clock clock
}

func New(assets Assets, directory string) (*App, error) {
	var game Game
	if err := json.Unmarshal(assets.Game, &game); err != nil || game.ID == "" {
		return nil, fmt.Errorf("game description is invalid")
	}
	catalog, err := ParseCatalog(assets.Catalog)
	if err != nil {
		return nil, err
	}
	manifest, err := pluginmeta.Read(assets.Manifest)
	if err != nil {
		return nil, err
	}
	if manifest.ID != "raylea."+game.ID {
		return nil, fmt.Errorf("manifest %s does not belong to game %s", manifest.ID, game.ID)
	}
	commands, err := newCommandSet(manifest)
	if err != nil {
		return nil, err
	}
	if game.Calc, err = reference.New(assets.Calc); err != nil {
		return nil, err
	}
	if game.Data, err = parseGameData(assets); err != nil {
		return nil, err
	}
	if directory == "" {
		return nil, fmt.Errorf("plugin data directory is required")
	}
	return &App{commands: commands, images: assets.Images, queries: assets.Queries, panel: assets.Panel, gacha: assets.Gacha, helpImage: assets.Help, monthlyStats: assets.MonthlyStats, calendarImage: assets.Calendar, entryPage: assets.Entry, simulationImage: assets.SimulationImage, rankImage: assets.Rank, cloudRankImage: assets.CloudRank, stygianRankImage: assets.StygianRank, showcase: assets.Showcase, panelList: assets.PanelList, charactersImage: assets.Characters, trainingImage: assets.Training, artifactListImage: assets.ArtifactList, dailyMaterialImage: assets.DailyMaterial, poolImage: assets.Pools, statisticsImage: assets.Statistics, characterCardImage: assets.CharacterCard, payLogImage: assets.PayLog, roleCardsImage: assets.RoleCards, atlasIndexImage: assets.AtlasIndex, PayLogs: &PayLogStore{Path: filepath.Join(directory, "pay-log.json")}, stats: &statisticsCache{entries: map[string]statisticsEntry{}}, uidListImage: assets.UIDList, rankStatsImage: assets.RankStats, Profiles: &PanelStore{Directory: filepath.Join(directory, "profiles")}, Manifest: manifest, Media: &MediaStore{Directory: filepath.Join(directory, "media")}, PanelPictures: &PanelPictureStore{Directory: filepath.Join(directory, "panel-images")}, Artwork: &artwork.Store{Root: filepath.Join(directory, "assets"), Sources: game.Artwork}, Interactions: &InteractionStore{Path: filepath.Join(directory, "interactions.json")}, GuideSettings: &GuideSettings{Path: filepath.Join(directory, "guides.json")}, Subscriptions: &ContentSubscriptions{Path: filepath.Join(directory, "content-subscriptions.json")}, Billing: &BillingStore{Directory: filepath.Join(directory, "billing")}, CloudArchive: &CloudArchiveStore{Directory: filepath.Join(directory, "cloud-archive")}, Monthly: &MonthlyStore{Directory: filepath.Join(directory, "monthly")}, Game: game, Catalog: catalog, BuildPresets: &BuildPresetStore{Path: buildPresetPath(directory)}, Gacha: &gacha.Store{Directory: filepath.Join(directory, "gacha"), Game: game.ID}, SyncTasks: syncTaskStore(directory), Reminders: reminderStore(directory), PanelHistory: &PanelHistoryStore{Directory: filepath.Join(directory, "panels")}, Groups: &GroupStore{Directory: filepath.Join(directory, "groups")}, Simulation: &SimulationStore{Game: game.ID, Deck: game.Data.Simulation, Directory: filepath.Join(directory, "simulation")}}, nil
}
func settings(event *rayleabot.EventContext) Settings {
	value := Settings{AccountProvider: "raylea.mihoyo-accounts", ImageReplies: true, PokeCard: true, CustomAliases: map[string]string{},
		Ark: ArkSettings{PanelRank: true, QueryType: 3, RankType: 2, LocalPanelRank: true, ProfileChangeDiff: true, DealLongDmgTitle: 1, ProfileChangeOCR: true, ExportPanelData: 1, ImportPanelData: 2, ExportPanelRequire: 1, GroupRank: true, LocalGroupRank: true, StygianRank: true, StygianDataFrom: 2}, Xiaoyao: XiaoyaoSettings{NoteSetAuth: 2}}
	_ = decodeObject(event.Config, &value)
	return value
}
func decodeObject(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
func (a *App) accountClient(event *rayleabot.EventContext) AccountsClient {
	return AccountsClient{Caller: event.Actions(), Provider: settings(event).AccountProvider, Game: a.Game.ID}
}
func (a *App) operation(name string) (Operation, bool) {
	for _, item := range a.Game.Operations {
		if item.Name == name {
			return item, true
		}
	}
	return Operation{}, false
}

// routedQuery is the official query a command runs, query unless the command
// draws on another operation's query, and the operation its text reply reads
// as.
func (a *App) routedQuery(operation Operation, query string) (string, Operation) {
	if route := a.queries[operation.Name]; route != nil {
		query = route(time.Now())
		if queried, ok := a.operation(query); ok {
			return query, queried
		}
	}
	return query, operation
}

func (a *App) Handle(ctx context.Context, event *rayleabot.EventContext) error {
	// Replies name commands with the first prefix the host gives this plugin;
	// the list is fixed for the process session.
	a.usageOnce.Do(func() {
		if len(event.CommandPrefixes) > 0 {
			a.Game.Prefix = event.CommandPrefixes[0]
		}
	})
	if event.Event.EventType == "notice.poke" {
		return a.handlePoke(ctx, event)
	}
	if event.Event.EventType == "scheduler.trigger" {
		// A trigger names the task it runs; one this plugin no longer keeps
		// deletes its job.
		task := event.Event.TaskID()
		switch {
		case task == "":
			return event.Fail("plugin.game_source_invalid", "任务来源无效。")
		case strings.HasPrefix(task, "game.content."):
			return a.runContentSubscription(ctx, event)
		case strings.HasPrefix(task, "game.sync."):
			return a.runSyncTask(ctx, event)
		case strings.HasPrefix(task, notePushTask):
			return a.runNotePush(ctx, event)
		}
		return a.runReminder(ctx, event)
	}
	if event.Event.EventType == "management.action" {
		if event.Event.SourceProtocol != "management" || event.Event.SourceAdapter != "management.ui" {
			return event.Fail("plugin.game_source_invalid", "管理动作来源无效。")
		}
		result, err := a.Manage(ctx, event, asText(event.Event.Payload["action"]), asObject(event.Event.Payload["payload"]))
		if err != nil {
			failure := PublicError(err)
			return event.FailDetails(failure.Code, failure.Message, failure.Details)
		}
		return event.Result(result)
	}
	if event.Event.EventType == "config.changed" {
		a.aliases.observe(settings(event).CustomAliases)
	}
	if event.Event.EventType != "message.private" && event.Event.EventType != "message.group" {
		return event.Result(map[string]any{"handled": false})
	}
	if handled, err := a.pictureMessage(ctx, event); handled {
		return err
	}
	if handled, err := a.gachaLinkMessage(ctx, event); handled {
		return err
	}
	if handled, err := a.payLinkMessage(ctx, event); handled {
		return err
	}
	if handled, err := a.postLinkMessage(ctx, event); handled {
		return err
	}
	if handled, err := a.gachaFileMessage(ctx, event); handled {
		return err
	}
	command, args, known := a.commands.resolve(event.Event.Command(), event.Event.Args())
	if !known {
		return event.Result(map[string]any{"handled": false})
	}
	prefix := a.Game.Prefix
	if event.Event.EventType == "message.group" {
		if command == "group-settings" {
			return a.groupSettings(event)
		}
		config, readErr := a.Groups.Config(groupScope(event))
		if readErr != nil {
			return event.SendText(friendlyError(readErr))
		}
		applyGroupConfig(event, config)
		if config.Enabled != nil && !*config.Enabled && command != "help" && command != "version" && command != "unsubscribe" && command != "challenge-withdraw" && command != "challenge-stop" && command != "challenge-status" && command != "community-stop" && command != "community-progress" && command != "cloud-game-stop" && !(command == "signin-task" && len(args) > 0 && args[0] == "关闭") {
			return event.Result(map[string]any{"handled": false})
		}
	}
	if hint, ok := a.Game.Hints[command]; ok {
		return event.SendText(strings.ReplaceAll(hint, "{prefix}", prefix))
	}
	if static, ok := a.Game.Pictures.Static[command]; ok {
		return a.staticCommand(event, static)
	}
	var view View
	var err error
	switch command {
	case "role-cards", "role-cards-exchange":
		return a.cardsCommand(ctx, event, command, args)
	case "birthday":
		return a.birthdayCommand(ctx, event, args)
	case "gacha-export", "gacha-import":
		return a.gachaFileCommand(ctx, event, command, args)
	case "panel-export":
		return a.panelExportCommand(ctx, event)
	case "help", "version":
		return a.helpCommand(ctx, event, command, args)
	case "artwork", "artwork-status":
		return a.artworkCommand(event, command, args)
	case "photo", "original-image":
		return a.interactionCommand(ctx, event, command, args)
	case "interaction":
		return a.wifeCommand(ctx, event, args)
	case "guides", "guide-help", "guide-default", "map", "enemy", "blueprint":
		return a.resourceToolsCommand(ctx, event, command, args)
	case "subscribe", "unsubscribe", "content-push":
		return a.subscriptionCommand(ctx, event, command, args)
	case "news", "info", "events", "search", "estimate":
		return a.newsCommand(ctx, event, command, args)
	case "live-calendar":
		return a.calendarCommand(ctx, event)
	case "alias-set", "alias-remove", "alias-list":
		return a.aliasCommand(ctx, event, command, args)
	case "codes", "redeem":
		return a.assetCommand(ctx, event, command, args)
	case "monthly-history":
		return a.monthlyCommand(ctx, event, command, args)
	case "monthly-task":
		return a.monthlyTask(ctx, event)
	case "challenge-remind", "challenge-stop", "challenge-status":
		return a.challengeReminderCommand(ctx, event, command, args)
	case "challenge-submit", "challenge-withdraw", "challenge-clear", "challenge-rank":
		return a.challengeCommand(ctx, event, command, args)
	case "gacha-background":
		return a.syncTaskCommand(ctx, event, args)
	case "community-progress":
		return a.communityProgress(ctx, event)
	case "community-task", "community-stop", "cloud-game-task", "cloud-game-stop":
		return a.accountTaskCommand(ctx, event, command, args)
	case "community-status", "community-sign", "cloud-game-status", "cloud-game-sign":
		return a.communityCommand(ctx, event, command, args)
	case "signin-task":
		return a.signinTaskCommand(ctx, event, args)
	case "signin", "signin-status":
		return a.signinCommand(ctx, event, command, args)
	case "simulation", "simulation-history", "simulation-reset":
		return a.simulationCommand(ctx, event, command, args)
	case "simulation-fate":
		return a.simulationCommand(ctx, event, command, args)
	case "rank", "rank-top", "rank-reset", "rank-refresh", "rank-switch":
		return a.rankCommand(ctx, event, command, args)
	case "panel-refresh", "panel-refresh-account", "panel-list", "panel-delete":
		return a.panelCommand(ctx, event, command, args)
	case "talent-refresh":
		return a.talentRefresh(ctx, event)
	case "artifact-list":
		return a.artifactList(ctx, event, args)
	case "panel-change":
		return a.panelChangeCommand(ctx, event)
	case "cloud-reforge":
		return a.reforgeCommand(ctx, event)
	case "daily-material":
		return a.dailyMaterial(ctx, event, args)
	case "characters":
		return a.characters(ctx, event, args)
	case "training", "talent-stat":
		return a.training(ctx, event, command, args)
	case "cloud-character-rank", "cloud-total-rank", "cloud-rank-stats":
		return a.cloudChatCommand(ctx, event, command, args)
	case "cloud-custom-rank":
		return a.customRankCommand(ctx, event)
	case "cloud-custom-rank-help":
		return a.customRankHelp(ctx, event)
	case "cloud-custom-rank-panel":
		return a.customRankPanel(ctx, event, args)
	case "cloud-usage":
		return a.cloudUsage(ctx, event)
	case "cloud-stygian-rank":
		return a.stygianRankCommand(ctx, event, args)
	case "cloud-export", "cloud-import":
		return a.cloudExchangeCommand(ctx, event, command, args)
	case "cloud-bind", "cloud-verify":
		return a.arkVerifyCommand(ctx, event, command)
	case "calendar", "banner-history":
		return a.poolCommand(ctx, event, command, args)
	case "materials":
		if len(args) == 0 {
			return event.SendText("使用“" + prefix + "材料 名称”查询固定参考资料。")
		}
		if answered, err := a.materialCommand(ctx, event, args); answered {
			return err
		}
		action := "materials.query"
		input := map[string]any{"query": strings.Join(args, " ")}
		result, queryErr := a.resourceQuery(action, input)
		if queryErr != nil {
			err = queryErr
			break
		}
		view = resourceView(a.Game, action, result)
	case "growth":
		// As the Yunzai calculator, target levels may follow the word: the
		// character, the weapon, then each talent; a 9–10 digit number is a UID.
		levels, uid, bad := []int{}, "", false
		for _, arg := range args[min(1, len(args)):] {
			for _, token := range strings.FieldsFunc(arg, func(r rune) bool { return r == ',' || r == '，' }) {
				n, convErr := strconv.Atoi(token)
				switch {
				case convErr != nil || n < 0:
					bad = true
				case len(token) >= 9:
					uid = token
				default:
					levels = append(levels, n)
				}
			}
		}
		if len(args) < 1 || bad {
			return event.SendText("使用“" + prefix + "角色名养成[目标等级…] [UID]”（如“" + prefix + "刻晴养成”或“" + prefix + "刻晴养成81 90 9 9 9”）计算升到目标等级的材料，不写目标时算到上限。")
		}
		input, _, parseErr := a.commandInput(Operation{Input: "characters"}, args[:1], a.aliasMap(event))
		if parseErr != nil {
			err = parseErr
			break
		}
		client := a.accountClient(event)
		listed, listErr := client.List(ctx, 0)
		if listErr != nil {
			err = listErr
			break
		}
		choice, _, chooseErr := Choose(listed, a.Game.ID, uid)
		if chooseErr != nil {
			err = chooseErr
			break
		}
		parameters := map[string]any{"account_ref": choice.AccountRef, "role_ref": choice.RoleRef, "character_id": asText(input["character_ids"].([]any)[0])}
		prepared, prepareErr := a.growthAction(ctx, client, "growth.prepare", parameters)
		if prepareErr != nil {
			err = prepareErr
			break
		}
		plan := prepared["plan"].(GrowthPlan)
		target := func(current, top, index int) int {
			if index < len(levels) {
				return max(current, min(levels[index], top))
			}
			return top
		}
		plan.Target = target(plan.Current, plan.Max, 0)
		plan.WeaponTarget = target(plan.WeaponCurrent, plan.WeaponMax, 1)
		next := 2
		for i := range plan.Skills {
			// Levels name the talents that level up, in the calculator's order.
			if plan.Skills[i].Max > 1 {
				plan.Skills[i].Target = target(plan.Skills[i].Current, plan.Skills[i].Max, next)
				next++
			} else {
				plan.Skills[i].Target = plan.Skills[i].Max
			}
		}
		parameters["plan"] = plan
		computed, computeErr := a.growthAction(ctx, client, "growth.compute", parameters)
		if computeErr != nil {
			err = computeErr
			break
		}
		view = computed["view"].(View)
	case "note-push", "note-push-group":
		return a.notePushCommand(ctx, event, command, args)
	case "enemy-level":
		return a.enemyLevelCommand(event, args)
	case "stat-cons", "stat-usage", "abyss-team":
		return a.statisticsCommand(ctx, event, command)
	case "character-card":
		return a.characterCard(ctx, event, args)
	case "pay-log", "pay-log-update", "pay-log-refresh":
		return a.payLogCommand(ctx, event, command)
	case "theater-training":
		return a.theaterTraining(ctx, event, args)
	case "photo-upload", "panel-image-upload", "panel-image-remove", "panel-image-list":
		return a.pictureCommand(ctx, event, command, args)
	case "build":
		// As miao, a number of fewer than nine digits picks the damage detail.
		index, rest := 0, []string{}
		for position, arg := range args {
			if number, convErr := strconv.Atoi(arg); position > 0 && convErr == nil && len(arg) < 9 {
				index = number
				continue
			}
			rest = append(rest, arg)
		}
		if len(rest) < 1 || len(rest) > 2 {
			return event.SendText("使用“" + prefix + "角色名伤害 [UID]”（如“" + prefix + "胡桃伤害”）查看固定参考情境下的角色伤害。")
		}
		panel, uid, panelErr := a.commandPanel(ctx, event, rest)
		if panelErr != nil {
			err = panelErr
			break
		}
		built, buildErr := a.panelDamage(ctx, panel, a.enemyLevel(event), &index)
		if buildErr != nil {
			err = buildErr
			break
		}
		view = a.panelView(ctx, event, panel, uid, true, "", &built)
	case "score":
		if len(args) < 1 || len(args) > 2 {
			return event.SendText("使用“" + prefix + "评分 角色 [UID]”，或按上游写法在角色名后接圣遗物。")
		}
		panel, uid, panelErr := a.commandPanel(ctx, event, args)
		if panelErr != nil {
			err = panelErr
			break
		}
		panel, err = a.scorePanel(ctx, panel)
		if err == nil {
			view = PanelView(a.Game, []CharacterPanel{panel}, uid)
		}
	case "talent-wiki":
		entry, ok := a.Catalog.Resolve(strings.Join(args, " "), "character", a.aliasMap(event))
		if !ok && changeWord.MatchString(event.Event.Command()) {
			// As in miao, a word the wiki cannot read may be a 面板换装, such as
			// 希儿换满行迹.
			return a.panelChangeCommand(ctx, event)
		}
		if !ok {
			return event.SendText("未找到该角色，请使用角色全名或别名。")
		}
		view = EntryView(a.Game, entry)
		view.Image = a.entryImage(ctx, event, command, entry)
	case "catalog":
		query := strings.Join(args, " ")
		entries := a.Catalog.Search(query, "", 20, a.aliasMap(event))
		names := []string{query}
		if len(entries) == 1 {
			view = EntryView(a.Game, entries[0])
			view.Image = a.entryImage(ctx, event, command, entries[0])
			names = []string{entries[0].Name, query}
		}
		// Without a drawn page, the downloaded xiaoyao 图鉴 answers; the Atlas
		// libraries have answered what they have before any command.
		if view.Image == nil {
			if file, ok := a.xiaoyaoPicture(names); ok {
				return a.sendArtwork(event, file)
			}
		}
		if len(entries) == 1 {
			if hint := a.pictureHint(a.Game.Pictures.catalogSources()...); view.Image == nil && hint != "" {
				view.Note = strings.TrimSpace(view.Note + " " + hint)
			}
		} else {
			view = View{Title: a.Game.Name + "图鉴", Subtitle: "资料版本 " + a.Catalog.Version, Rows: []Row{}}
			for _, entry := range entries {
				view.Rows = append(view.Rows, Row{Label: entry.Name, Value: kindLabel(entry.Kind) + " · " + entry.ID})
			}
			if len(entries) == 0 {
				view.Note = "未找到匹配资料，请尝试角色、装备全名或 ID。"
			}
		}
	case "character":
		if len(args) < 1 || len(args) > 2 {
			return event.Result(map[string]any{"handled": false})
		}
		panel, uid, panelErr := a.commandPanel(ctx, event, args)
		if panelErr != nil {
			err = panelErr
			break
		}
		view = a.fullPanelView(ctx, event, panel, uid, true, "")
	case "gacha-full":
		return a.fullLinkCommand(event)
	case "gacha-help-port":
		return a.gachaHelpPort(event)
	case "accounts", "select", "uid-remove":
		return a.uidCommand(ctx, event, command, args)
	case "gacha", "gacha-versions", "gacha-detail", "gacha-stat":
		// As on Miao-Yunzai, miao's pages take every word their rules match;
		// Yunzai's keep the rest.
		word := event.Event.Command()
		miao := command == "gacha-detail" || command == "gacha-stat"
		uid := ""
		if len(args) > 0 {
			uid = args[0]
		}
		archive, role, readErr := a.chatArchive(ctx, event, uid)
		missing := "UID:" + role.UID + " 本地暂无抽卡信息，请通过【" + prefix + "抽卡帮助】获得绑定帮助..."
		if readErr != nil {
			if miao && PublicError(readErr).Code == "archive_missing" {
				return event.SendText(missing)
			}
			err = readErr
			break
		}
		game := a.Game
		view = GachaView(game, archive)
		if strings.HasSuffix(word, "统计") {
			view = versionDrawView(game, versionDraws(game, archive))
		}
		if a.gacha != nil {
			image := GachaImage{UID: role.UID, Role: role, Word: word, Archive: archive, Group: event.Event.Target.Type == "group", Miao: miao}
			if miao {
				saved, _ := a.Profiles.Read(role.UID)
				image.Nickname, image.Face = saved.Nickname, saved.Face
			}
			drawn, ok := a.gacha(a.imageContext(ctx), image)
			switch {
			case ok:
				view.Image = &drawn
			case miao:
				return event.SendText(missing)
			case command == "gacha":
				// Yunzai answers 全部记录 without any pool's records as it
				// answers a UID without records.
				err = a.noGachaRecords()
			}
		}
	default:
		var operation Operation
		matched := false
		for _, item := range a.Game.Operations {
			// Operation names use underscores where command IDs use hyphens.
			if item.Name == a.Game.ID+"."+strings.ReplaceAll(command, "-", "_") {
				operation = item
				matched = true
				break
			}
		}
		if !matched {
			return event.Result(map[string]any{"handled": false})
		}
		if operation.Input == "month" && gluedMonth(event.Event.Command(), args) && uidPattern.MatchString(args[0]) {
			// A number no month has, written with the word, is this month.
			args = args[1:]
		}
		input, uid, parseErr := a.commandInput(operation, args, a.aliasMap(event))
		if parseErr != nil {
			err = parseErr
			break
		}
		listed, listErr := a.accountClient(event).List(ctx, 0)
		if listErr != nil {
			err = listErr
			break
		}
		choice, _, chooseErr := Choose(listed, a.Game.ID, uid)
		if chooseErr != nil {
			err = chooseErr
			break
		}
		// A routed command's text reply reads as the query that ran; its image
		// is the command's own.
		query, textOperation := a.routedQuery(operation, operation.Name)
		result, queryErr := a.accountClient(event).Execute(ctx, choice, query, input)
		if queryErr != nil {
			err = queryErr
			break
		}
		if query == a.Game.ID+".monthly" {
			// Once answered, the month read and the others the official
			// report still offers are kept, as upstream's saveLedger; a month
			// that cannot be kept has still been answered.
			client := a.accountClient(event)
			defer func() { _ = a.keepMonthly(ctx, client, choice, result) }()
		}
		view = BusinessView(a.Game, textOperation, result, a.Catalog)
		view.Image = a.featureImage(ctx, a.accountClient(event), choice, operation.Name, event.Event.Command(), input, result)
	}
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, view)
}

func (a *App) commandInput(operation Operation, args []string, aliases map[string]string) (map[string]any, string, error) {
	input := map[string]any{}
	uid := ""
	switch operation.Input {
	case "period":
		// Upstream names the period only in words, as in "上期深渊", which
		// the trigger's period group leads the arguments with.
		if len(args) > 0 {
			if period, ok := map[string]int{"本期": 1, "上期": 2, "往期": 2}[args[0]]; ok {
				input["schedule_type"] = period
				args = args[1:]
			}
		}
	case "month":
		if len(args) > 0 && !uidPattern.MatchString(args[0]) {
			month, err := ledgerMonth(args[0], time.Now().In(time.FixedZone("UTC+8", 28800)))
			if err != nil {
				return nil, "", err
			}
			input["month"] = month
			args = args[1:]
		}
	case "characters":
		if len(args) == 0 {
			return nil, "", gameError("input_invalid", "请提供角色名或角色 ID。")
		}
		id := args[0]
		if entry, ok := a.Catalog.Resolve(id, "character", aliases); ok {
			id = entry.ID
		}
		if _, err := strconv.Atoi(id); err != nil {
			return nil, "", gameError("character_ambiguous", "角色名未唯一匹配，请先查询图鉴或使用角色 ID。")
		}
		input["character_ids"] = []any{id}
		args = args[1:]
	}
	if len(args) > 0 {
		uid = args[0]
	}
	return input, uid, nil
}

func (a *App) Manage(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "public.query":
		result, err := a.accountClient(event).PublicProfile(ctx, asText(input["uid"]), asText(input["region"]))
		if err != nil {
			return nil, err
		}
		view := BusinessView(a.Game, Operation{Name: a.Game.ID + ".profile", Label: a.Game.Name + "公开资料"}, result, a.Catalog)
		view.Note = "通过已获所有者授权的公共查询池读取官方公开资料，不代表目标 UID 已绑定。"
		return map[string]any{"view": view}, nil
	case "signin.status", "signin.rewards", "signin.run":
		return a.signinAction(ctx, a.accountClient(event), action, input)
	case "content.start", "content.poll", "content.cancel":
		return a.contentAction(action, input)
	case "cards.query", "cards.group.list", "cards.group.remove":
		return a.cardsAction(ctx, event, action, input)
	case "birthday.list", "birthday.claim":
		return a.birthdayAction(ctx, a.accountClient(event), action, input)
	case "help.query":
		out, _, err := a.help(event, asText(input["query"]))
		return out, err
	case "statistics.teams":
		return a.statisticsTeams(ctx, event, input)
	case "aliases.validate":
		return a.aliasAction(input)
	case "interaction.list", "interaction.remove":
		return a.interactionManage(action, input)
	case "guides.schema", "guides.settings", "guides.configure":
		return a.GuideSettings.Manage(action, input)
	case "enemies.schema", "enemies.query":
		return a.enemyAction(action, input)
	case "map.query":
		return a.mapQuery(input)
	case "blueprint.read", "blueprint.compute":
		return a.blueprintAction(ctx, a.accountClient(event), action, input)
	case "content.subscription.list", "content.subscription.remove":
		return a.subscriptionManage(ctx, event, action, input)
	case "codes.query":
		return a.Content.codes(ctx)
	case "billing.schema":
		return map[string]any{"categories": billingCategories()}, nil
	case "billing.page", "redeem.run":
		return a.assetAccountAction(ctx, a.accountClient(event), action, input)
	case "monthly.list", "monthly.fetch", "monthly.get", "monthly.remove":
		return a.monthlyAction(ctx, a.accountClient(event), action, input)
	case "challenge.reminder.create", "challenge.reminder.list", "challenge.reminder.remove", "challenge.reminder.check":
		return a.challengeReminder(ctx, event, action, input)
	case "challenge.schema", "challenge.list", "challenge.clear":
		return a.manageChallenge(action, input)
	case "groups.list", "groups.clear", "groups.set":
		return a.manageGroups(action, input)
	case "materials.query", "calendar.query":
		return a.resourceQuery(action, input)
	case "banners.query":
		return a.bannerQuery(input)
	case "growth.prepare", "growth.compute":
		return a.growthAction(ctx, a.accountClient(event), action, input)
	case "build.prepare", "build.compare":
		return a.buildAction(ctx, a.accountClient(event), action, input)
	case "build.conditions", "build.presets.list", "build.presets.save", "build.presets.remove":
		return a.buildPresetAction(action, input)
	case "status":
		return map[string]any{"game": a.Game, "catalog_version": a.Catalog.Version, "catalog_entries": len(a.Catalog.Entries), "settings": settings(event), "bots": event.Bots}, nil
	case "accounts.list":
		result, err := a.accountClient(event).List(ctx, number(input["page"]))
		return map[string]any{"accounts": result}, err
	case "accounts.select":
		choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
		err := a.accountClient(event).Select(ctx, choice)
		return map[string]any{"selected": choice}, err
	case "query":
		operation, ok := a.operation(asText(input["operation"]))
		if !ok {
			return nil, gameError("operation_denied", "查询操作不存在。")
		}
		parameters := asObject(input["input"])
		query, textOperation := a.routedQuery(operation, operation.Name)
		result, err := a.accountClient(event).Execute(ctx, Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, query, parameters)
		if err != nil {
			return nil, err
		}
		output := map[string]any{"view": BusinessView(a.Game, textOperation, result, a.Catalog), "result": result}
		if strings.HasSuffix(operation.Name, ".character") {
			output["panels"] = NormalizePanels(result, a.Catalog)
			delete(output, "result")
		}
		return output, nil
	case "panel.score":
		panel, err := a.queryScoredPanel(ctx, a.accountClient(event), Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, asText(input["character_id"]))
		if err != nil {
			return nil, err
		}
		return map[string]any{"panels": []CharacterPanel{panel}, "view": PanelView(a.Game, []CharacterPanel{panel}, "")}, nil
	case "showcase":
		uid := asText(input["uid"])
		saved, profile, err := a.refreshShowcase(ctx, event, uid)
		if err != nil {
			return nil, gameError("showcase_failed", a.showcaseReply(err, uid))
		}
		view := View{Title: a.Game.Name + "公开展柜", Subtitle: uid, Rows: []Row{}, Note: "已保存到此 UID 的面板，聊天中可直接查看这些角色的面板、评分与伤害。"}
		for _, panel := range profile.Panels {
			view.Rows = append(view.Rows, Row{Label: panel.Name, Value: "等级 " + strconv.Itoa(panel.Level) + " · " + strconv.Itoa(panel.Rank)})
		}
		if saved.Nickname != "" {
			view.Subtitle = saved.Nickname + " · " + uid
		}
		return map[string]any{"view": view}, nil
	case "catalog.search":
		entries := a.Catalog.Search(asText(input["query"]), asText(input["kind"]), 50, a.aliasMap(event))
		return map[string]any{"entries": entries, "version": a.Catalog.Version}, nil
	case "catalog.get":
		entry, ok := a.Catalog.Get(asText(input["id"]))
		if !ok {
			return nil, gameError("entry_missing", "未找到这份资料。")
		}
		return map[string]any{"entry": entry, "view": EntryView(a.Game, entry)}, nil
	default:
		if strings.HasPrefix(action, "billing.") {
			return a.billingManage(ctx, a.accountClient(event), action, input)
		}
		if strings.HasPrefix(action, "media.") {
			return a.mediaAction(action, input)
		}
		if strings.HasPrefix(action, "artwork.") {
			return a.artworkAction(action, input)
		}
		if strings.HasPrefix(action, "community.task.") || strings.HasPrefix(action, "cloudgame.task.") {
			return a.signinTask(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "community.") || strings.HasPrefix(action, "cloudgame.") {
			return a.communityAction(ctx, a.accountClient(event), action, input)
		}
		if strings.HasPrefix(action, "simulation.") {
			return a.simulationAction(event, action, input)
		}
		if strings.HasPrefix(action, "cloud.") {
			return a.cloudAction(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "signin.task.") || strings.HasPrefix(action, "monthly.task.") {
			return a.signinTask(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "panel.history.") {
			return a.panelHistory(ctx, a.accountClient(event), action, input)
		}
		if strings.HasPrefix(action, "reminder.") {
			return a.manageReminder(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "gacha.sync.") {
			return a.manageSync(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "gacha.task.") {
			return a.syncTaskAction(ctx, event, action, input)
		}
		if strings.HasPrefix(action, "gacha.") {
			return a.manageGacha(action, input)
		}
		return nil, gameError("operation_denied", "操作不存在。")
	}
}

func (a *App) manageGacha(action string, input map[string]any) (map[string]any, error) {
	uid, region, ref := asText(input["uid"]), asText(input["region"]), asText(input["ref"])
	switch action {
	case "gacha.list":
		items, err := a.Gacha.List()
		return map[string]any{"items": items}, err
	case "gacha.history":
		var filter gacha.HistoryFilter
		if decodeObject(input, &filter) != nil || filter.Validate() != nil {
			return nil, gameError("input_invalid", "抽卡筛选条件无效。")
		}
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		history, err := gacha.Browse(archive, filter)
		if errors.Is(err, gacha.ErrConflict) {
			return nil, gameError("archive_changed", "档案已更新，请从第一页重新查询。")
		}
		if err != nil {
			return nil, gameError("input_invalid", "抽卡记录或筛选条件无效。")
		}
		return map[string]any{"history": history}, nil
	case "gacha.summary":
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		summary := gacha.Summarize(archive)
		for i := range summary {
			if len(summary[i].Rare) > 100 {
				summary[i].Rare = summary[i].Rare[len(summary[i].Rare)-100:]
			}
		}
		return map[string]any{"summary": summary, "view": GachaView(a.Game, archive)}, nil
	case "gacha.versions":
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		return versionDraws(a.Game, archive), nil
	case "gacha.remove":
		// The archive's background syncs stop and its daily syncs pause; a
		// round merging its last page cannot revive the removed archive.
		a.BackgroundSyncs.stop("archive_removed", func(s BackgroundSync) bool { return s.Role.UID == uid && s.Role.Region == region })
		err := a.SyncTasks.pauseArchive(uid, region)
		if err == nil {
			err = a.Gacha.Remove(uid, region)
		}
		return map[string]any{"removed": err == nil}, err
	case "gacha.import.start":
		var archive gacha.Archive
		if decodeObject(input, &archive) != nil {
			return nil, gameError("input_invalid", "抽卡档案信息无效。")
		}
		archive.Records = nil
		if gacha.Validate(archive) != nil {
			return nil, gameError("input_invalid", "请选择有效 UID、区服与时区。")
		}
		transfer, err := a.Transfers.Start(archive, false)
		return map[string]any{"transfer": transfer}, err
	case "gacha.import.append":
		var payload struct {
			Offset  int            `json:"offset"`
			Records []gacha.Record `json:"records"`
		}
		if decodeObject(input, &payload) != nil {
			return nil, gameError("input_invalid", "记录批次格式无效。")
		}
		for i := range payload.Records {
			record := &payload.Records[i]
			if entry, ok := a.Catalog.Get(record.ItemID); ok {
				if record.Name == "" {
					record.Name = entry.Name
				}
				if record.Rank == "" && entry.Rarity > 0 {
					record.Rank = strconv.Itoa(entry.Rarity)
				}
			}
		}
		accepted, err := a.Transfers.Append(ref, payload.Offset, payload.Records)
		return map[string]any{"accepted": accepted}, err
	case "gacha.import.finish":
		result, err := a.Transfers.Finish(ref, a.Gacha)
		if err != nil {
			return nil, gameError("import_failed", "记录存在格式或内容冲突，原档案已保留。")
		}
		return map[string]any{"added": result.Added, "total": result.Total}, nil
	case "gacha.import.cancel", "gacha.export.close":
		a.Transfers.Close(ref)
		return map[string]any{"closed": true}, nil
	case "gacha.export.start":
		archive, err := a.Gacha.Read(uid, region)
		if err != nil {
			return nil, gameError("archive_missing", "未找到此抽卡档案。")
		}
		transfer, err := a.Transfers.Start(archive, true)
		return map[string]any{"transfer": transfer}, err
	case "gacha.export.read":
		records, more, err := a.Transfers.Read(ref, number(input["offset"]), 500)
		return map[string]any{"records": records, "more": more}, err
	}
	return nil, gameError("operation_denied", "抽卡操作不存在。")
}

func friendlyError(err error) string {
	failure := PublicError(err)
	switch failure.Code {
	case "plugin.service_unavailable":
		return "米游社账号插件未运行，请先启用并扫码登录。"
	case "plugin.vault_locked":
		if failure.Details["reason"] == "native_key_unavailable" {
			return failure.Message
		}
		return "账号库已锁定，请联系机器人管理员解锁。"
	case "plugin.account_caller_denied":
		return "此游戏未获准使用米游社账号，请联系机器人管理员授权。"
	}
	return failure.Message
}

// sendView replies with the view's own template when it has one, then with
// the generic summary card, and with text when image replies are off or both
// renders fail.
func (a *App) sendView(ctx context.Context, event *rayleabot.EventContext, view View) error {
	if path := a.renderView(ctx, event, view); path != "" {
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image(path))
	}
	return event.SendText(view.Text())
}

// renderView draws a view with its own template, else the generic summary
// card, and returns the image path; it is empty when image replies are off
// or rendering failed.
func (a *App) renderView(ctx context.Context, event *rayleabot.EventContext, view View) string {
	return renderImage(ctx, event.Actions(), settings(event).ImageReplies, view)
}

// imageRenderer draws images: an event's actions.
type imageRenderer interface {
	RenderImage(context.Context, rayleabot.RenderImageRequest) (rayleabot.ActionResult, error)
}

// viewMessage is a message of the view: its image, or its text when images
// is off or rendering failed.
func viewMessage(ctx context.Context, host imageRenderer, images bool, view View) []rayleabot.Segment {
	if path := renderImage(ctx, host, images, view); path != "" {
		return []rayleabot.Segment{rayleabot.Image(path)}
	}
	return []rayleabot.Segment{rayleabot.Text(view.Text())}
}

// renderImage is renderView with the host's renderer and whether image
// replies are on.
func renderImage(ctx context.Context, host imageRenderer, images bool, view View) string {
	if !images {
		return ""
	}
	text := view.Text()
	requests := []rayleabot.RenderImageRequest{}
	if view.Image != nil {
		requests = append(requests, rayleabot.RenderImageRequest{Template: view.Image.Template, Output: "png", FallbackText: text, Data: view.Image.Data, Resources: view.Image.Resources})
	}
	data := map[string]any{}
	_ = decodeObject(view, &data)
	requests = append(requests, rayleabot.RenderImageRequest{Template: "summary", Output: "png", FallbackText: text, Data: data})
	for _, request := range requests {
		if result, err := host.RenderImage(ctx, request); err == nil {
			if path := asText(result["image_path"]); path != "" {
				return path
			}
		}
	}
	return ""
}
