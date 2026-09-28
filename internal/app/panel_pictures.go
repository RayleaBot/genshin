package app

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// miao's picture uploads: 上传X照片 adds to the photos 照片 and the character
// card draw from (character-img/<name>/upload there, the image library here),
// 上传X面板图 to the pictures a panel draws in place of the splash
// (profile/normal-character/<name>), with 删除X面板图N and X面板图列表.

// PanelPictureStore keeps the 面板图 uploaded in chat, a directory per
// character, named by their MD5 as miao names them.
type PanelPictureStore struct {
	mu        sync.Mutex
	Directory string
}

// List names a character's pictures in the order miao numbers them.
func (s *PanelPictureStore) List(id string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, _ := os.ReadDir(filepath.Join(s.Directory, id))
	names := []string{}
	for _, entry := range entries {
		if entry.Type().IsRegular() && slices.Contains(pictureExtensions, strings.ToLower(filepath.Ext(entry.Name()))) {
			names = append(names, entry.Name())
		}
	}
	return names
}

// Path is a picture's path under the plugin data directory, for templates.
func (s *PanelPictureStore) Path(id, name string) string {
	return filepath.Base(s.Directory) + "/" + id + "/" + name
}

func (s *PanelPictureStore) Read(id, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.Directory, id, filepath.Base(name)))
}

// Add keeps a picture unless the same picture is there already.
func (s *PanelPictureStore) Add(id string, data []byte) error {
	mime, _, _, err := mediaInfo(data)
	if err != nil {
		return err
	}
	sum := md5.Sum(data)
	name := hex.EncodeToString(sum[:]) + "." + strings.TrimPrefix(mime, "image/")
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Join(s.Directory, id), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Directory, id, name), data, 0o644)
}

// Remove deletes the picture List numbers index, from 1.
func (s *PanelPictureStore) Remove(id string, index int) bool {
	names := s.List(id)
	if index < 1 || index > len(names) {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(filepath.Join(s.Directory, id, names[index-1])) == nil
}

// panelSplash is the picture a panel draws: an uploaded 面板图 at random, else
// miao's splash, with the reference 原图 resends.
func (a *App) panelSplash(panel CharacterPanel, name string) (rayleabot.RenderImageResource, string) {
	if names := a.PanelPictures.List(panel.ID); len(names) > 0 {
		picked := names[rand.IntN(len(names))]
		return rayleabot.RenderImageResource{ID: "splash", Path: a.PanelPictures.Path(panel.ID, picked)}, "panel:" + panel.ID + "/" + picked
	}
	portraits, _ := CharacterFolders(panel.ID, name, "")
	file := portraits + "imgs/splash.webp"
	resource, _ := a.imageContext(context.Background()).ArtworkResource("splash", "miao-plugin", file)
	return resource, "artwork:miao-plugin/" + file
}

// uploadedImages downloads the images of the message and the message it
// replies to, as miao's upload reads them.
func uploadedImages(ctx context.Context, event *rayleabot.EventContext) ([][]byte, string) {
	urls := messageImages(ctx, event)
	if len(urls) == 0 {
		return nil, "消息中未找到图片，请将要发送的图片与消息一同发送或引用要添加的图像.."
	}
	images := [][]byte{}
	for _, url := range urls {
		data, err := downloadFile(ctx, url, mediaMaxBytes)
		if err != nil {
			if strings.Contains(err.Error(), "exceeds") {
				return nil, "添加失败：图片太大了。"
			}
			return nil, "图片下载失败。"
		}
		images = append(images, data)
	}
	return images, ""
}

// pictureCommand answers 上传X照片, 上传X面板图, 删除X面板图N and X面板图列表.
func (a *App) pictureCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if len(args) == 0 {
		return event.Result(map[string]any{"handled": false})
	}
	entry, ok := a.Catalog.Resolve(args[0], "character", a.aliasMap(event))
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	reply := func(text string) error {
		if event.Event.Target.Type == "group" {
			return event.Send(event.Event.Target.Type, event.Event.Target.ID, atSender(event, text)...)
		}
		return event.SendText(text)
	}
	switch command {
	case "photo-upload", "panel-image-upload":
		images, problem := uploadedImages(ctx, event)
		if problem != "" {
			return event.SendText(problem)
		}
		count := 0
		for index, data := range images {
			var err error
			if command == "panel-image-upload" {
				err = a.PanelPictures.Add(entry.ID, data)
			} else {
				_, err = a.Media.add(MediaEntry{Title: entry.Name + "照片" + strconv.Itoa(index+1), Category: "photo", CatalogID: entry.ID, Source: "聊天上传", License: "聊天上传"}, data)
			}
			if err != nil {
				return event.SendText(friendlyError(err))
			}
			count++
		}
		kind := "图片"
		if command == "panel-image-upload" {
			kind = "面板图"
		}
		return reply("成功添加" + strconv.Itoa(count) + "张" + entry.Name + kind + "。")
	case "panel-image-remove":
		index, _ := strconv.Atoi(args[len(args)-1])
		if len(args) < 2 || !a.PanelPictures.Remove(entry.ID, index) {
			return event.SendText("删除失败，请检查序列号是否正确")
		}
		return event.SendText("删除成功")
	}
	names := a.PanelPictures.List(entry.ID)
	if len(names) == 0 {
		return event.SendText("暂无" + entry.Name + "的角色面板图")
	}
	parts := [][]rayleabot.Segment{{rayleabot.Text("当前查看的是" + entry.Name + "面板图,共" + strconv.Itoa(len(names)) + "张，可输入【" + a.Game.Prefix + "删除" + entry.Name + "面板图(序列号)】进行删除")}}
	for index, name := range names {
		data, err := a.PanelPictures.Read(entry.ID, name)
		if err != nil {
			continue
		}
		parts = append(parts, []rayleabot.Segment{rayleabot.Text(strconv.Itoa(index+1) + "."), rayleabot.Image("base64://" + base64.StdEncoding.EncodeToString(data))})
	}
	return a.sendForward(ctx, event, parts)
}

// sendPanelPicture resends the uploaded 面板图 a panel drew, for 原图.
func (a *App) sendPanelPicture(event *rayleabot.EventContext, ref string) error {
	id, name, _ := strings.Cut(ref, "/")
	data, err := a.PanelPictures.Read(id, path.Base(name))
	if err != nil {
		return event.SendText("图片已删除。")
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
}
