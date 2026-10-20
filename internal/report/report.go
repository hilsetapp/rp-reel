// Package report делает текстовый отчет и JSON-дайджест.
package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hilsetapp/rp-reel/internal/evtlog"
	"github.com/hilsetapp/rp-reel/internal/stats"
	"github.com/hilsetapp/rp-reel/internal/template"
	"github.com/hilsetapp/rp-reel/internal/timeline"
)

// Text - отчет для report.txt, по-русски.
func Text(version, video, logPath string, p template.Preset, duration float64, log *evtlog.Log, clips []timeline.Clip, st *stats.Stats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Отчет RP Reel %s\n\n", version)
	fmt.Fprintf(&b, "Видео: %s (%.1f мин)\n", video, duration/60)
	fmt.Fprintf(&b, "Журнал: %s\n", logPath)
	fmt.Fprintf(&b, "Событий в журнале: %d, на видео: %d\n", len(log.Events), st.Total)
	if log.Skipped > 0 {
		fmt.Fprintf(&b, "Строк не разобрано: %d\n", log.Skipped)
		for _, e := range log.Errors {
			fmt.Fprintf(&b, "  %s\n", e)
		}
	}
	fmt.Fprintf(&b, "Пресет: %s - %s\n", p.Name, p.Title)
	fmt.Fprintf(&b, "Клипов: %d\n\n", len(clips))

	fmt.Fprintln(&b, "Типы событий:")
	for _, name := range sortedKeys(st.ByType) {
		fmt.Fprintf(&b, "  %s: %d\n", evtlog.TypeName(name), st.ByType[name])
	}
	fmt.Fprintln(&b)

	if st.Busiest.Count > 0 {
		fmt.Fprintf(&b, "Самая горячая минута: %s (%d событий)\n\n", clock(st.Busiest.Start), st.Busiest.Count)
	}

	fmt.Fprintln(&b, "Топ игроков:")
	for _, p := range st.Top {
		fmt.Fprintf(&b, "  %s: %d\n", p.Name, p.Count)
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "Клипы:")
	for i, c := range clips {
		fmt.Fprintf(&b, "  %02d. %s-%s  %s\n", i+1, clock(c.Start), clock(c.End), c.Label())
	}
	return b.String()
}

// Digest - структурированная сводка для digest.json.
func Digest(version, video, logPath string, p template.Preset, videoStart, duration float64, log *evtlog.Log, clips []timeline.Clip, st *stats.Stats) []byte {
	d := DigestOut{
		Version:    version,
		Video:      video,
		Log:        logPath,
		Preset:     p.Name,
		VideoStart: videoStart,
		Duration:   duration,
		Total:      len(log.Events),
		OnVideo:    st.Total,
		Skipped:    log.Skipped,
		Stats:      st,
	}
	for _, c := range clips {
		d.Clips = append(d.Clips, DigestClip{
			Start: c.Start, End: c.End, Label: c.Label(), Events: c.Events,
		})
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return []byte("{}")
	}
	return data
}

type DigestOut struct {
	Version    string       `json:"version"`
	Video      string       `json:"video"`
	Log        string       `json:"log"`
	Preset     string       `json:"preset"`
	VideoStart float64      `json:"video_start"`
	Duration   float64      `json:"duration"`
	Total      int          `json:"total_events"`
	OnVideo    int          `json:"on_video"`
	Skipped    int          `json:"skipped_lines"`
	Clips      []DigestClip `json:"clips"`
	Stats      *stats.Stats `json:"stats"`
}

type DigestClip struct {
	Start  float64        `json:"start"`
	End    float64        `json:"end"`
	Label  string         `json:"label"`
	Events []evtlog.Event `json:"events"`
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func clock(t float64) string {
	if t < 0 {
		t = 0
	}
	m := int(t) / 60
	sec := int(t) % 60
	return fmt.Sprintf("%02d:%02d", m, sec)
}

var _ = strconv.Itoa
