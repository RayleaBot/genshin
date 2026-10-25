package app

import "testing"

func TestPoolHistoryUsesMiaoDescendingOrder(t *testing.T) {
	pools := []PoolInfo{
		{Version: "7.1", Half: "上半", Kind: "event"},
		{Version: "6.8", Half: "下半", Kind: "event"},
		{Version: "7.1", Half: "下半", Kind: "event"},
		{Version: "7.1", Half: "上半", Kind: "chronicled"},
	}
	sortPoolsDesc(pools)
	if pools[0].Version != "7.1" || pools[0].Half != "下半" || pools[1].Kind != "event" || pools[2].Kind != "chronicled" || pools[3].Version != "6.8" {
		t.Fatalf("pool history order = %+v", pools)
	}
}
