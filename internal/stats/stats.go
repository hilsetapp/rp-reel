// Package stats считает сводку по событиям сервера.
package stats

import (
	"sort"

	"github.com/hilsetapp/rp-reel/internal/evtlog"
)

// Bucket - одна минута видеоленты.
type Bucket struct {
	Start float64 // секунда от начала видео
	Count int
}

// PlayerCount - сколько раз игрок встретился в событиях.
type PlayerCount struct {
	Name  string
	Count int
}

// Stats - сводка за видео.
type Stats struct {
	Total     int
	ByType    map[string]int
	ByPlayer  map[string]int
	PerMinute []Bucket
	Busiest   Bucket
	Top       []PlayerCount
}

// Compute считает сводку по событиям, попавшим на видео.
func Compute(events []evtlog.Event, videoStart, duration float64) *Stats {
	s := &Stats{ByType: map[string]int{}, ByPlayer: map[string]int{}}

	minutes := 1
	if duration > 0 {
		minutes = int(duration/60) + 1
	}
	perMinute := make([]int, minutes)

	for _, e := range events {
		t := e.TS - videoStart
		if t < 0 || (duration > 0 && t > duration) {
			continue
		}
		s.Total++
		s.ByType[e.Type]++
		if e.Player != "" {
			s.ByPlayer[e.Player]++
		}
		if e.Other != "" {
			s.ByPlayer[e.Other]++
		}
		idx := int(t / 60)
		if idx >= 0 && idx < len(perMinute) {
			perMinute[idx]++
		}
	}

	for i, n := range perMinute {
		s.PerMinute = append(s.PerMinute, Bucket{Start: float64(i * 60), Count: n})
		if n > s.Busiest.Count {
			s.Busiest = Bucket{Start: float64(i * 60), Count: n}
		}
	}

	for name, n := range s.ByPlayer {
		s.Top = append(s.Top, PlayerCount{Name: name, Count: n})
	}
	sort.Slice(s.Top, func(i, j int) bool {
		if s.Top[i].Count != s.Top[j].Count {
			return s.Top[i].Count > s.Top[j].Count
		}
		return s.Top[i].Name < s.Top[j].Name
	})
	if len(s.Top) > 10 {
		s.Top = s.Top[:10]
	}
	return s
}
