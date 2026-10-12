package app

import (
	"encoding/json"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func accountMessageKey(prefix string, event *rayleabot.EventContext, messageID string) string {
	raw, _ := json.Marshal([]string{event.Event.SourceProtocol, event.Event.BotID, messageID})
	return prefix + string(raw)
}
