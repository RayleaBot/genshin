package images

import (
	"regexp"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

var (
	abyssFloorWord = regexp.MustCompile(`(9|10|11|12|九|十一|十二|十)层`)
	abyssFloors    = map[string]int{"9": 9, "10": 10, "11": 11, "12": 12, "九": 9, "十": 10, "十一": 11, "十二": 12}
)

// abyssFloorArtwork maps the images named in the converted abyss-floor
// stylesheet and page to Yunzai's paths.
var abyssFloorArtwork = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"img-abyss-floor9", "resources/img/abyss/floor9.png"},
	{"img-abyss-floor10", "resources/img/abyss/floor10.png"},
	{"img-abyss-floor11", "resources/img/abyss/floor11.png"},
	{"img-abyss-floor12", "resources/img/abyss/floor12.png"},
	{"img-other-bg105", "resources/img/other/bg105.png"},
	{"img-other-bg4", "resources/img/other/bg4.png"},
	{"img-other-bg5", "resources/img/other/bg5.png"},
	{"star", "resources/img/abyss/star.png"},
	{"fill", "resources/img/other/fill.png"},
}

// AbyssFloor draws one Spiral Abyss floor the way Yunzai's abyss-floor does:
// the floor's stars, then each chamber fought in both halves with its clear
// time, stars and every character's constellation, level and face. The
// command word names the floor.
func AbyssFloor(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	match := abyssFloorWord.FindStringSubmatch(context.Word)
	if match == nil {
		return gamekit.Image{}, false
	}
	number := abyssFloors[match[1]]
	var floor map[string]any
	floors, _ := result.Data["floors"].([]any)
	for _, raw := range floors {
		if item, _ := raw.(map[string]any); gamekit.Int(item["index"]) == number {
			floor = item
		}
	}
	// Upstream answers in text while the floor has no record.
	if floor == nil {
		return gamekit.Image{}, false
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range abyssFloorArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	names := characterNames(context)
	abbreviations := map[string]string{}
	if context.Game.Calc != nil {
		for _, record := range context.Game.Calc.Metadata().Characters {
			abbreviations[record.ID] = abbreviation(record)
		}
	}
	// The constellations come from the account index, as upstream reads them.
	lives := map[string]int{}
	if context.Query != nil {
		if index, err := context.Query("genshin.profile", nil); err == nil {
			avatars, _ := index.Data["avatars"].([]any)
			for _, raw := range avatars {
				avatar, _ := raw.(map[string]any)
				lives[gamekit.Text(avatar["id"])] = gamekit.Int(avatar["actived_constellation_num"])
			}
		}
	}
	rooms := []any{}
	levels, _ := floor["levels"].([]any)
	for _, raw := range levels {
		level, _ := raw.(map[string]any)
		battles, _ := level["battles"].([]any)
		if len(battles) < 2 {
			continue
		}
		halves := []any{}
		for _, rawBattle := range battles {
			battle, _ := rawBattle.(map[string]any)
			avatars := []any{}
			list, _ := battle["avatars"].([]any)
			for _, rawAvatar := range list {
				avatar, _ := rawAvatar.(map[string]any)
				id := gamekit.Text(avatar["id"])
				avatars = append(avatars, map[string]any{"name": abbreviations[id], "life": lives[id], "rarity": gamekit.Text(avatar["rarity"]), "level": gamekit.Text(avatar["level"]),
					"icon": resources.Artwork("face-"+id, "miao-plugin", "resources/meta-gs/character/"+names[id]+"/imgs/face.webp")})
			}
			halves = append(halves, map[string]any{"index": gamekit.Text(battle["index"]), "avatars": avatars})
		}
		first, _ := battles[0].(map[string]any)
		star := gamekit.Int(level["star"])
		rooms = append(rooms, map[string]any{"index": gamekit.Text(level["index"]), "stars": []any{star >= 1, star >= 2, star >= 3}, "battles": halves,
			"time": time.Unix(int64(gamekit.Int(first["timestamp"])), 0).In(chinaTime).Format("2006-01-02 15:04:05")})
	}
	return gamekit.Image{Template: "abyss-floor", Data: map[string]any{
		"uid": result.Role.UID, "floor": number, "star": gamekit.Text(floor["star"]), "max_star": gamekit.Text(floor["max_star"]), "rooms": rooms,
	}, Resources: resources.List}, true
}

// Queries lists the commands that run another operation's query: a single
// abyss floor draws on the abyss record.
func Queries() map[string]func(time.Time) string {
	return map[string]func(time.Time) string{"genshin.abyss_floor": func(time.Time) string { return "genshin.abyss" }}
}
