package app

import (
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func managementInvalid(message string) error {
	return &rayleabot.ActionError{Code: "platform.invalid_request", Message: message}
}

// Actions carry their own scope requirement, even when a caller omits page_id.
func accountManagementAction(action string) bool {
	switch action {
	case "query", "panel.score", "statistics.teams", "redeem.run", "cards.query", "interaction.list", "interaction.remove":
		return true
	case "billing.schema", "challenge.schema", "build.conditions", "cloud.filter_schema":
		return false
	}
	for _, prefix := range []string{"accounts.", "groups.", "cards.group.", "content.subscription.", "reminder.", "signin.", "monthly.", "challenge.", "growth.", "blueprint.", "community.", "cloudgame.", "billing.", "panel.history.", "gacha.sync.", "gacha.task.", "cloud."} {
		if strings.HasPrefix(action, prefix) {
			return true
		}
	}
	return action == "build.prepare" || action == "build.compare"
}

func ownsBot(bot rayleabot.BotAccount, owner Subject) bool {
	return owner.SourceProtocol == bot.SourceProtocol && owner.BotID == bot.BotID && owner.BotID != ""
}

func managementOwns(event *rayleabot.EventContext, owner Subject) bool {
	bot, ok := event.Event.ManagementBot()
	return ok && ownsBot(bot, owner)
}

func checkManagementOwner(event *rayleabot.EventContext, owner Subject) error {
	if event.Event.EventType == "management.action" && !managementOwns(event, owner) {
		return managementInvalid("记录不属于当前机器人账号。")
	}
	return nil
}

// Explicit business identities may agree with the host, but never override it.
func checkManagementIdentity(input map[string]any, bot rayleabot.BotAccount) error {
	if value := asText(input["bot_id"]); value != "" && value != bot.BotID {
		return managementInvalid("机器人账号与当前页面不符。")
	}
	if value := asText(input["source_protocol"]); value != "" && value != bot.SourceProtocol {
		return managementInvalid("机器人协议与当前页面不符。")
	}
	for _, key := range []string{"owner", "subject", "bot", "scope"} {
		if value, exists := input[key]; exists {
			var identity map[string]any
			if decodeObject(value, &identity) != nil {
				return managementInvalid("账号归属格式无效。")
			}
			if err := checkManagementIdentity(identity, bot); err != nil {
				return err
			}
		}
	}
	return nil
}

func managementInput(event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	bot, ok := event.Event.ManagementBot()
	if accountManagementAction(action) && !ok {
		return nil, managementInvalid("请先选择机器人账号。")
	}
	if ok {
		if err := checkManagementIdentity(input, bot); err != nil {
			return nil, err
		}
	}
	if strings.HasPrefix(action, "groups.") && action != "groups.list" || action == "challenge.list" || action == "challenge.clear" || strings.HasPrefix(action, "cards.group.") {
		var scope GroupScope
		if decodeObject(input["scope"], &scope) != nil || scope.GroupID == "" {
			return nil, managementInvalid("请提供群 ID。")
		}
		scope.Protocol, scope.BotID = bot.SourceProtocol, bot.BotID
		input["scope"] = scope
	}
	return input, nil
}
