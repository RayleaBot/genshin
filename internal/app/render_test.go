package app

import "testing"

// A render that times out leaves no time for the summary card: the view is
// not drawn again with it.
func TestRenderTimeoutSkipsFallbackTemplate(t *testing.T) {
	a := pluginApp(t)
	a.helpImage = func(ImageContext, HelpImage) (Image, bool) {
		return Image{Template: "help"}, true
	}
	host := newSDKHost(t, a, map[string]any{"image_replies": true}, nil)
	host.renderFailure = "platform.render_timeout"
	_, actions := host.groupMessage("#原神帮助", "原神帮助")
	var templates []string
	for _, action := range actions {
		if action.Name == "render.image" {
			templates = append(templates, asText(action.Data["template"]))
		}
	}
	if len(templates) != 1 || templates[0] != "help" {
		t.Fatalf("rendered templates = %v, want only help", templates)
	}
}
