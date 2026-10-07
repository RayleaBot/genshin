package images

import (
	"testing"

	"github.com/RayleaBot/genshin/internal/app"
)

func TestDamageMatrixFormatsTradesLikeMiao(t *testing.T) {
	critical := 92562.8
	matrix := app.DamageMatrix{Index: 3, Title: "霜华矢", Avg: 72632.2, Dmg: &critical,
		Attrs: []app.DamageAttr{{Title: "大攻击", Text: "5.83%"}, {Title: "暴击率", Text: "3.88%"}},
		Rows:  [][]app.DamageCell{{{Type: "na"}, {Type: "gt", Avg: 73232.9, Dmg: &critical}}, {{Type: "lt", Avg: 71912.5}, {Type: "na"}}}}
	data := damageMatrixData(matrix)
	rows := data["rows"].([]any)
	gain := rows[0].(map[string]any)["cells"].([]any)[1].(map[string]any)
	loss := rows[1].(map[string]any)["cells"].([]any)[0].(map[string]any)
	if data["avg"] != "72,632" || gain["val"] != "+601" || gain["avg"] != "73,233" || gain["dmg"] != "92,563" || loss["val"] != "-720" || loss["dmg"] != "" {
		t.Fatalf("data = %v", data)
	}
}
