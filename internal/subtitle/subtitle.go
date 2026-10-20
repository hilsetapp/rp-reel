// Package subtitle делает SRT с подписями событий.
package subtitle

import (
	"fmt"
	"strings"

	"github.com/hilsetapp/rp-reel/internal/timeline"
)

// SRT собирает субтитры: каждый клип - одна подпись с событиями внутри.
func SRT(clips []timeline.Clip) string {
	var b strings.Builder
	for i, c := range clips {
		fmt.Fprintf(&b, "%d\n%s --> %s\n", i+1, stamp(c.Start), stamp(c.End))
		if len(c.Events) == 1 {
			b.WriteString(c.Events[0].Caption())
		} else {
			b.WriteString(c.Label())
			for _, e := range c.Events {
				fmt.Fprintf(&b, "\n- %s", e.Caption())
			}
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

func stamp(t float64) string {
	if t < 0 {
		t = 0
	}
	h := int(t) / 3600
	m := (int(t) % 3600) / 60
	s := int(t) % 60
	ms := int((t - float64(int(t))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
