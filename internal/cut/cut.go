// Package cut строит команды ffmpeg для вырезки клипов.
package cut

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/hilsetapp/rp-reel/internal/template"
	"github.com/hilsetapp/rp-reel/internal/timeline"
)

// Job - одно задание на вырезку.
type Job struct {
	Index int
	Start float64
	End   float64
	Label string
	Path  string // куда писать клип
	Args  []string
}

// Plan готовит задания для всех клипов.
func Plan(video string, clips []timeline.Clip, outDir string, p template.Preset) []Job {
	jobs := make([]Job, 0, len(clips))
	for i, c := range clips {
		path := filepath.Join(outDir, fmt.Sprintf("clip-%03d.mp4", i+1))
		args := []string{"-y"}
		args = append(args, "-ss", strconv.FormatFloat(c.Start, 'f', 3, 64))
		args = append(args, "-i", video)
		args = append(args, "-t", strconv.FormatFloat(c.Len(), 'f', 3, 64))
		if p.Reencode {
			args = append(args, p.ExtraArgs...)
		} else {
			args = append(args, "-c", "copy", "-movflags", "+faststart")
		}
		args = append(args, path)
		jobs = append(jobs, Job{
			Index: i + 1, Start: c.Start, End: c.End, Label: c.Label(), Path: path, Args: args,
		})
	}
	return jobs
}

// ConcatList - содержимое файла списка для склейки (concat demuxer).
func ConcatList(jobs []Job) string {
	var b []byte
	for _, j := range jobs {
		b = append(b, []byte(fmt.Sprintf("file '%s'\n", escape(j.Path)))...)
	}
	return string(b)
}

func escape(p string) string {
	r := ""
	for _, r0 := range p {
		if r0 == '\'' {
			r += "'\\''"
		} else {
			r += string(r0)
		}
	}
	return r
}

// ConcatArgs - команда склейки клипов в один ролик.
func ConcatArgs(ffmpeg, listPath, reelPath string, p template.Preset) []string {
	args := []string{"-y", "-f", "concat", "-safe", "0", "-i", listPath}
	if p.Reencode {
		args = append(args, p.ExtraArgs...)
	} else {
		args = append(args, "-c", "copy", "-movflags", "+faststart")
	}
	return append(args, reelPath)
}
