package app

import (
	"context"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// PoolImage is 卡池 as miao's gacha/gacha-info draws it: the banners of a
// version, or those that featured one character or weapon, which show only
// the item's own row unless Detail asks for the whole banner.
type PoolImage struct {
	Pools  []PoolInfo
	Item   Entry
	Detail bool
}

// PoolImageBuilder draws PoolImage.
type PoolImageBuilder func(ImageContext, PoolImage) (Image, bool)

// poolCommand answers miao's 卡池信息: a version's banners (x.x卡池,
// x.x上半卡池) and a character's or weapon's (名称卡池, 名称卡池详情), events
// before chronicled wishes as miao lists poolDetail before mixPoolDetail.
func (a *App) poolCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	page := PoolImage{}
	label := ""
	matches := func(pool PoolInfo) bool { return false }
	if command == "calendar" {
		version, half := args[0], ""
		if len(args) > 1 && (args[1] == "上半" || args[1] == "下半") {
			half = args[1]
		}
		label = a.Game.Name + version + half
		matches = func(pool PoolInfo) bool { return pool.Version == version && (half == "" || pool.Half == half) }
	} else {
		label = args[0]
		page.Detail = len(args) > 1 && (args[1] == "详情" || args[1] == "详细")
		aliases := a.aliasMap(event)
		entry, ok := a.Catalog.Resolve(args[0], "character", aliases)
		if !ok {
			entry, ok = a.Catalog.Resolve(args[0], "weapon", aliases)
		}
		if ok {
			page.Item = entry
			matches = func(pool PoolInfo) bool { return slices.Contains(allBannerNames(pool, entry.Kind), entry.Name) }
		}
	}
	for _, kind := range []string{"event", "chronicled"} {
		for _, pool := range a.Game.Data.Resources.Pools {
			if pool.Kind == kind && matches(pool) {
				page.Pools = append(page.Pools, pool)
			}
		}
	}
	if len(page.Pools) == 0 {
		return event.SendText("未找到 " + label + " 的卡池信息")
	}
	view := View{Title: a.Game.Name + "卡池", Rows: []Row{}}
	for _, pool := range page.Pools {
		names := []string{}
		for _, group := range [][]string{pool.Characters5, pool.Characters4, pool.Weapons5, pool.Weapons4} {
			names = append(names, group...)
		}
		view.Rows = append(view.Rows, Row{Label: strings.TrimSpace(pool.Version + " " + pool.Half), Value: strings.Join(names, "、")})
	}
	if a.poolImage != nil {
		if drawn, ok := a.poolImage(a.imageContext(ctx), page); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}
