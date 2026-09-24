package app

import (
	"context"
	"regexp"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var cloudQQPattern = regexp.MustCompile(`^[1-9][0-9]{4,11}$`)

// arkErrors is ark-plugin's ERROR_MAP (apps/user.js): the reply for each
// retcode ark refuses a request with. 304 names this plugin's verify command
// with the prefix replies use, as the plugin serves only Genshin.
var arkErrors = map[string]string{
	"-1":  "插件版本过低，请更新插件",
	"101": "角色ID不存在",
	"102": "未查询到角色信息",
	"103": "请求参数错误",
	"104": "请求超过速率限制",
	"105": "未知错误",
	"106": "数据过大，请确保导出的数据小于2MB",
	"201": "请求超过速率限制，请5分钟后重试",
	"202": "未发现该用户的数据，请重新导出面板",
	"301": "请求类型仅支持原神/星铁",
	"302": "验证失败，个人签名不匹配，请五分钟后重试",
	"303": "验证失败，请稍后再试",
	"304": "该uid未验证号主，请通过 {prefix}ark验证原神uid 验证uid",
	"305": "验证超时，请重新绑定",
	"306": "验证失败，未获取到签名，请五分钟后重试",
	"307": "服务器中无该uid数据...",
}

// arkError is ark-plugin's dealError: the reply for ark's answer to a request
// that did not succeed. A request that failed has no answer, which upstream
// reads as 未知错误 too.
func arkError(prefix string, result map[string]any) string {
	return strings.ReplaceAll(cmpOr(arkErrors[asText(result["retcode"])], "未知错误"), "{prefix}", prefix)
}

// arkAnswer sends a chat command's ark request as ArkApi.req does and reads
// the answer as ark-plugin's characterRank does: the answer when its retcode
// is 100, else the reply dealError gives.
func (a *App) arkAnswer(ctx context.Context, event *rayleabot.EventContext, route string, body map[string]any) (map[string]any, string) {
	decoded, err := a.arkRequest(ctx, event, route, body)
	if err != nil {
		return nil, arkError(a.Game.Prefix, nil)
	}
	result := asObject(decoded)
	if asText(result["retcode"]) != "100" {
		return nil, arkError(a.Game.Prefix, result)
	}
	return result, ""
}

// needUIDReply is what miao's getTargetUid replies when the sender has no UID
// to read, as ark-plugin's commands take it.
func (a *App) needUIDReply() string {
	return "请先发送【" + a.Game.Prefix + "绑定+你的UID】来绑定查询目标"
}

// arkVerifyCommand is ark-plugin's ark绑定原神uid (arkGetBindUid) and
// ark验证原神uid (arkBindUid) for the UID in use or of a mentioned player:
// 绑定 asks ark for the code to write into the game signature, 验证 has ark
// read the signature and tie the UID to the sender's QQ, so other bots take
// it as verified.
func (a *App) arkVerifyCommand(ctx context.Context, event *rayleabot.EventContext, command string) error {
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil || owner.UID == "" {
		return event.SendText(a.needUIDReply())
	}
	route, body, refusal := arkVerifyRequest(command, owner.UID, chatOwner(event))
	if refusal != "" {
		return event.SendText(refusal)
	}
	result, failure := a.arkAnswer(ctx, event, route, body)
	if failure != "" {
		return event.SendText(failure)
	}
	return event.SendText(arkVerifyReply(command, a.Game.Prefix, result))
}

// arkVerifyRequest is the ark request of 绑定 (cloud-bind) or 验证
// (cloud-verify). ark knows a player only by QQ number, which 验证 sends, so
// a sender that is not a OneBot11 QQ user is refused instead.
func arkVerifyRequest(command, uid string, sender Subject) (route string, body map[string]any, refusal string) {
	body = map[string]any{"uid": uid, "type": arkGame}
	if command == "cloud-bind" {
		return "verify/code", body, ""
	}
	if sender.SourceProtocol != "onebot11" || !cloudQQPattern.MatchString(sender.ActorID) {
		return "", nil, "ark 按 QQ 号验证 UID，当前平台无法验证"
	}
	body["qq"] = sender.ActorID
	return "verify", body, ""
}

// arkVerifyReply is ark-plugin's reply when ark accepts: the code with its
// directions for 绑定, 验证成功 for 验证.
func arkVerifyReply(command, prefix string, result map[string]any) string {
	if command == "cloud-verify" {
		return "验证成功"
	}
	return "验证码: " + asText(asObject(result["data"])["verifyCode"]) + "\n使用方式：\n原神：派蒙头像——右上角编辑资料——设置签名——填入验证码，待签名审核通过后输入 " + prefix + "ark验证原神uid\n验证码有效期24小时，验证通过后自动与QQ绑定，在其他Bot上无需再次绑定"
}
