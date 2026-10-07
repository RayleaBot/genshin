package images

import "github.com/RayleaBot/genshin/internal/app"

// AtlasIndex draws Atlas's numbered list the way its resource/massage
// text.html does: the sender's name, then each entry beside its number.
// text.css takes its number font from Yunzai's resources/font.
func AtlasIndex(context app.ImageContext, list app.AtlasIndexImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	resources.Artwork("tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf")
	items := []any{}
	for index, entry := range list.Entries {
		items = append(items, map[string]any{"num": index + 1, "name": entry})
	}
	return app.Image{Template: "atlas-index", Data: map[string]any{"name": list.Name, "items": items}, Resources: resources.List}, true
}
