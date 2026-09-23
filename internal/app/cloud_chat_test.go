package app

import "testing"

func TestArkUsageTextFollowsArk(t *testing.T) {
	// Anonymous quota: no permission or multiplier, no advanced quota.
	anonymous := cloudObject(t, `{"retcode":0,"data":{"auth":{"mode":"ip","ip":"203.0.113.9"},"quota":{"rank":{"minute":{"limit":300,"remaining":300},"hour":{"limit":3000,"remaining":2999},"day":{"limit":10000,"remaining":9988}},"custom":{"normal":{"limit":60,"remaining":56},"advanced":null}}}}`)
	want := "权限类型：普通\n全部请求剩余额度：300/2999/9988\n自定义排名请求额度倍率：1x\n自定义排名普通请求剩余额度：56\n自定义排名高级请求剩余额度：-"
	if got := arkUsageText(anonymous); got != want {
		t.Fatalf("anonymous usage:\n%s", got)
	}
	token := cloudObject(t, `{"retcode":0,"data":{"auth":{"permission":1,"limit_normal":2},"quota":{"rank":{"minute":{"remaining":10}},"custom":{"normal":{"remaining":100},"advanced":{"remaining":20}}}}}`)
	want = "权限类型：高级\n全部请求剩余额度：10/-/-\n自定义排名请求额度倍率：2x\n自定义排名普通请求剩余额度：100\n自定义排名高级请求剩余额度：20"
	if got := arkUsageText(token); got != want {
		t.Fatalf("token usage:\n%s", got)
	}
	for raw, want := range map[string]string{`{"retcode":401,"message":"invalid token"}`: "invalid token", `{"retcode":0}`: "查询失败"} {
		if got := arkUsageText(cloudObject(t, raw)); got != want {
			t.Errorf("%s: %q", raw, got)
		}
	}
}
