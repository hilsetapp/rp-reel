package subtitle

import (
	"strings"
	"testing"

	"github.com/hilsetapp/rp-reel/internal/evtlog"
	"github.com/hilsetapp/rp-reel/internal/timeline"
)

func TestSRT(t *testing.T) {
	clips := []timeline.Clip{
		{Start: 10, End: 30, Events: []evtlog.Event{{Type: evtlog.Kill, Player: "A", Other: "B"}}},
		{Start: 61.5, End: 80, Events: []evtlog.Event{
			{Type: evtlog.Kill, Player: "C", Other: "D"},
			{Type: evtlog.Wreck, Player: "E"},
		}},
	}
	out := SRT(clips)
	if !strings.Contains(out, "00:00:10,000 --> 00:00:30,000") {
		t.Fatalf("первый таймкод: %q", out)
	}
	if !strings.Contains(out, "00:01:01,500 --> 00:01:20,000") {
		t.Fatalf("второй таймкод: %q", out)
	}
	if !strings.Contains(out, "убийство: A → B") {
		t.Fatalf("подпись: %q", out)
	}
	if !strings.Contains(out, "2 событий") {
		t.Fatalf("слитный клип без счета: %q", out)
	}
}
