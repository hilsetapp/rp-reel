// Package ffmpeg находит и запускает ffmpeg/ffprobe.
package ffmpeg

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ErrMissing - ffmpeg или ffprobe не найдены в PATH.
var ErrMissing = errors.New("ffmpeg не найден в PATH")

// Tools - пути к найденным инструментам.
type Tools struct {
	FFmpeg  string
	FFprobe string
}

// Find ищет ffmpeg и ffprobe. Явный путь (если задан) проверяется первым.
func Find(ffmpegPath, ffprobePath string) (Tools, error) {
	t := Tools{
		FFmpeg:  lookup(ffmpegPath, "ffmpeg"),
		FFprobe: lookup(ffprobePath, "ffprobe"),
	}
	if t.FFmpeg == "" || t.FFprobe == "" {
		return t, ErrMissing
	}
	return t, nil
}

func lookup(explicit, name string) string {
	if explicit != "" {
		if _, err := exec.LookPath(explicit); err == nil {
			return explicit
		}
		return ""
	}
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

// Duration - длина видео в секундах.
func Duration(t Tools, video string) (float64, error) {
	out, err := exec.Command(t.FFprobe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		video,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe вернул %q", strings.TrimSpace(string(out)))
	}
	return d, nil
}

// Thumb вытаскивает один кадр в момент t и кладет его в out (PNG).
func Thumb(t Tools, video string, at float64, out string) error {
	_, err := exec.Command(t.FFmpeg,
		"-y", "-ss", strconv.FormatFloat(at, 'f', 3, 64),
		"-i", video, "-frames:v", "1", "-q:v", "3", out,
	).CombinedOutput()
	return err
}

// Run выполняет команду ffmpeg и возвращает понятную ошибку с хвостом вывода.
func Run(tools string, args ...string) error {
	out, err := exec.Command(tools, args...).CombinedOutput()
	if err != nil {
		tail := string(out)
		if len(tail) > 400 {
			tail = tail[len(tail)-400:]
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(tail))
	}
	return nil
}
