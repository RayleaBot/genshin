package app

import (
	"context"
	"encoding/base64"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 照片, 老婆 and 图鉴 read the upstream images an administrator downloaded:
// photos the way miao-plugin reads character-img, 图鉴 the way Atlas and
// xiaoyao read their libraries, from the sources game.json declares.

// Pictures are where the downloaded chat images are.
type Pictures struct {
	// Photos are directories of a character's photos; {name} stands for the
	// character.
	Photos []PictureSource `json:"photos"`
	// Atlas are the Atlas libraries, searched in order.
	Atlas []AtlasLibrary `json:"atlas"`
	// Xiaoyao is xiaoyao's 图鉴 library, file paths with {name}.
	Xiaoyao PictureSource `json:"xiaoyao"`
	// Static are commands answered with a fixed upstream image, by command.
	Static map[string]StaticPicture `json:"static"`
	// AtlasHelp is the picture Atlas's atlasHelp answers with.
	AtlasHelp StaticPicture `json:"atlas_help"`
}

type PictureSource struct {
	Source string   `json:"source"`
	Paths  []string `json:"paths"`
}

// photoSources are the sources of the character photos.
func (p Pictures) photoSources() []string {
	sources := []string{}
	for _, source := range p.Photos {
		sources = append(sources, source.Source)
	}
	return sources
}

// catalogSources are the sources of the 图鉴 pictures: the Atlas libraries,
// then xiaoyao's.
func (p Pictures) catalogSources() []string {
	sources := []string{}
	for _, library := range p.Atlas {
		sources = append(sources, library.Source)
	}
	return append(sources, p.Xiaoyao.Source)
}

// artworkFile is a file of an artwork source.
type artworkFile struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

var pictureExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

// characterPhotos lists a character's downloaded photos; the Traveler's are
// both twins', as miao's.
func (a *App) characterPhotos(entry Entry) []artworkFile {
	names := []string{entry.Name}
	if entry.Name == "旅行者" {
		names = []string{"空", "荧"}
	}
	files := []artworkFile{}
	for _, source := range a.Game.Pictures.Photos {
		for _, pattern := range source.Paths {
			for _, name := range names {
				for _, file := range a.Artwork.List(source.Source, strings.ReplaceAll(pattern, "{name}", name)) {
					if slices.Contains(pictureExtensions, strings.ToLower(path.Ext(file))) {
						files = append(files, artworkFile{source.Source, file})
					}
				}
			}
		}
	}
	return files
}

// xiaoyaoPicture finds the first downloaded xiaoyao 图鉴 image of any of the
// names.
func (a *App) xiaoyaoPicture(names []string) (artworkFile, bool) {
	source := a.Game.Pictures.Xiaoyao
	for _, pattern := range source.Paths {
		for _, name := range names {
			file := strings.ReplaceAll(pattern, "{name}", name)
			if _, found := a.Artwork.File(source.Source, file); found {
				return artworkFile{source.Source, file}, true
			}
		}
	}
	return artworkFile{}, false
}

// pictureMessage answers the messages the picture plugins take before the
// ones this plugin follows otherwise: on Miao-Yunzai, Atlas's rule sees every
// message at priority 10, ahead of miao's and Yunzai's rules. handled is
// false for the messages left to the plugin's own handlers.
func (a *App) pictureMessage(ctx context.Context, event *rayleabot.EventContext) (handled bool, err error) {
	if event.Event.Target.Type == "group" {
		if config, err := a.Groups.Config(groupScope(event)); err == nil && config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	// Every message passes here, so the custom aliases are read only when a
	// name is looked up.
	aliases := sync.OnceValue(func() map[string]string { return a.aliasMap(event) })
	// miao's accept checks run before every rule and hand the message on as
	// their own command word, which no picture plugin answers.
	msg := a.miaoAccept(yunzaiMessage(event), aliases)
	// Atlas's atlasHelp comes at priority 9 and passes the message on.
	if atlasHelpWords.MatchString(msg) {
		if _, err := post(ctx, event, a.staticReply(a.Game.Pictures.AtlasHelp)); err != nil {
			return true, err
		}
	}
	run := atlasRun{app: a, owner: chatOwner(event), aliases: aliases}
	ended := run.atlas(msg)
	for _, answer := range run.answers {
		if answer.list != nil {
			err = a.postAtlasList(ctx, event, answer.list)
		} else {
			_, err = post(ctx, event, a.artworkReply(event, answer.picture))
		}
		if err != nil {
			return true, err
		}
	}
	if ended {
		return true, event.Result(map[string]any{"handled": true})
	}
	return false, nil
}

// atlasHelpWords are the words of Atlas's help, after # or /.
var atlasHelpWords = regexp.MustCompile(`^[#/](图鉴|wiki|百科|Atlas)(\s*)(帮助|菜单|功能|help)`)

// yunzaiMessage is a chat message as Yunzai's plugins read it (e.msg): the
// text segments, each trimmed, with # for the prefix the host parsed.
func yunzaiMessage(event *rayleabot.EventContext) string {
	if command := event.Event.Command(); command != "" {
		return "#" + strings.Join(append([]string{command}, event.Event.Args()...), " ")
	}
	if len(event.Event.Message.Segments) == 0 {
		return strings.TrimSpace(event.Event.Message.PlainText)
	}
	text := ""
	for _, segment := range event.Event.Message.Segments {
		if segment.Type == "text" {
			text += strings.TrimSpace(asText(segment.Data["text"]))
		}
	}
	return text
}

// pictureHint tells how to download the sources a reply found nothing in.
func (a *App) pictureHint(sources ...string) string {
	missing := []string{}
	for _, source := range sources {
		if !a.Artwork.Ready(source) && !slices.Contains(missing, source) {
			missing = append(missing, source)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "管理员可发送“" + a.Game.Prefix + "素材更新 " + strings.Join(missing, " ") + "”下载图片素材。"
}

// rememberImage keeps the image just sent for 原图.
func (a *App) rememberImage(event *rayleabot.EventContext, ref string) error {
	return a.Interactions.edit(chatOwner(event), func(p *InteractionProfile) error {
		p.LastImage = ref
		p.LastImageMS = time.Now().UnixMilli()
		p.LastImageTargetType = event.Event.Target.Type
		p.LastImageTargetID = event.Event.Target.ID
		return nil
	})
}

// artworkReply is a downloaded image to send, kept for 原图, or the text that
// tells why it cannot be sent.
func (a *App) artworkReply(event *rayleabot.EventContext, file artworkFile) rayleabot.Segment {
	data, err := a.Artwork.Open(file.Source, file.Path)
	if err != nil {
		return rayleabot.Text("图片素材读取失败，请重新下载素材。")
	}
	if err = a.rememberImage(event, "artwork:"+file.Source+"/"+file.Path); err != nil {
		return rayleabot.Text(friendlyError(err))
	}
	return rayleabot.Image("base64://" + base64.StdEncoding.EncodeToString(data))
}

// sendArtwork sends a downloaded image.
func (a *App) sendArtwork(event *rayleabot.EventContext, file artworkFile) error {
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, a.artworkReply(event, file))
}

// CharacterFolders are the miao folders of a character's pictures, as miao's
// getImgs picks them: portraits under the character's name, which is 空 or 荧
// for the Traveler, and the Traveler's card, banner and constellation icons
// under 旅行者/<element>.
func CharacterFolders(id, name, elem string) (portraits, icons string) {
	switch id {
	case "10000005":
		name = "空"
	case "10000007":
		name = "荧"
	default:
		folder := "resources/meta-gs/character/" + name + "/"
		return folder, folder
	}
	return "resources/meta-gs/character/" + name + "/", "resources/meta-gs/character/旅行者/" + elem + "/"
}
