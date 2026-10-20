// rp-reel-cli - нарезка RP-видео по журналу событий сервера.
//
//	rp-reel-cli -log rpreel.jsonl -video day.mp4 -video-ts 1729440000
//	rp-reel-cli -log rpreel.jsonl -video day.mp4 -preset discord-digest
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hilsetapp/rp-reel/internal/cut"
	"github.com/hilsetapp/rp-reel/internal/evtlog"
	"github.com/hilsetapp/rp-reel/internal/ffmpeg"
	"github.com/hilsetapp/rp-reel/internal/report"
	"github.com/hilsetapp/rp-reel/internal/stats"
	"github.com/hilsetapp/rp-reel/internal/subtitle"
	"github.com/hilsetapp/rp-reel/internal/template"
	"github.com/hilsetapp/rp-reel/internal/timeline"
)

var version = "dev"

func main() {
	logPath := flag.String("log", "", "журнал событий rpreel.jsonl")
	videoPath := flag.String("video", "", "видеофайл записи")
	videoTS := flag.Float64("video-ts", 0, "время начала записи в секундах Unix")
	offset := flag.Float64("offset", 0, "сдвиг видео относительно журнала, секунд")
	presetName := flag.String("preset", "", "пресет галереи (base, twitch-short, discord-digest, vk-vertical)")
	outDir := flag.String("out", "reel-out", "куда складывать результат")
	noCut := flag.Bool("no-cut", false, "только отчет и план, без ffmpeg")
	showVersion := flag.Bool("version", false, "версия")
	flag.Parse()

	if *showVersion {
		fmt.Printf("rp-reel %s\n", version)
		return
	}
	if *logPath == "" || *videoPath == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "ошибка: нужны оба файла: -log и -video")
		os.Exit(2)
	}

	log, err := evtlog.Load(*logPath)
	if err != nil {
		fail("журнал: %v", err)
	}
	if len(log.Events) == 0 {
		fail("в журнале нет ни одного события")
	}

	preset, err := template.Get(*presetName)
	if err != nil {
		fail("%v", err)
	}

	tools, err := ffmpeg.Find("", "")
	if err != nil {
		fail("%v - установите ffmpeg и добавьте в PATH, либо укажите путь в настройках", err)
	}

	duration, derr := ffmpeg.Duration(tools, *videoPath)
	if derr != nil {
		fail("не удалось прочать видео: %v", derr)
	}

	// синхронизация: время начала записи
	videoStart := log.Events[0].TS - *offset
	if *videoTS > 0 {
		videoStart = *videoTS
	}

	w := timeline.Window{Pre: preset.Pre, Post: preset.Post, MinLen: preset.MinLen, MergeGap: preset.MergeGap}
	clips := timeline.Build(log.Events, videoStart, duration, w)
	if len(clips) == 0 {
		fail("ни одно событие не попало на видео: проверьте синхронизацию (-video-ts или -offset)")
	}
	onVideo := 0
	for _, c := range clips {
		onVideo += len(c.Events)
	}

	if preset.MaxClips > 0 && len(clips) > preset.MaxClips {
		ranked := make([]timeline.Clip, len(clips))
		copy(ranked, clips)
		sort.SliceStable(ranked, func(i, j int) bool {
			return len(ranked[i].Events) > len(ranked[j].Events)
		})
		keep := ranked[:preset.MaxClips]
		sort.SliceStable(keep, func(i, j int) bool { return keep[i].Start < keep[j].Start })
		clips = keep
	}

	st := stats.Compute(log.Events, videoStart, duration)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail("%v", err)
	}

	if err := os.WriteFile(filepath.Join(*outDir, "report.txt"), []byte(report.Text(version, *videoPath, *logPath, preset, duration, log, clips, st)), 0o644); err != nil {
		fail("%v", err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "events.srt"), []byte(subtitle.SRT(clips)), 0o644); err != nil {
		fail("%v", err)
	}
	digest := report.Digest(version, *videoPath, *logPath, preset, videoStart, duration, log, clips, st)
	if err := os.WriteFile(filepath.Join(*outDir, "digest.json"), digest, 0o644); err != nil {
		fail("%v", err)
	}

	fmt.Printf("RP Reel %s\n", version)
	fmt.Printf("видео: %s (%.1f мин)\n", filepath.Base(*videoPath), duration/60)
	fmt.Printf("журнал: %d событий, на видео попало %d\n", len(log.Events), onVideo)
	fmt.Printf("клипов: %d (пресет %q)\n", len(clips), preset.Name)
	if log.Skipped > 0 {
		fmt.Printf("внимание: строк в журнале не разобрано: %d\n", log.Skipped)
	}
	if st.Busiest.Count > 0 {
		fmt.Printf("самая горячая минута: %s (%d событий)\n", clock(st.Busiest.Start), st.Busiest.Count)
	}
	if len(st.Top) > 0 {
		fmt.Printf("топ игроков: %s\n", topLine(st.Top))
	}

	if *noCut {
		fmt.Println("режим -no-cut: клипы не вырезаны, отчет и план готовы")
		return
	}

	jobs := cut.Plan(*videoPath, clips, *outDir, preset)
	for _, j := range jobs {
		fmt.Printf("  вырезаю clip-%03d.mp4  %s-%s  %s\n", j.Index, clock(j.Start), clock(j.End), j.Label)
		if err := ffmpeg.Run(tools.FFmpeg, j.Args...); err != nil {
			fail("клип %d: %v", j.Index, err)
		}
	}

	listPath := filepath.Join(*outDir, "concat.txt")
	if err := os.WriteFile(listPath, []byte(cut.ConcatList(jobs)), 0o644); err != nil {
		fail("%v", err)
	}
	reelPath := filepath.Join(*outDir, "reel.mp4")
	fmt.Println("склеиваю ролик reel.mp4 ...")
	if err := ffmpeg.Run(tools.FFmpeg, cut.ConcatArgs(tools.FFmpeg, listPath, reelPath, preset)...); err != nil {
		fail("склейка: %v", err)
	}
	fmt.Printf("готово: %s\n", reelPath)
}

func clock(t float64) string {
	if t < 0 {
		t = 0
	}
	m := int(t) / 60
	s := int(t) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func topLine(top []stats.PlayerCount) string {
	parts := make([]string, 0, len(top))
	for _, p := range top {
		parts = append(parts, fmt.Sprintf("%s (%d)", p.Name, p.Count))
	}
	return strings.Join(parts, ", ")
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ошибка: "+format+"\n", a...)
	os.Exit(1)
}
