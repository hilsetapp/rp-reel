// Package timeline превращает события в окна нарезки на видеоленте.
package timeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hilsetapp/rp-reel/internal/evtlog"
)

// Window - настройки окна вокруг события, в секундах.
type Window struct {
	Pre      float64 // захватить до события
	Post     float64 // захватить после события
	MinLen   float64 // минимальная длина окна
	MergeGap float64 // щель меньше этой - окна сливаются в один клип
}

// Default - разумные значения для игровых моментов.
func Default() Window {
	return Window{Pre: 10, Post: 10, MinLen: 3, MergeGap: 2}
}

// Clip - одно окно на видео: от Start до End секунды видеоленты.
type Clip struct {
	Start  float64
	End    float64
	Events []evtlog.Event
}

// Len - длина клипа в секундах.
func (c Clip) Len() float64 { return c.End - c.Start }

// Label - короткая подпись клипа для списка и субтитров.
func (c Clip) Label() string {
	if len(c.Events) == 1 {
		return c.Events[0].Caption()
	}
	seen := map[string]bool{}
	var parts []string
	for _, e := range c.Events {
		name := evtlog.TypeName(e.Type)
		if !seen[name] {
			seen[name] = true
			parts = append(parts, name)
		}
	}
	return fmt.Sprintf("%d событий: %s", len(c.Events), strings.Join(parts, ", "))
}

// Build раскладывает события по клипам.
//
// videoStart - время начала записи в секундах Unix, duration - длина видео
// в секундах (0 если неизвестна, тогда конец не обрезается). События,
// которых нет на видео, пропускаются.
func Build(events []evtlog.Event, videoStart, duration float64, w Window) []Clip {
	type raw struct {
		start, end float64
		ev         evtlog.Event
	}
	var raws []raw
	for _, e := range events {
		t := e.TS - videoStart
		if t < -w.Pre || (duration > 0 && t > duration+w.Post) {
			continue
		}
		start, end := t-w.Pre, t+w.Post
		if start < 0 {
			start = 0
		}
		if duration > 0 && end > duration {
			end = duration
		}
		if end-start < w.MinLen {
			// растянуть окно до минимума: сначала вправо, потом влево
			if duration <= 0 || end+w.MinLen-(end-start) <= duration {
				end = start + w.MinLen
			} else {
				start = end - w.MinLen
				if start < 0 {
					start = 0
				}
			}
		}
		raws = append(raws, raw{start: start, end: end, ev: e})
	}
	if len(raws) == 0 {
		return nil
	}

	// сортировка по началу окна, дальше по времени события
	sort.SliceStable(raws, func(i, j int) bool {
		if raws[i].start != raws[j].start {
			return raws[i].start < raws[j].start
		}
		return raws[i].ev.TS < raws[j].ev.TS
	})

	var clips []Clip
	cur := Clip{Start: raws[0].start, End: raws[0].end, Events: []evtlog.Event{raws[0].ev}}
	for _, r := range raws[1:] {
		if r.start <= cur.End+w.MergeGap {
			if r.end > cur.End {
				cur.End = r.end
			}
			cur.Events = append(cur.Events, r.ev)
			continue
		}
		clips = append(clips, cur)
		cur = Clip{Start: r.start, End: r.end, Events: []evtlog.Event{r.ev}}
	}
	return append(clips, cur)
}
