package app

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeClock is a clock tests move by hand; sleeping moves it on. Events the
// SDK runs read it from goroutines of their own, so it is locked.
type fakeClock struct {
	mu sync.Mutex
	at time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	if d > 0 {
		c.at = c.at.Add(d)
	}
	c.mu.Unlock()
	return ctx.Err()
}

// set moves the clock to at.
func (c *fakeClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = at
}

// 原石任务 moves to the background before the first account, then saves every
// account's report and answers as Yunzai does.
func TestMonthlyTaskSavesEveryAccountInTheBackground(t *testing.T) {
	a := pluginApp(t)
	a.clock = &fakeClock{at: time.Now()}
	month := int(time.Now().In(time.FixedZone("UTC+8", 8*3600)).Month())
	accounts := &fakeAccounts{t: t, data: map[string]map[string]any{"genshin.monthly": {"data_month": month, "month_data": map[string]any{"current_primogems": 1000}}}}
	host := newSDKHost(t, a, nil, accounts.answer)
	end, actions := host.message("#原石任务", "原石任务")
	if _, ok := detached(actions); !ok || end["type"] != "result" {
		t.Fatalf("the task ended with %v after %+v", end, actions)
	}
	if len(host.sent) != 3 || sentText(host.sent[0].Message) != "开始任务：保存原石数据，完成前请勿重复执行" || !strings.HasPrefix(sentText(host.sent[1].Message), "札记ck：1个") || sentText(host.sent[2].Message) != "原石任务完成" {
		t.Fatalf("answered %+v", host.sent)
	}
	if archive, err := a.Monthly.Read("raylea.mihoyo-accounts", Selection{AccountRef: "account", RoleRef: "role"}); err != nil || len(archive.Items) != 1 {
		t.Fatalf("saved %+v, %v", archive, err)
	}
}

// 推送公告 moves to the background, then checks every group's push; the
// pushes are its only answer.
func TestContentPushCommandChecksEveryGroupInTheBackground(t *testing.T) {
	a := pluginApp(t)
	a.Content = PublicContentClient{HTTP: &recentNews{}}
	host := newSDKHost(t, a, map[string]any{"image_replies": false}, (&fakeAccounts{t: t}).answer)
	for _, group := range []string{"g", "g2"} {
		host.chat(map[string]any{"type": "group", "id": group}, "admin", "#开启公告推送", "开启公告推送")
	}
	sent := len(host.sent)
	end, actions := host.message("#推送公告", "推送公告")
	if _, ok := detached(actions); !ok || end["type"] != "result" {
		t.Fatalf("the command ended with %v after %+v", end, actions)
	}
	pushed := map[string]bool{}
	for _, message := range host.sent[sent:] {
		if strings.Contains(sentText(message.Message), "版本更新说明") {
			pushed[message.TargetID] = true
		}
	}
	if len(host.sent) != sent+2 || !pushed["g"] || !pushed["g2"] {
		t.Fatalf("pushed %+v", host.sent[sent:])
	}
}
