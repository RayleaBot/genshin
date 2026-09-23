package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// miao's 剧诗练度统计 (202507剧诗练度统计): the month's Imaginarium Theater
// conditions from nanoka.cc, miao's default source, as text, and 练度统计
// narrowed to the characters the month allows.

// theaterElements are nanoka's element numbers; 0 is an unused slot.
var theaterElements = map[string]string{"2": "pyro", "3": "hydro", "4": "dendro", "5": "electro", "6": "cryo", "7": "anemo", "8": "geo"}

var theaterElementNames = map[string]string{"anemo": "风", "geo": "岩", "electro": "雷", "dendro": "草", "hydro": "水", "pyro": "火", "cryo": "冰"}

// nanokaJSON reads a static.nanoka.cc data file. miao waits five seconds;
// the files often take longer to arrive from mainland networks, so this waits
// fifteen.
func (c PublicContentClient) nanokaJSON(ctx context.Context, path string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://static.nanoka.cc/"+path, nil)
	if err != nil {
		return nil, err
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, gameError("statistics_unavailable", "nanoka.cc 暂时无法访问。")
	}
	var data map[string]any
	decoder := json.NewDecoder(io.LimitReader(response.Body, 8<<20))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// theaterMonth is miao's getMazeId: the months since 2024-07.
func theaterMonth(month string) int {
	number, _ := strconv.Atoi(month[3:])
	return number/100*12 + number%100 - 1 - (4*12 + 7 - 1)
}

// theaterConditions reads a month's opening characters, invited characters
// and elements from nanoka.cc and writes miao's text: the elements, both
// character lists, then the monsters of the hardest difficulty's acts.
func (a *App) theaterConditions(ctx context.Context, month string) (map[string]any, string, error) {
	failed := gameError("statistics_unavailable", "请求 nanoka.cc 数据库出错")
	manifest, err := a.Content.nanokaJSON(ctx, "manifest.json")
	if err != nil {
		return nil, "", failed
	}
	version := asText(asObject(manifest["gi"])["latest"])
	overall, err := a.Content.nanokaJSON(ctx, "gi/"+version+"/rolecombat.json")
	if err != nil || version == "" {
		return nil, "", failed
	}
	id := theaterMonth(month)
	if id < 0 || id >= len(overall) {
		last := len(overall) - 1 + 4*12 + 7 - 1
		return nil, "", gameError("input_invalid", fmt.Sprintf("当前月份不在 nanoka.cc 数据库中\nnanoka.cc 数据库目前可供查询的月份：202407 - 202%d%02d", last/12, last%12+1))
	}
	maze, err := a.Content.nanokaJSON(ctx, "gi/"+version+"/zh/rolecombat/"+strconv.Itoa(id+3)+".json")
	if err != nil {
		return nil, "", failed
	}
	config := asObject(maze["avatar_config"])
	initial, invite, elements := []string{}, []string{}, []string{}
	for _, raw := range asList(config["buff_avatar_list"]) {
		initial = append(initial, asText(asObject(raw)["id"]))
	}
	for _, raw := range asList(config["invite_avatar_list"]) {
		invite = append(invite, asText(raw))
	}
	for _, raw := range asList(config["element_list"]) {
		if element := theaterElements[asText(raw)]; element != "" {
			elements = append(elements, element)
		}
	}
	if len(initial) == 0 || len(invite) == 0 || len(elements) == 0 {
		return nil, "", gameError("statistics_invalid", "在 nanoka.cc 数据库中查询当前月份未获取到开幕角色、特邀角色、限制元素")
	}
	names := func(ids []string) string {
		out := []string{}
		for _, id := range ids {
			if entry, ok := a.Catalog.Get(id); ok {
				out = append(out, entry.Name)
			}
		}
		return strings.Join(out, "、")
	}
	shown := []string{}
	for _, element := range elements {
		shown = append(shown, theaterElementNames[element])
	}
	lines := []string{"限制元素：" + strings.Join(shown, "、"), "开幕角色：" + names(initial), "特邀角色：" + names(invite)}
	// The hardest difficulty's acts with bosses: its last entry by key.
	difficulties := asObject(maze["difficulty_config"])
	keys := make([]string, 0, len(difficulties))
	for key := range difficulties {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return numericLess(keys[i], keys[j]) })
	if len(keys) > 0 {
		hardest := asObject(difficulties[keys[len(keys)-1]])
		for _, rooms := range []struct {
			field string
			acts  [][2]string
		}{{"room", [][2]string{{"第三幕", "3"}, {"第六幕", "6"}, {"第八幕", "8"}, {"第十幕", "10"}}}, {"hard_room", [][2]string{{"圣牌挑战 I", "4"}, {"圣牌挑战 II", "7"}}}} {
			for _, act := range rooms.acts {
				monsters := asList(asObject(asObject(hardest[rooms.field])[act[1]])["monster_preview_list"])
				if monsters == nil {
					continue
				}
				named := []string{}
				for _, monster := range monsters {
					named = append(named, asText(asObject(monster)["name"]))
				}
				lines = append(lines, act[0]+"："+strings.Join(named, "、"))
			}
		}
	}
	return map[string]any{"initial": initial, "invite": invite, "elements": elements}, strings.Join(lines, "\n"), nil
}

// theaterTraining answers 剧诗练度统计 as miao's ProfileStat.roleStat: the
// player data of the UID written after the month's word, a mentioned
// user's or the one in use, read as for 练度统计, then the month's
// conditions, sent with the 练度统计 narrowed to who may enter in one
// forwarded message.
func (a *App) theaterTraining(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 1 {
		uid = args[1]
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	image, answered, err := a.playerData(ctx, event, owner)
	if answered {
		return err
	}
	theater, text, err := a.theaterConditions(ctx, args[0])
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	operation, _ := a.operation(a.Game.ID + ".training")
	view := a.playerView(ctx, event, owner, image, operation, a.trainingImage, map[string]any{"theater": theater})
	path := a.renderView(ctx, event, view)
	if path == "" {
		return event.SendText(text + "\n\n" + view.Text())
	}
	return a.sendForward(ctx, event, [][]rayleabot.Segment{{rayleabot.Text(text)}, {rayleabot.Image(path)}})
}
