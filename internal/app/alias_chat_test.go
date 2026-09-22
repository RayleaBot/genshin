package app

import (
	"testing"
)

func TestAliasOwnerPrefersCustomAliases(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[{"id":"10000046","name":"胡桃","aliases":["Hutao"],"kind":"character"},{"id":"10000052","name":"雷电将军","aliases":["雷神"],"kind":"character"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Catalog: catalog}
	custom := map[string]string{"将军": "10000052", "HUTAO": "10000052"}
	if entry, alias, ok := a.aliasOwner("hutao", custom); !ok || entry.ID != "10000052" || alias != "HUTAO" {
		t.Fatal(entry, alias, ok)
	}
	if entry, alias, ok := a.aliasOwner("雷神", custom); !ok || entry.ID != "10000052" || alias != "" {
		t.Fatal(entry, alias, ok)
	}
	if _, _, ok := a.aliasOwner("不存在", custom); ok {
		t.Fatal("found a missing alias")
	}
}
