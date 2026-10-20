package timeline

import (
	"testing"

	"github.com/hilsetapp/rp-reel/internal/evtlog"
)

func ev(ts float64, typ, player, other string) evtlog.Event {
	return evtlog.Event{TS: ts, Type: typ, Player: player, Other: other}
}

func TestWindowsAroundEvents(t *testing.T) {
	w := Default()
	clips := Build([]evtlog.Event{ev(1010, evtlog.Kill, "A", "B")}, 1000, 600, w)
	if len(clips) != 1 {
		t.Fatalf("клипов: %d", len(clips))
	}
	if clips[0].Start != 0 || clips[0].End != 20 {
		t.Fatalf("окно: %.1f..%.1f, хочу 0..20", clips[0].Start, clips[0].End)
	}
	if clips[0].Label() != "убийство: A → B" {
		t.Fatalf("подпись: %q", clips[0].Label())
	}
}

func TestMergeCloseEvents(t *testing.T) {
	w := Default()
	events := []evtlog.Event{
		ev(1010, evtlog.Kill, "A", "B"),
		ev(1015, evtlog.Wreck, "C", ""), // щель 5 сек - сливается в один клип
		ev(1060, evtlog.Arrest, "D", "E"),
	}
	clips := Build(events, 1000, 600, w)
	if len(clips) != 2 {
		t.Fatalf("клипов: %d, хочу 2 (слияние + отдельный)", len(clips))
	}
	if len(clips[0].Events) != 2 {
		t.Fatalf("в первом клипе событий: %d, хочу 2", len(clips[0].Events))
	}
	if clips[0].End != 25 {
		t.Fatalf("конец первого клипа: %.1f, хочу 25", clips[0].End)
	}
}

func TestClampToDuration(t *testing.T) {
	w := Default()
	clips := Build([]evtlog.Event{ev(1595, evtlog.Death, "A", "")}, 1000, 600, w)
	if len(clips) != 1 {
		t.Fatalf("клипов: %d", len(clips))
	}
	if clips[0].End != 600 {
		t.Fatalf("конец: %.1f, хочу обрезку на 600", clips[0].End)
	}
}

func TestSkipOutside(t *testing.T) {
	w := Default()
	clips := Build([]evtlog.Event{
		ev(900, evtlog.Kill, "A", "B"),  // до начала видео
		ev(2000, evtlog.Kill, "C", "D"), // после конца видео
		ev(1300, evtlog.Kill, "E", "F"), // внутри
	}, 1000, 600, w)
	if len(clips) != 1 {
		t.Fatalf("клипов: %d, хочу 1", len(clips))
	}
	if clips[0].Events[0].Player != "E" {
		t.Fatalf("взято не то событие: %+v", clips[0].Events[0])
	}
}

func TestMinLenStretch(t *testing.T) {
	w := Default()
	clips := Build([]evtlog.Event{ev(1002, evtlog.Join, "A", "")}, 1000, 600, w)
	if len(clips) != 1 {
		t.Fatal("клип потерялся")
	}
	if clips[0].Len() < w.MinLen-1e-9 {
		t.Fatalf("длина %.2f меньше минимума %.2f", clips[0].Len(), w.MinLen)
	}
	if clips[0].Start != 0 {
		t.Fatalf("начало %.2f, хочу 0 (у видео нет отрицательного времени)", clips[0].Start)
	}
}

func TestEmpty(t *testing.T) {
	if clips := Build(nil, 0, 0, Default()); clips != nil {
		t.Fatalf("клипов без событий: %d", len(clips))
	}
}
