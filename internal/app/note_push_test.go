package app

import "testing"

func TestNotePushGroupSettingsFollowXiaoyao(t *testing.T) {
	var p NotePush
	if p.threshold() != 120 {
		t.Fatal(p.threshold())
	}
	for _, tc := range []struct {
		args []string
		want NotePush
	}{
		{[]string{"关闭"}, NotePush{Off: true}},
		{[]string{"200"}, NotePush{Off: true, Resin: 160}},
		{[]string{"5"}, NotePush{Off: true, Resin: 20}},
		{nil, NotePush{Off: true, Resin: 120}},
		{[]string{"开启"}, NotePush{Resin: 120}},
		{[]string{"0"}, NotePush{Resin: 120}},
	} {
		p.set(tc.args)
		if p != tc.want {
			t.Fatalf("%v: %+v", tc.args, p)
		}
	}
}

func TestNotePushGoesToTheFirstGroupReached(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	scope := func(id string) GroupScope {
		return GroupScope{Protocol: "onebot11", Adapter: "napcat", BotID: "bot", GroupID: id}
	}
	off, high, low := scope("off"), scope("high"), scope("low")
	disabled := false
	for group, fn := range map[GroupScope]func(*GroupData){
		off:  func(g *GroupData) { g.NotePush.Off = true },
		high: func(g *GroupData) { g.NotePush.Resin = 160 },
		low:  func(g *GroupData) { g.NotePush.Resin = 40 },
	} {
		if err := s.Update(group, func(g *GroupData) error { fn(g); return nil }); err != nil {
			t.Fatal(err)
		}
	}
	groups := []GroupScope{off, high, low}
	// A group that turned its push off is passed over, as is one whose
	// threshold the resin has not reached.
	if got, ok := s.pushGroup(groups, 150); !ok || got != low {
		t.Fatal(got, ok)
	}
	if got, ok := s.pushGroup(groups, 160); !ok || got != high {
		t.Fatal(got, ok)
	}
	if _, ok := s.pushGroup(groups, 30); ok {
		t.Fatal("pushed below every threshold")
	}
	// A group that turned the plugin off gets no push.
	if err := s.Update(low, func(g *GroupData) error { g.Config.Enabled = &disabled; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.pushGroup(groups, 150); ok {
		t.Fatal("pushed to a group with the plugin off")
	}
}
