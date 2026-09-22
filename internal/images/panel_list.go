package images

import (
	"strings"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/game-plugin-kit/reference"
)

// panelListArtwork maps the images named in the converted stylesheets (miao's
// common, character/profile-detail and character/profile-list) that the list
// shows to their paths.
var panelListArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-dmg-rank-bg", "resources/character/imgs/dmg-rank-bg.png"},
	{"character-imgs-mark-icon", "resources/character/imgs/mark-icon.png"},
	{"character-imgs-mark-rank-bg", "resources/character/imgs/mark-rank-bg.png"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
}

// PanelList draws 面板列表 and the reply to 更新面板 the way miao's
// character/profile-list does: each kept character's face with its
// constellation, the characters the refresh updated marked and listed first,
// the UID's places in the group's ranking, and the update service.
func PanelList(context gamekit.ImageContext, list gamekit.PanelListImage) (gamekit.Image, bool) {
	if context.Game.Calc == nil || len(list.Panels) == 0 {
		return gamekit.Image{}, false
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range panelListArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	characters := context.Game.Calc.Metadata().Characters
	chars := []any{}
	for _, saved := range list.Panels {
		panel := saved.Panel
		// The Traveler has a record per element; the panel names its own.
		var record reference.Character
		for _, candidate := range characters {
			if candidate.ID == panel.ID && (record.ID == "" || strings.EqualFold(candidate.Element, panel.Element)) {
				record = candidate
			}
		}
		star := gamekit.Int(record.Data["star"])
		if star != 4 {
			star = 5
		}
		path := "resources/meta-gs/character/" + record.Name + "/imgs/"
		face := resources.Artwork("face-"+panel.ID, "miao-plugin", path+"face-q.webp")
		if face == "" {
			face = resources.Artwork("face-"+panel.ID, "miao-plugin", path+"face.webp")
		}
		char := map[string]any{"face": face, "star": star, "abbr": abbreviation(context.Catalog, record), "cons": panel.Rank, "is_new": list.Updated[panel.ID]}
		if place, ok := list.Ranks[panel.ID]; ok {
			// miao's rankNumber is 15: places beyond it show no number.
			class := place.Rank
			if class >= 15 {
				class = 10
			} else if class > 3 {
				class = 4
			}
			char["rank"] = map[string]any{"place": place.Rank, "class": class, "type": place.Mode}
		}
		chars = append(chars, char)
	}
	demo := "雷神"
	if abbr, _ := chars[0].(map[string]any)["abbr"].(string); abbr != "" {
		demo = abbr
	}
	data := map[string]any{"elem": "hydro", "game": "gs", "prefix": context.Game.Prefix, "uid": list.UID, "demo": demo, "chars": chars, "has_new": len(list.Updated) > 0, "service": list.Service,
		"group_rank": list.Ranks != nil, "rank_since": time.UnixMilli(list.RankSinceMS).In(chinaTime).Format("01-02 15:04")}
	if len(list.Updated) > 0 {
		data["msg"] = "获取角色面板数据成功"
	}
	if list.Profiles.RefreshedAtMS > 0 {
		data["update_time"] = time.UnixMilli(list.Profiles.RefreshedAtMS).In(chinaTime).Format("01-02 15:04")
	}
	return gamekit.Image{Template: "panel-list", Data: data, Resources: resources.List}, true
}
