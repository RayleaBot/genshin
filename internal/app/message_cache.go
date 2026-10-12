package app

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func accountMessageKey(prefix string, event *rayleabot.EventContext, messageID string) string {
	raw, _ := json.Marshal([]string{event.Event.SourceProtocol, event.Event.BotID, messageID})
	return prefix + string(raw)
}

// Old message references have a bounded lifetime and no saved bot identity.
// When the old adapter still belongs to this account, move a reference on read
// without extending its expiry. Other old references are left to expire.
func accountMessageCache(ctx context.Context, event *rayleabot.EventContext, prefix, messageID string) (rayleabot.ActionResult, error) {
	key := accountMessageKey(prefix, event, messageID)
	result, err := event.Actions().KVGet(ctx, key)
	if err != nil || result["exists"] == true {
		return result, err
	}
	adapters := []string{event.Event.SourceAdapter}
	for _, bot := range event.Bots {
		if bot.SourceProtocol == event.Event.SourceProtocol && bot.ID == event.Event.BotID && !slices.Contains(adapters, bot.SourceAdapter) {
			adapters = append(adapters, bot.SourceAdapter)
		}
	}
	for _, adapter := range adapters {
		legacy := prefix + adapter + ":" + messageID
		result, err = event.Actions().KVGet(ctx, legacy)
		if err != nil {
			return nil, err
		}
		if result["exists"] != true {
			continue
		}
		expires := int64(number(result["expires_at_ms"]))
		ttl := time.Until(time.UnixMilli(expires)).Truncate(time.Second)
		if ttl < time.Second {
			return result, nil
		}
		if _, err = event.Actions().KVSetWithOptions(ctx, key, result["value"], rayleabot.KVSetOptions{TTL: ttl}); err != nil {
			return nil, err
		}
		_, err = event.Actions().KVDelete(ctx, legacy)
		return result, err
	}
	return result, nil
}
