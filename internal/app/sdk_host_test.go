package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// sdkHost runs the plugin through the SDK runtime as the host does: it writes
// the host's frames, answers the actions the plugin asks for and sends the
// scheduler triggers of the jobs the plugin created in the host's shape.
type sdkHost struct {
	t      *testing.T
	writer *io.PipeWriter
	lines  chan []byte
	next   int
	// service answers plugin.call: a result, or a failure code. scheduled
	// marks a call of a scheduler trigger.
	service func(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string)
	// jobs are the scheduler.create requests of the jobs by task ID; deleted
	// are the task IDs scheduler.delete removed.
	jobs    map[string]rayleabot.SchedulerCreateRequest
	deleted []string
	// sent are the message.send actions, not counting terminal replies.
	sent []rayleabot.MessageSendRequest
	// triggers are the request IDs of scheduler triggers.
	triggers map[string]bool
	// busy refuses event.detach as the host does while the plugin holds its
	// limit of background events.
	busy bool
	// deadline and background are how long an event, and an event moved to
	// the background, may take: the host's defaults when zero. due is when
	// the last event sent is due.
	deadline, background time.Duration
	due                  time.Time
}

// hostAction is an action the plugin asked for during one event.
type hostAction struct {
	Name string
	Data map[string]any
}

// detached is the event.detach an event asked for, if any.
func detached(actions []hostAction) (hostAction, bool) {
	for _, action := range actions {
		if action.Name == "event.detach" {
			return action, true
		}
	}
	return hostAction{}, false
}

// newSDKHost starts the plugin with the host's init: bot "bot" on adapter
// "a", prefix "#", the plugin config and four concurrent events.
func newSDKHost(t *testing.T, a *App, config map[string]any, service func(rayleabot.ServiceCallRequest, bool) (map[string]any, string)) *sdkHost {
	t.Helper()
	if config == nil {
		config = map[string]any{}
	}
	inReader, hostWriter := io.Pipe()
	hostReader, outWriter := io.Pipe()
	finished := make(chan error, 1)
	go func() {
		finished <- rayleabot.Run(context.Background(), rayleabot.Options{Stdin: inReader, Stdout: outWriter, Stderr: io.Discard}, a)
		outWriter.Close()
	}()
	h := &sdkHost{t: t, writer: hostWriter, lines: make(chan []byte, 64), service: service, jobs: map[string]rayleabot.SchedulerCreateRequest{}, triggers: map[string]bool{}}
	t.Cleanup(func() {
		hostWriter.Close()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("SDK did not stop")
		}
		hostReader.Close()
	})
	go func() {
		scanner := bufio.NewScanner(hostReader)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			h.lines <- bytes.Clone(scanner.Bytes())
		}
		close(h.lines)
	}()
	h.exchange("init", map[string]any{"type": "init", "request_id": "init", "protocol_version": rayleabot.ProtocolVersion, "plugin_id": "raylea.genshin", "bots": []any{map[string]any{"source_adapter": "a", "source_protocol": "onebot11", "id": "bot"}}, "config": config, "super_admins": []string{}, "command_prefixes": []string{"#"}, "timezone": "Asia/Shanghai", "concurrency": 4})
	return h
}

func (h *sdkHost) write(frame map[string]any) {
	h.t.Helper()
	raw, err := json.Marshal(frame)
	if err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.writer.Write(append(raw, '\n')); err != nil {
		h.t.Fatal(err)
	}
}

