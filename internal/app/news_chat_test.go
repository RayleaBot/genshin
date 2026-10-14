package app

import (
	"context"
	"net/http"
	"testing"
)

func TestSearchLeavesSignInWords(t *testing.T) {
	a := pluginApp(t)
	a.Content.HTTP = doerFunc(func(*http.Request) (*http.Response, error) {
		t.Error("sign-in words made a content request")
		return nil, context.Canceled
	})
	host := newSDKHost(t, a, nil, (&fakeAccounts{t: t}).answer)
	for _, word := range []string{"米游社签到", "mys签到", "米游社全部签到", "米游社每日签到"} {
		end, _ := host.message("#"+word, word)
		if asObject(end["data"])["handled"] != false || len(host.sent) != 0 {
			t.Fatalf("%s: %+v, sent %+v", word, end, host.sent)
		}
	}
}
