package app

import (
	"math"
	"testing"
)

// rankSpecificAnswer is ark's answer to rank/specific with percent 0: the
// artifact distribution, then the damage one, both naming the calculation.
const rankSpecificAnswer = `[{"retcode":100,"data":{"scores":["268.70","255.60","248.70","240.40","233.90","223.00","210.80","187.10","170.00","104.90"],"total":60184,"name":"重击伤害","top1":"294.00"}},
	{"retcode":100,"data":{"scores":["68249.76","56693.54","52188.35","48193.57","45706.58","41279.80","35789.10","23696.87","15476.92","4425.55"],"total":60391,"name":"重击伤害","top1":"86620.75"}}]`

// 排名统计 reads ark's two distributions as the panel page reads them, with
// no panel placed on them.
func TestReadRankStatsFollowsPanelPage(t *testing.T) {
	entry := Entry{ID: "10000046", Name: "胡桃"}
	image, err := readRankStats(entry, cloudList(t, rankSpecificAnswer))
	if err != nil || image.DamageTitle != "重击伤害" {
		t.Fatalf("image = %+v %v", image, err)
	}
	if damage := image.Damage; damage.Top != 100 || math.Abs(damage.Scores[0]-78.79) > 0.01 || damage.Total != "60391" || damage.Percent != -100 {
		t.Fatalf("damage = %+v", damage)
	}
	if artis := image.Artis; artis.Top != 294 || artis.Scores[9] != 104.9 || artis.Total != "60184" {
		t.Fatalf("artis = %+v", artis)
	}
	// A failed answer is ark's refusal; one with nothing to draw cannot be
	// read.
	for raw, code := range map[string]string{`{"retcode":102}`: "plugin.game_cloud_rejected", `{"retcode":100,"data":{"scores":[1,2,3,4,5,6,7,8,9,10]}}`: "plugin.game_cloud_invalid", `[{"retcode":102},{"retcode":102}]`: "plugin.game_cloud_invalid"} {
		if _, err := readRankStats(entry, cloudList(t, raw)); PublicError(err).Code != code {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

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

func TestArkPlayerPanelsReadArkPlayerData(t *testing.T) {
	a := pluginApp(t)
	player := cloudObject(t, `{"uid":"100000001","avatars":{"10000046":{"id":10000046,"elem":"pyro","level":90,"promote":6,"cons":0,"talent":{"a":10,"e":10,"q":10},
		"weapon":{"name":"护摩之杖","level":90,"promote":6,"affix":1},"artis":{"1":{"level":20,"name":"魔女的炎之花","star":5,"mainId":14001,"attrIds":[501033,501201]}}}}}`)
	panels := a.arkPlayerPanels(t.Context(), "100000001", player)
	if len(panels) != 1 || panels[0].ID != "10000046" || panels[0].Source != "share" || panels[0].Weapon == nil || panels[0].Weapon.Name != "护摩之杖" {
		t.Fatalf("panels = %+v", panels)
	}
	// Player data of another UID is not kept.
	if panels := a.arkPlayerPanels(t.Context(), "100000002", player); len(panels) != 0 {
		t.Fatalf("kept another UID's panels: %+v", panels)
	}
}

func TestPanelDataRefusalFollowsArkPermission(t *testing.T) {
	const account, master = "为确保数据安全，目前仅允许绑定CK用户导入/导出自己UID的面板数据，请联系Bot主人导入/导出...", "为确保数据安全，目前仅允许主人导入/导出自己UID的面板数据，请联系Bot主人导入/导出..."
	cases := []struct {
		level              int
		account, superUser bool
		want               string
	}{
		{0, false, false, ""},
		{1, true, false, ""}, {1, false, true, ""}, {1, false, false, account},
		{2, true, false, master}, {2, false, true, ""},
		{3, true, true, "当前功能已被禁用..."},
	}
	for _, tc := range cases {
		if got := panelDataRefusal(tc.level, tc.account, tc.superUser); got != tc.want {
			t.Errorf("level %d account %v super %v: %q", tc.level, tc.account, tc.superUser, got)
		}
	}
	if hasAccount(Accounts{Items: []Account{{Roles: nil}}}) || !hasAccount(Accounts{Items: []Account{{}, {Roles: []Role{{UID: "100000001"}}}}}) {
		t.Fatal("account")
	}
}