// exchange sends a frame and answers the plugin's actions until it ends the
// request, which an event moved to the background with event.detach ends
// later; it returns the ending frame and the actions asked for.
func (h *sdkHost) exchange(id string, frame map[string]any) (map[string]any, []hostAction) {
	h.t.Helper()
	h.write(frame)
	var actions []hostAction
	for {
		var raw []byte
		select {
		case raw = <-h.lines:
			if raw == nil {
				h.t.Fatal("SDK closed unexpectedly")
			}
		case <-time.After(5 * time.Second):
			h.t.Fatal("SDK response timed out")
		}
		var response map[string]any
		if json.Unmarshal(raw, &response) != nil {
			h.t.Fatal("invalid SDK frame")
		}
		if response["request_id"] == id {
			return response, actions
		}
		if response["type"] != "action" {
			continue
		}
		data := asObject(response["data"])
		actions = append(actions, hostAction{Name: asText(response["action"]), Data: data})
		result, failure := h.answer(asText(response["action"]), data, h.triggers[asText(response["parent_request_id"])])
		if failure != "" {
			h.write(map[string]any{"type": "error", "request_id": response["request_id"], "code": failure, "message": failure})
			continue
		}
		h.write(map[string]any{"type": "result", "request_id": response["request_id"], "status": "success", "data": result})
	}
}

// answer carries out an action as the host would.
func (h *sdkHost) answer(action string, data map[string]any, scheduled bool) (map[string]any, string) {
	switch action {
	case "plugin.call":
		var request rayleabot.ServiceCallRequest
		if decodeObject(data, &request) != nil {
			h.t.Fatal("invalid plugin.call")
		}
		return h.service(request, scheduled)
	case "scheduler.create":
		var request rayleabot.SchedulerCreateRequest
		if decodeObject(data, &request) != nil || request.TaskID == "" || request.Cron == "" {
			h.t.Fatal("invalid scheduler.create")
		}
		request.Payload = maps.Clone(request.Payload)
		h.jobs[request.TaskID] = request
		return map[string]any{"task_id": request.TaskID}, ""
	case "scheduler.delete":
		id := asText(data["task_id"])
		if id == "" {
			// The host stops a plugin that deletes a job without an ID.
			h.t.Error("scheduler.delete without a task ID")
		}
		_, exists := h.jobs[id]
		delete(h.jobs, id)
		h.deleted = append(h.deleted, id)
		return map[string]any{"task_id": id, "deleted": exists}, ""
	case "message.send":
		var request rayleabot.MessageSendRequest
		if decodeObject(data, &request) != nil {
			h.t.Fatal("invalid message.send")
		}
		h.sent = append(h.sent, request)
		return map[string]any{"message_id": fmt.Sprintf("sent-%d", len(h.sent)), "delivery_kind": "send"}, ""
	case "event.detach":
		if h.busy {
			return nil, "platform.rate_limited"
		}
		// The host's default background deadline is 900 seconds.
		background := h.background
		if background == 0 {
			background = 15 * time.Minute
		}
		return map[string]any{"deadline_at_ms": time.Now().Add(background).UnixMilli()}, ""
	case "render.image":
		return nil, "platform.render_unavailable"
	case "logger.write":
		return map[string]any{}, ""
	}
	h.t.Fatalf("unexpected action %s", action)
	return nil, ""
}

// chat sends a message of user "u" in target, with the host's parsed
// command, if any.
func (h *sdkHost) chat(target map[string]any, role, text, command string, args ...string) (map[string]any, []hostAction) {
	h.t.Helper()
	h.next++
	id := fmt.Sprintf("chat-%d", h.next)
	eventType := "message.private"
	if target["type"] == "group" {
		eventType = "message.group"
	}
	event := map[string]any{"event_id": id, "event_type": eventType, "source_protocol": "onebot11", "source_adapter": "a", "timestamp": time.Now().Unix(), "actor": map[string]any{"id": "u", "role": role}, "target": target, "message": map[string]any{"plain_text": text, "segments": []any{map[string]any{"type": "text", "data": map[string]any{"text": text}}}}}
	if command != "" {
		event["payload"] = map[string]any{"command": command, "args": args}
	}
	return h.exchange(id, h.frame(id, event))
}

