package app

import (
	"context"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-genshin/internal/reference"
)

// CharacterCardImage is miao's character-card: a random photo of the
// character with its level, constellation, weapon, talents and artifacts
// from the UID's panel, or only its name when there is no panel.
type CharacterCardImage struct {
	Entry Entry
	UID   string
	Panel *CharacterPanel
	// Record is the panel's character in the pinned upstream data.
	Record reference.Character
	// Picture is the photo, Width and Height its size, which decide miao's
	// layout.
	Picture       rayleabot.RenderImageResource
	Width, Height int
}

// CharacterCardImageBuilder draws a CharacterCardImage.
type CharacterCardImageBuilder func(ImageContext, CharacterCardImage) (Image, bool)

var (
	cardUID   = regexp.MustCompile(`(18|[1-9])[0-9]{8}`)
	cardWords = regexp.MustCompile(`#|老婆|老公|卡片`)
)

// characterCard answers miao's check: a message naming a character, with an
// optional UID and 卡片, 老婆 or 老公, draws its card. Anything else is left to
// other plugins.
func (a *App) characterCard(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	name, uid := cardQuery(strings.Join(append([]string{event.Event.Command()}, args...), ""))
	entry, _, found := a.aliasOwner(name, a.aliasMap(event))
	if !found || entry.Kind != "character" {
		return event.Result(map[string]any{"handled": false})
	}
	if _, _, _, ref := a.cardPicture(ctx, entry); ref == "" {
		return event.SendText(entry.Name + "暂无角色图片")
	}
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.characterCardFor(ctx, event, owner, entry)
}

// characterCardFor draws the card of a character for a UID's owner.
func (a *App) characterCardFor(ctx context.Context, event *rayleabot.EventContext, owner panelOwner, entry Entry) error {
	image := CharacterCardImage{Entry: entry, UID: owner.UID}
	var ref string
	if image.Picture, image.Width, image.Height, ref = a.cardPicture(ctx, entry); ref == "" {
		return event.SendText(entry.Name + "暂无角色图片")
	}
	// The Traveler's panel is the twin the player chose.
	ids := []string{entry.ID}
	if entry.ID == "20000000" {
		ids = []string{"10000005", "10000007"}
	}
	for _, id := range ids {
		if panel, err := a.characterPanel(ctx, event, owner, id); err == nil {
			image.Panel = &panel
			image.Record, _ = findBuildCharacter(a.Game.Calc, panel)
			break
		}
	}
	view := View{Title: entry.Name, Subtitle: "UID " + owner.UID, Rows: []Row{}}
	if image.Panel != nil {
		view = PanelView(a.Game, []CharacterPanel{*image.Panel}, owner.UID)
	}
	if a.characterCardImage != nil {
		if drawn, ok := a.characterCardImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	// As miao, 原图 answers with the card's photo.
	if err := a.rememberImage(event, ref); err != nil {
		return event.SendText(friendlyError(err))
	}
	return a.sendView(ctx, event, view)
}

// cardQuery is how miao's check reads a message: the first UID anywhere, and
// the rest without #, 老婆, 老公 and 卡片 as the name.
func cardQuery(text string) (name, uid string) {
	uid = cardUID.FindString(text)
	return strings.TrimSpace(cardWords.ReplaceAllString(strings.Replace(text, uid, "", 1), "")), uid
}

// cardPicture picks the card's photo from the same pool as 照片: imported
// images of the character and its downloaded photos. It returns the render
// resource, the size and the reference 原图 resends.
func (a *App) cardPicture(ctx context.Context, entry Entry) (rayleabot.RenderImageResource, int, int, string) {
	a.Media.mu.Lock()
	all, _ := a.Media.entries()
	a.Media.mu.Unlock()
	media := []MediaEntry{}
	for _, item := range all {
		if item.CatalogID == entry.ID && (item.Category == "photo" || item.Category == "character" || item.Category == "birthday") {
			media = append(media, item)
		}
	}
	photos := a.characterPhotos(entry)
	total := len(media) + len(photos)
	if total == 0 {
		return rayleabot.RenderImageResource{}, 0, 0, ""
	}
	if pick := rand.IntN(total); pick < len(media) {
		item := media[pick]
		path, err := a.Media.cached(item.Ref)
		if err != nil {
			return rayleabot.RenderImageResource{}, 0, 0, ""
		}
		return rayleabot.RenderImageResource{ID: "bg", Path: path}, item.Width, item.Height, item.Ref
	} else {
		file := photos[pick-len(media)]
		raw, err := a.Artwork.Open(file.Source, file.Path)
		if err != nil {
			return rayleabot.RenderImageResource{}, 0, 0, ""
		}
		_, width, height, err := mediaInfo(raw)
		resource, ok := a.imageContext(ctx).ArtworkResource("bg", file.Source, file.Path)
		if err != nil || !ok {
			return rayleabot.RenderImageResource{}, 0, 0, ""
		}
		return resource, width, height, "artwork:" + file.Source + "/" + file.Path
	}
}

// cached writes an imported image out as a file under the plugin data
// directory, for templates, and returns its path relative to that directory.
func (s *MediaStore) cached(ref string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.read(ref)
	if err != nil {
		return "", err
	}
	name := ref + "-" + strconv.FormatUint(file.Entry.Revision, 10) + "." + strings.TrimPrefix(file.Entry.MIME, "image/")
	target := filepath.Join(filepath.Dir(s.Directory), "media-cache", name)
	if _, err := os.Stat(target); err == nil {
		return "media-cache/" + name, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	// An edited image replaces the copy of its earlier revision.
	earlier, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ref+"-*"))
	for _, file := range earlier {
		_ = os.Remove(file)
	}
	if err := os.WriteFile(target, file.Data, 0o644); err != nil {
		return "", err
	}
	return "media-cache/" + name, nil
}
