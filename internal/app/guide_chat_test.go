package app

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func guidePost(t *testing.T, raw string) map[string]any {
	t.Helper()
	return cloudObject(t, raw)
}

func TestGuidePostImageFollowsEachUpstream(t *testing.T) {
	// 原神 takes the largest image, the later one on a tie.
	post := guidePost(t, `{"post":{"subject":"【原神】珊瑚宫心海攻略"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/a.png","size":"10"},{"url":"https://upload-bbs.miyoushe.com/b.png","size":"30"},{"url":"https://upload-bbs.miyoushe.com/c.png","size":"30"}]}`)
	if got := guidePostImage("1", "珊瑚宫心海", post); got != "https://upload-bbs.miyoushe.com/c.png" {
		t.Fatal(got)
	}
	if got := guidePostImage("1", "胡桃", post); got != "" {
		t.Fatal("another character's post matched")
	}
	// Source 4 names each character's image inside one post.
	post = guidePost(t, `{"post":{"subject":"常驻角色","structured_content":"【胡桃】…image\":\"77\""},"image_list":[{"url":"https://upload-bbs.miyoushe.com/other.png","image_id":"76"},{"url":"https://upload-bbs.miyoushe.com/hutao.png","image_id":"77"}]}`)
	if got := guidePostImage("4", "胡桃", post); got != "https://upload-bbs.miyoushe.com/hutao.png" {
		t.Fatal(got)
	}
}

func TestMapPictureReportsServiceFailures(t *testing.T) {
	a := App{}
	a.Content.HTTP = cloudDoer(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "map.minigg.cn" || req.URL.Query().Get("map_id") != "7" || req.URL.Query().Get("resource_name") != "夜泊石" {
			t.Fatal(req.URL)
		}
		return nil, errors.New("remote error: tls: handshake failure")
	})
	if _, err := a.mapPicture(t.Context(), "夜泊石", "7"); err == nil || !strings.Contains(PublicError(err).Message, "TLS 握手失败") {
		t.Fatal(err)
	}
	a.Content.HTTP = cloudDoer(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":-1,"message":"未找到该资源"}`))}, nil
	})
	if _, err := a.mapPicture(t.Context(), "不存在", ""); err == nil || PublicError(err).Message != "未找到该资源" {
		t.Fatal(err)
	}
	png := "\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16)
	a.Content.HTTP = cloudDoer(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(png))}, nil
	})
	if image, err := a.mapPicture(t.Context(), "甜甜花", ""); err != nil || string(image) != png {
		t.Fatal(err)
	}
}
