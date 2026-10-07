package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/genshin/internal/artwork"
)

func TestArtworkStatusDoesNotSendLocalErrors(t *testing.T) {
	var payload bytes.Buffer
	compressed := gzip.NewWriter(&payload)
	archive := tar.NewWriter(compressed)
	if err := archive.WriteHeader(&tar.Header{Name: "fixture-main/assets/fixture.png", Mode: 0o600, Size: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write([]byte("png")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload.Bytes()) }))
	defer server.Close()

	blocked := filepath.Join(t.TempDir(), "private-fixture-root")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := pluginApp(t)
	a.Artwork = &artwork.Store{Root: blocked, Sources: []artwork.Source{{
		ID: "fixture", Name: "fixture", Archives: []string{server.URL}, Include: []string{"assets/"}, Extensions: []string{".png"},
	}}}
	if _, err := a.Artwork.Start("fixture"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for {
		status := a.Artwork.Statuses()[0]
		if status.Error != "" {
			if !strings.Contains(status.Error, blocked) {
				t.Fatalf("expected filesystem diagnostic, got %q", status.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("download did not fail")
		}
		time.Sleep(5 * time.Millisecond)
	}

	host := newSDKHost(t, a, nil, nil)
	terminal, _ := host.groupMessage("素材状态", "素材状态")
	text := terminalText(terminal)
	if terminal["action"] != "message.send" || !strings.Contains(text, "fixture") || !strings.Contains(text, "失败") {
		t.Fatalf("missing source failure notice: %v", terminal)
	}
	for _, private := range []string{blocked, "private-fixture-root", server.URL} {
		if strings.Contains(text, private) {
			t.Fatalf("chat leaked a diagnostic: %q", text)
		}
	}
	if !strings.Contains(a.Artwork.Statuses()[0].Error, blocked) {
		t.Fatal("management diagnostics were lost")
	}
}

const privateCloudDiagnostic = "open /srv/private/cache.json: permission denied; cookie=fixture-only-cookie"

func TestArkRepliesDoNotForwardUpstreamDiagnostics(t *testing.T) {
	for _, code := range []any{401, "403", 429, 999, "cookie=fixture-only-cookie"} {
		result := map[string]any{"retcode": code, "message": privateCloudDiagnostic}
		_, _, panelFailure := customRankAvatar(result, "fixture")
		for _, text := range []string{panelFailure, arkUsageText(result), arkUsageText(privateCloudDiagnostic)} {
			if text == "" {
				t.Fatal("failure notice is empty")
			}
			for _, private := range []string{"/srv/private", "permission denied", "fixture-only-cookie", "cookie="} {
				if strings.Contains(text, private) {
					t.Fatalf("chat leaked upstream diagnostics: %q", text)
				}
			}
		}
	}
	for _, code := range []int{401, 403, 429} {
		result := map[string]any{"retcode": code, "message": "does not exist"}
		if arkQueryFailure(result) == arkQueryFailure(nil) {
			t.Fatalf("known error code %d lost its actionable notice", code)
		}
		result["message"] = privateCloudDiagnostic
		if arkQueryFailure(result) != arkQueryFailure(map[string]any{"retcode": code}) {
			t.Fatal("upstream message changed the public failure")
		}
	}
}

func TestArkFailureCommandsSendOnlyLocalMessages(t *testing.T) {
	a := pluginApp(t)
	calls := 0
	service := func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		if request.Method != "ark.request" {
			t.Fatalf("unexpected service method: %s", request.Method)
		}
		calls++
		return map[string]any{"data": map[string]any{"retcode": 999, "message": privateCloudDiagnostic}}, ""
	}
	host := newSDKHost(t, a, nil, service)
	for _, command := range []string{"ark胡桃排行", "arktoken用量"} {
		terminal, _ := host.groupMessage(command, command)
		text := terminalText(terminal)
		if terminal["action"] != "message.send" || text == "" {
			t.Fatalf("missing failure reply: %v", terminal)
		}
		if strings.Contains(text, "/srv/private") || strings.Contains(text, "fixture-only-cookie") {
			t.Fatalf("command leaked upstream diagnostics: %q", text)
		}
	}
	if calls != 2 {
		t.Fatalf("upstream calls = %d, want 2", calls)
	}
}
