package images

import (
	"strconv"

	"github.com/RayleaBot/genshin/internal/app"
)

// uidListArtwork maps the images named in the converted stylesheets (miao's
// common for the layout, the Yunzai 原神插件's html/user/uid-list) to their
// sources and paths.
var uidListArtwork = [][3]string{
	{"Number", "miao-plugin", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "miao-plugin", "resources/common/font/NZBZ.woff"},
	{"YS", "miao-plugin", "resources/common/font/HYWH-65W.woff"},
	{"common-bg-bg-hydro", "miao-plugin", "resources/common/bg/bg-hydro.webp"},
	{"img-icon-check", "yunzai-genshin", "resources/img/icon/check.webp"},
}

// UIDList draws 我的uid the way the Yunzai 原神插件's html/user/uid-list
// does, for 原神: each UID with its number and CK or bound tag, the one in
// use marked, and the player's name and level with the face and name card of the first saved character (miao's faceImgs).
func UIDList(context app.ImageContext, list app.UIDListImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range uidListArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	names := map[string]string{}
	if context.Game.Calc != nil {
		for _, record := range context.Game.Calc.Metadata().Characters {
			if names[record.ID] == "" {
				names[record.ID] = record.Name
			}
		}
	}
	uids := []any{}
	for index, entry := range list.Entries {
		// miao's faceImgs: the first saved character's face and name card,
		// else its common ones.
		face, banner := "", ""
		if name := names[entry.Character]; name != "" {
			path := "resources/meta-gs/character/" + name + "/imgs/"
			face = resources.Artwork("face-"+strconv.Itoa(index), "miao-plugin", path+"face.webp")
			banner = resources.Artwork("banner-"+strconv.Itoa(index), "miao-plugin", path+"banner.webp")
		}
		if face == "" {
			face = resources.Artwork("face-common", "miao-plugin", "resources/common/item/face.webp")
		}
		if banner == "" {
			banner = resources.Artwork("banner-common", "miao-plugin", "resources/meta-gs/character/common/imgs/banner.webp")
		}
		kind := "reg"
		if entry.Account {
			kind = "ck"
		}
		item := map[string]any{"index": index + 1, "uid": entry.UID, "type": kind, "active": entry.Active, "face": face, "banner": banner}
		if entry.Nickname != "" && entry.Level > 0 {
			item["name"], item["level"] = entry.Nickname, entry.Level
		}
		uids = append(uids, item)
	}
	return app.Image{Template: "uid-list", Data: map[string]any{"mark": context.Game.Prefix, "name": "原神", "no_info": "暂无uid信息，可通过更新面板获取信息", "uids": uids}, Resources: resources.List}, true
}
