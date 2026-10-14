package app

import (
	"strings"
	"testing"
)

func TestCloudGameTaskLineShowsState(t *testing.T) {
	task := Reminder{Kind: "cloudgame", Once: true, Hour: 8, LastCode: "plugin.upstream_cloud_unavailable"}
	if line := cloudGameTaskLine(task, "x"); !strings.Contains(line, "一次") || !strings.Contains(line, "上次未完成") || !strings.Contains(line, "已停止") {
		t.Error(line)
	}
}
