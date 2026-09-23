package app

import (
	"context"
	"net/url"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Yunzai's 角色素材 (material.js): 胡桃素材, 胡桃突破, 胡桃培养 send the
// character's ascension and talent material picture from 友人A's 米游社
// collection, else from two others; 更新 reads the collections again.

// materialCollections are Yunzai's collections in the order it tries them,
// with the index of the picture each post keeps it at.
var materialCollections = []struct {
	id    string
	image int
}{{"428421", 1}, {"1164644", 0}, {"1362644", 2}}

// materialSpecial are the characters whose 友人A post has the picture one
// further on.
var materialSpecial = []string{"雷电将军", "珊瑚宫心海", "菲谢尔", "托马", "八重神子", "九条裟罗", "辛焱", "神里绫华"}

var materialOSS = "?x-oss-process=image//resize,s_1000/quality,q_80/auto-orient,0/interlace,1/format,jpg"

// materialPicture is the address of a character's material picture, "" when
// no collection has a post naming it.
func (c PublicContentClient) materialPicture(ctx context.Context, name string) (string, error) {
	for _, collection := range materialCollections {
		params := url.Values{"gids": {bbsGID}, "order_type": {"2"}, "collection_id": {collection.id}}
		page, err := c.get(ctx, "https://bbs-api.mihoyo.com/post/wapi/getPostFullInCollection?"+params.Encode(), nil)
		if err != nil {
			return "", err
		}
		for _, raw := range asList(page["posts"]) {
			item := asObject(raw)
			if !strings.Contains(asText(asObject(item["post"])["subject"]), name) {
				continue
			}
			index := collection.image
			if collection.id == "428421" && slices.Contains(materialSpecial, name) {
				index++
			}
			images := asList(item["image_list"])
			if index >= len(images) {
				break
			}
			found := asText(asObject(images[index])["url"])
			link, err := url.Parse(found)
			if err != nil || link.Scheme != "https" || !(strings.HasSuffix(link.Hostname(), ".miyoushe.com") || strings.HasSuffix(link.Hostname(), ".mihoyo.com")) {
				return "", gameError("public_invalid", "素材图片地址不在允许范围内。")
			}
			return found + materialOSS, nil
		}
	}
	return "", nil
}

// materialCommand sends a character's material picture; a name that is no
// character is left to the reference material search.
func (a *App) materialCommand(ctx context.Context, event *rayleabot.EventContext, args []string) (bool, error) {
	raw := strings.Join(args, "")
	refresh := strings.Contains(raw, "更新")
	entry, ok := a.Catalog.Resolve(strings.ReplaceAll(raw, "更新", ""), "character", a.aliasMap(event))
	if !ok {
		return false, nil
	}
	if traveler := []string{"10000005", "10000007", "20000000"}; slices.Contains(traveler, entry.ID) {
		return true, event.SendText("暂无主角素材")
	}
	key := a.Game.ID + "/material/" + entry.Name
	link, cached := a.guides.get(key)
	if !cached || refresh {
		var err error
		if link, err = a.Content.materialPicture(ctx, entry.Name); err != nil {
			return true, event.SendText("暂无素材数据，请稍后再试")
		}
		if link == "" {
			// Yunzai answers nothing when no collection names the character.
			return true, event.Result(map[string]any{"handled": false})
		}
		a.guides.put(key, link)
	}
	return true, event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image(link))
}
