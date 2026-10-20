package stats

import (
	"testing"

	"github.com/hilsetapp/rp-reel/internal/evtlog"
)

func TestCompute(t *testing.T) {
	events := []evtlog.Event{
		{TS: 1000, Type: evtlog.Kill, Player: "A", Other: "B"},
		{TS: 1050, Type: evtlog.Kill, Player: "A", Other: "C"}, // та же минута
		{TS: 1120, Type: evtlog.Wreck, Player: "B"},
		{TS: 1500, Type: evtlog.Kill, Player: "D", Other: "A"}, // минута 8
		{TS: 900, Type: evtlog.Kill, Player: "X", Other: "Y"},  // до видео
	}
	s := Compute(events, 1000, 900)

	if s.Total != 4 {
		t.Fatalf("событий: %d, хочу 4", s.Total)
	}
	if s.ByType[evtlog.Kill] != 3 || s.ByType[evtlog.Wreck] != 1 {
		t.Fatalf("по типам: %v", s.ByType)
	}
	if s.ByPlayer["A"] != 3 { // дважды убийца и один раз жертва
		t.Fatalf("игрок A в событиях: %d, хочу 3", s.ByPlayer["A"])
	}
	if len(s.PerMinute) != 16 {
		t.Fatalf("минут: %d, хочу 16", len(s.PerMinute))
	}
	if s.PerMinute[0].Count != 2 {
		t.Fatalf("первая минута: %d событий, хочу 2", s.PerMinute[0].Count)
	}
	if s.Busiest.Count != 2 || s.Busiest.Start != 0 {
		t.Fatalf("горячая минута: %+v, хочу start=0 count=2", s.Busiest)
	}
	if s.Top[0].Name != "A" || s.Top[0].Count != 3 {
		t.Fatalf("топ: %+v", s.Top[0])
	}
}

func TestEmpty(t *testing.T) {
	s := Compute(nil, 0, 0)
	if s.Total != 0 || len(s.PerMinute) != 1 || len(s.Top) != 0 {
		t.Fatalf("пустая сводка не пустая: %+v", s)
	}
}