// message sends a private chat message of user "u".
func (h *sdkHost) message(text, command string, args ...string) (map[string]any, []hostAction) {
	h.t.Helper()
	return h.chat(map[string]any{"type": "private", "id": "u"}, "", text, command, args...)
}

// groupMessage sends a message of user "u", an administrator of group "g".
func (h *sdkHost) groupMessage(text, command string, args ...string) (map[string]any, []hostAction) {
	h.t.Helper()
	return h.chat(map[string]any{"type": "group", "id": "g"}, "admin", text, command, args...)
}

// manage runs an action of the plugin's management page.
func (h *sdkHost) manage(action string, input map[string]any) (map[string]any, []hostAction) {
	h.t.Helper()
	h.next++
	id := fmt.Sprintf("manage-%d", h.next)
	return h.exchange(id, h.frame(id, map[string]any{"event_id": id, "event_type": "management.action", "source_protocol": "management", "source_adapter": "management.ui", "timestamp": time.Now().Unix(), "payload": map[string]any{"action": action, "payload": input}}))
}

// trigger runs a job the plugin created as the host's scheduler does: the
// event has no target, and its payload holds the task ID, and the job's
// payload and its action when the job has one.
func (h *sdkHost) trigger(taskID string) (map[string]any, []hostAction) {
	h.t.Helper()
	job, exists := h.jobs[taskID]
	if !exists {
		h.t.Fatalf("no job %s", taskID)
	}
	h.next++
	id := fmt.Sprintf("scheduler-%d", h.next)
	h.triggers[id] = true
	payload := map[string]any{"task_id": taskID}
	if len(job.Payload) > 0 {
		payload["payload"] = maps.Clone(job.Payload)
	}
	if action, ok := job.Payload["action"].(string); ok && action != "" {
		payload["action"] = action
	}
	return h.exchange(id, h.frame(id, map[string]any{"event_id": "scheduler-" + taskID + "-" + id, "event_type": "scheduler.trigger", "source_protocol": "scheduler", "source_adapter": "scheduler.internal", "timestamp": time.Now().Unix(), "payload": payload}))
}

// frame is the host's frame of event id, due by the event deadline.
func (h *sdkHost) frame(id string, event map[string]any) map[string]any {
	deadline := h.deadline
	if deadline == 0 {
		deadline = time.Minute
	}
	h.due = time.Now().Add(deadline)
	return map[string]any{"type": "event", "request_id": id, "deadline_at_ms": h.due.UnixMilli(), "event": event}
}

// triggerUntilDone runs the job each minute after start, as the host's
// scheduler does, until the plugin deletes it.
func triggerUntilDone(t *testing.T, clock *fakeClock, host *sdkHost, ref string, start time.Time, limit int) {
	t.Helper()
	for minute := 1; len(host.deleted) == 0; minute++ {
		if minute > limit {
			t.Fatalf("%d triggers of %s did not finish it", limit, ref)
		}
		clock.set(start.Add(time.Duration(minute) * time.Minute))
		end, _ := host.trigger(ref)
		if end["type"] != "result" {
			t.Fatalf("trigger %d ended with %v", minute, end)
		}
	}
}

// job is the only job of the plugin whose task ID starts with prefix.
func (h *sdkHost) job(prefix string) string {
	h.t.Helper()
	found := []string{}
	for id := range h.jobs {
		if strings.HasPrefix(id, prefix) {
			found = append(found, id)
		}
	}
	if len(found) != 1 {
		h.t.Fatalf("jobs %v", found)
	}
	return found[0]
}

// terminalText is the text of a message an event ended with.
func terminalText(frame map[string]any) string {
	var message rayleabot.MessageOut
	_ = decodeObject(asObject(frame["data"])["message"], &message)
	return sentText(message)
}

// sentText is the text of a message.
func sentText(message rayleabot.MessageOut) string {
	var text string
	for _, segment := range message.Segments {
		text += asText(segment.Data["text"])
	}
	return text
}
