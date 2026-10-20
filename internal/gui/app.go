// GUI RP Reel: таймлайн событий RP-сервера на видеозаписи.
package gui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/hilsetapp/rp-reel/internal/cut"
	"github.com/hilsetapp/rp-reel/internal/evtlog"
	"github.com/hilsetapp/rp-reel/internal/ffmpeg"
	"github.com/hilsetapp/rp-reel/internal/report"
	"github.com/hilsetapp/rp-reel/internal/stats"
	"github.com/hilsetapp/rp-reel/internal/subtitle"
	"github.com/hilsetapp/rp-reel/internal/template"
	"github.com/hilsetapp/rp-reel/internal/timeline"
)

const thumbW = 170
const thumbH = thumbW * 9 / 16

type clipView struct {
	thumb image.Image
	job   cut.Job
}

type state struct {
	th *material.Theme

	videoEd, logEd, tsEd, offEd widget.Editor
	presetIdx                   int

	videoPath, logPath string
	videoTS            float64
	offset             float64

	log        *evtlog.Log
	duration   float64
	videoStart float64
	clips      []timeline.Clip
	st         *stats.Stats
	onVideo    int

	presets []template.Preset
	preset  template.Preset

	jobs   []cut.Job
	thumbs []clipView
	outDir string
	sel    int

	status  string
	journal []string
	busy    bool

	invalidate func()
}

func newState(th *material.Theme) *state {
	presets := template.List()
	s := &state{
		th:      th,
		presets: presets,
		preset:  presets[0],
		sel:     -1,
		status:  "выбери видео и журнал, потом нажми \"Собрать таймлайн\"",
	}
	return s
}

func (s *state) logLine(format string, a ...any) {
	s.journal = append(s.journal, fmt.Sprintf(format, a...))
	if len(s.journal) > 200 {
		s.journal = s.journal[len(s.journal)-200:]
	}
	if s.invalidate != nil {
		s.invalidate()
	}
}

// Run запускает окно приложения.
// Можно сразу передать файлы: rp-reel -video day.mp4 -log rpreel.jsonl,
// тогда таймлайн соберется сам после запуска.
func Run(version string) {
	var preVideo, preLog string
	args := os.Args[1:]
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "-video":
			preVideo = args[i+1]
		case "-log":
			preLog = args[i+1]
		}
	}
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("RP Reel - нарезка моментов RP-сервера"),
			app.Size(unit.Dp(1140), unit.Dp(760)),
		)
		th := newTheme()
		s := newState(th)
		s.invalidate = w.Invalidate
		s.videoEd.SetText(preVideo)
		s.logEd.SetText(preLog)
		if preVideo != "" && preLog != "" {
			go func() {
				time.Sleep(400 * time.Millisecond)
				s.build()
			}()
		}
		var ops op.Ops
		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				s.layout(gtx, version)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

func (s *state) layout(gtx layout.Context, version string) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.header(version)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(s.filesCard),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(s.settingsCard),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Flexed(1, s.mainArea),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(s.journalCard),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(s.statusStrip),
		)
	})
}

func (s *state) header(version string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H5(s.th, "RP Reel")
				lbl.Font.Weight = font.Bold
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(s.th, "нарезка моментов RP-сервера из видео и журнала событий")
				lbl.Color = colDim
				return lbl.Layout(gtx)
			}),
			layout.Flexed(1, layout.Spacer{}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Caption(s.th, version)
				lbl.Color = colDim
				return lbl.Layout(gtx)
			}),
		)
	}
}

func (s *state) filesCard(gtx layout.Context) layout.Dimensions {
	var videoPick, logPick widget.Clickable
	return surface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.H6(s.th, "Файлы").Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return material.Editor(s.th, &s.videoEd, "путь к видеозаписи, mp4 или mkv").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							for videoPick.Clicked(gtx) {
								if p := pickFileDialog("Выбери видеозапись", "Видео (*.mp4;*.mkv;*.mov)\x00*.mp4;*.mkv;*.mov\x00Все файлы (*.*)\x00*.*\x00\x00"); p != "" {
									s.videoEd.SetText(p)
								}
							}
							return material.Button(s.th, &videoPick, "Видео...").Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return material.Editor(s.th, &s.logEd, "журнал событий rpreel.jsonl с сервера").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							for logPick.Clicked(gtx) {
								if p := pickFileDialog("Выбери журнал событий", "JSON Lines (*.jsonl)\x00*.jsonl\x00Все файлы (*.*)\x00*.*\x00\x00"); p != "" {
									s.logEd.SetText(p)
								}
							}
							return material.Button(s.th, &logPick, "Журнал...").Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

func (s *state) settingsCard(gtx layout.Context) layout.Dimensions {
	var presetBtn widget.Clickable
	return surface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.Body2(s.th, "Пресет").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							for presetBtn.Clicked(gtx) {
								s.presetIdx = (s.presetIdx + 1) % len(s.presets)
								s.preset = s.presets[s.presetIdx]
							}
							btn := material.Button(s.th, &presetBtn, s.preset.Title)
							btn.Background = colSoft
							btn.Color = colText
							return btn.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.Body2(s.th, "Начало записи, Unix (0 = от первого события)").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Editor(s.th, &s.tsEd, "0").Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(material.Body2(s.th, "Сдвиг видео, сек").Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Editor(s.th, &s.offEd, "0").Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					hint := material.Caption(s.th, "Если события не совпадают с видео: поставь время начала записи или крути сдвиг.")
					hint.Color = colDim
					return hint.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(s.buildButton),
			)
		})
	})
}

func (s *state) buildButton(gtx layout.Context) layout.Dimensions {
	var btn widget.Clickable
	for btn.Clicked(gtx) {
		if !s.busy {
			s.build()
		}
	}
	b := material.Button(s.th, &btn, "Собрать таймлайн")
	if s.busy {
		b.Background = colLine
		b.Color = colDim
	}
	return b.Layout(gtx)
}

func (s *state) mainArea(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(0.62, func(gtx layout.Context) layout.Dimensions {
			return surface(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if len(s.thumbs) == 0 {
						lbl := material.Body1(s.th, "пока пусто: выбери видео и журнал, нажми \"Собрать таймлайн\"")
						lbl.Color = colDim
						lbl.Alignment = text.Middle
						return lbl.Layout(gtx)
					}
					var list widget.List
					list.Axis = layout.Vertical
					return material.List(s.th, &list).Layout(gtx, len(s.thumbs), func(gtx layout.Context, i int) layout.Dimensions {
						return s.clipCard(gtx, i)
					})
				})
			})
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(0.38, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(s.statsCard),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Flexed(1, s.selectedCard),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(s.actionsCard),
			)
		}),
	)
}

func (s *state) clipCard(gtx layout.Context, i int) layout.Dimensions {
	cv := s.thumbs[i]
	selected := s.sel == i
	var card widget.Clickable
	for card.Clicked(gtx) {
		s.sel = i
	}
	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return surfaceSel(gtx, selected, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(thumbWidget(cv.thumb)),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body1(s.th, cv.job.Label)
								lbl.Font.Weight = font.Medium
								if selected {
									lbl.Color = colAccent
								}
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								t := material.Caption(s.th, fmt.Sprintf("%s - %s  (%.0f сек)", clock(cv.job.Start), clock(cv.job.End), cv.job.End-cv.job.Start))
								t.Color = colDim
								return t.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if i < len(s.clips) {
									return eventList(s.th, gtx, s.clips[i].Events)
								}
								return layout.Dimensions{}
							}),
						)
					}),
				)
			})
		})
	})
}

func eventList(th *material.Theme, gtx layout.Context, events []evtlog.Event) layout.Dimensions {
	show := events
	if len(show) > 3 {
		show = show[:3]
	}
	var rows []layout.FlexChild
	for _, e := range show {
		e := e
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return chip(th, gtx, e)
		}))
	}
	if len(events) > 3 {
		more := fmt.Sprintf("... еще %d", len(events)-3)
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(th, more)
			lbl.Color = colDim
			return lbl.Layout(gtx)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func chip(th *material.Theme, gtx layout.Context, e evtlog.Event) layout.Dimensions {
	col := EventColor(e.Type)
	return layout.Inset{Right: unit.Dp(6), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return rounded(gtx, col, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(th, evtlog.TypeName(e.Type)+": "+name(e))
			lbl.Color = colOnAccent
			return layout.UniformInset(unit.Dp(3)).Layout(gtx, lbl.Layout)
		})
	})
}

func name(e evtlog.Event) string {
	if e.Player != "" {
		return e.Player
	}
	if e.Label != "" {
		return e.Label
	}
	return "?"
}

func (s *state) statsCard(gtx layout.Context) layout.Dimensions {
	return surface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if s.st == nil {
				lbl := material.Body2(s.th, "Статистика появится после сборки таймлайна")
				lbl.Color = colDim
				return lbl.Layout(gtx)
			}
			var rows []layout.FlexChild
			rows = append(rows, layout.Rigid(material.H6(s.th, "Статистика").Layout))
			rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
			rows = append(rows, layout.Rigid(statLine(s.th, "Событий в журнале", strconv.Itoa(len(s.log.Events)))))
			rows = append(rows, layout.Rigid(statLine(s.th, "Попало на видео", strconv.Itoa(s.onVideo))))
			rows = append(rows, layout.Rigid(statLine(s.th, "Клипов", strconv.Itoa(len(s.clips)))))
			rows = append(rows, layout.Rigid(statLine(s.th, "Длина видео", fmt.Sprintf("%.0f мин", s.duration/60))))
			rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
			rows = append(rows, layout.Rigid(material.Body2(s.th, "Типы:").Layout))
			for _, name := range sortedKeys(s.st.ByType) {
				rows = append(rows, layout.Rigid(statLine(s.th, evtlog.TypeName(name), strconv.Itoa(s.st.ByType[name]))))
			}
			rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
			rows = append(rows, layout.Rigid(material.Body2(s.th, "Топ игроков:").Layout))
			for _, p := range s.st.Top {
				rows = append(rows, layout.Rigid(statLine(s.th, p.Name, strconv.Itoa(p.Count))))
			}
			if s.st.Busiest.Count > 0 {
				rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body2(s.th, fmt.Sprintf("Самая горячая минута: %s (%d событий)", clock(s.st.Busiest.Start), s.st.Busiest.Count))
					lbl.Color = colAccent
					lbl.Font.Weight = font.Medium
					return lbl.Layout(gtx)
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		})
	})
}

func statLine(th *material.Theme, key, val string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(0.55, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, key)
				lbl.Color = colDim
				return lbl.Layout(gtx)
			}),
			layout.Flexed(0.45, func(gtx layout.Context) layout.Dimensions {
				return material.Body2(th, val).Layout(gtx)
			}),
		)
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *state) selectedCard(gtx layout.Context) layout.Dimensions {
	return surface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if s.sel < 0 || s.sel >= len(s.clips) {
				lbl := material.Body2(s.th, "Кликни по клипу на таймлайне")
				lbl.Color = colDim
				return lbl.Layout(gtx)
			}
			c := s.clips[s.sel]
			var rows []layout.FlexChild
			rows = append(rows, layout.Rigid(material.H6(s.th, "Выбранный клип").Layout))
			rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
			rows = append(rows, layout.Rigid(statLine(s.th, "Время", fmt.Sprintf("%s - %s", clock(c.Start), clock(c.End)))))
			rows = append(rows, layout.Rigid(statLine(s.th, "Событий", strconv.Itoa(len(c.Events)))))
			rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
			for _, e := range c.Events {
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body2(s.th, e.Caption()).Layout(gtx)
				}))
			}
			rows = append(rows, layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout))
			rows = append(rows, layout.Rigid(s.playButton()))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		})
	})
}

func (s *state) playButton() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		var btn widget.Clickable
		for btn.Clicked(gtx) {
			s.playSelected()
		}
		return material.Button(s.th, &btn, "Показать в плеере").Layout(gtx)
	}
}

func (s *state) actionsCard(gtx layout.Context) layout.Dimensions {
	var cutBtn, openBtn widget.Clickable
	return surface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(material.H6(s.th, "Результат").Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					for cutBtn.Clicked(gtx) {
						if !s.busy && len(s.jobs) > 0 {
							s.render()
						}
					}
					b := material.Button(s.th, &cutBtn, "Вырезать ролик")
					b.Background = colAccent
					b.Color = colOnAccent
					if len(s.jobs) == 0 {
						b.Background = colLine
						b.Color = colDim
					}
					return b.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					for openBtn.Clicked(gtx) {
						if s.outDir != "" {
							openFolder(s.outDir)
						}
					}
					lbl := material.Button(s.th, &openBtn, "Открыть папку результата")
					if s.outDir == "" {
						lbl.Background = colLine
						lbl.Color = colDim
					}
					return lbl.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					txt := material.Caption(s.th, "Клипы пишутся рядом с видео, в папку reel-out.")
					txt.Color = colDim
					return txt.Layout(gtx)
				}),
			)
		})
	})
}

func (s *state) journalCard(gtx layout.Context) layout.Dimensions {
	return surface(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if len(s.journal) == 0 {
				return layout.Dimensions{}
			}
			lines := s.journal
			var list widget.List
			list.Axis = layout.Vertical
			return material.List(s.th, &list).Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
				lbl := material.Caption(s.th, lines[len(lines)-1-i])
				lbl.Color = colDim
				return lbl.Layout(gtx)
			})
		})
	})
}

func (s *state) statusStrip(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(s.th, s.status)
			lbl.Color = colDim
			return lbl.Layout(gtx)
		}),
		layout.Flexed(1, layout.Spacer{}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !s.busy {
				return layout.Dimensions{}
			}
			lbl := material.Caption(s.th, "работаю...")
			lbl.Color = colAccent
			return lbl.Layout(gtx)
		}),
	)
}

// build собирает таймлайн в фоновой горутине, окно не виснет.
func (s *state) build() {
	s.busy = true
	s.status = "собираю таймлайн..."
	go func() {
		defer func() {
			s.busy = false
			s.status = "готово"
			if s.invalidate != nil {
				s.invalidate()
			}
		}()
		s.doBuild()
	}()
}

func (s *state) doBuild() {
	s.videoPath = strings.TrimSpace(s.videoEd.Text())
	s.logPath = strings.TrimSpace(s.logEd.Text())
	if s.videoPath == "" || s.logPath == "" {
		s.status = "выбери видео и журнал"
		return
	}
	tools, err := ffmpeg.Find("", "")
	if err != nil {
		s.status = "ffmpeg не найден: поставь его и добавь в PATH"
		return
	}

	log, err := evtlog.Load(s.logPath)
	if err != nil {
		s.status = fmt.Sprintf("журнал не читается: %v", err)
		return
	}
	if len(log.Events) == 0 {
		s.status = "в журнале нет событий"
		return
	}
	s.log = log
	s.logLine("журнал: %d событий", len(log.Events))
	if log.Skipped > 0 {
		s.logLine("строк не разобрано: %d", log.Skipped)
	}

	dur, err := ffmpeg.Duration(tools, s.videoPath)
	if err != nil {
		s.status = fmt.Sprintf("видео не читается: %v", err)
		return
	}
	s.duration = dur

	s.videoTS = parseFloat(s.tsEd.Text(), 0)
	s.offset = parseFloat(s.offEd.Text(), 0)
	s.videoStart = log.Events[0].TS - s.offset
	if s.videoTS > 0 {
		s.videoStart = s.videoTS
	}

	w := timeline.Window{Pre: s.preset.Pre, Post: s.preset.Post, MinLen: s.preset.MinLen, MergeGap: s.preset.MergeGap}
	clips := timeline.Build(log.Events, s.videoStart, dur, w)
	if len(clips) == 0 {
		s.status = "ни одно событие не попало на видео: проверь синхронизацию"
		return
	}
	onVideo := 0
	for _, c := range clips {
		onVideo += len(c.Events)
	}
	if s.preset.MaxClips > 0 && len(clips) > s.preset.MaxClips {
		ranked := make([]timeline.Clip, len(clips))
		copy(ranked, clips)
		sort.SliceStable(ranked, func(i, j int) bool {
			return len(ranked[i].Events) > len(ranked[j].Events)
		})
		keep := ranked[:s.preset.MaxClips]
		sort.SliceStable(keep, func(i, j int) bool { return keep[i].Start < keep[j].Start })
		clips = keep
	}
	s.clips = clips
	s.onVideo = onVideo
	s.st = stats.Compute(log.Events, s.videoStart, dur)
	s.sel = -1

	s.outDir = filepath.Join(filepath.Dir(s.videoPath), "reel-out")
	if err := os.MkdirAll(filepath.Join(s.outDir, "thumbs"), 0o755); err != nil {
		s.status = fmt.Sprintf("не создаю папку: %v", err)
		return
	}

	s.jobs = cut.Plan(s.videoPath, clips, s.outDir, s.preset)
	s.logLine("клипов: %d, пресет %q", len(clips), s.preset.Name)

	s.thumbs = make([]clipView, len(clips))
	for i := range clips {
		mid := (clips[i].Start + clips[i].End) / 2
		path := filepath.Join(s.outDir, "thumbs", fmt.Sprintf("clip-%03d.png", i+1))
		if err := ffmpeg.Thumb(tools, s.videoPath, mid, path); err == nil {
			if data, err := os.ReadFile(path); err == nil {
				if img, err := png.Decode(bytes.NewReader(data)); err == nil {
					s.thumbs[i].thumb = img
				}
			}
		}
		s.thumbs[i].job = s.jobs[i]
		s.logLine("миниатюра %d/%d", i+1, len(clips))
	}
	s.status = fmt.Sprintf("таймлайн готов: %d клипов, можно вырезать ролик", len(clips))
}

// render вырезает клипы и склеивает ролик в фоновой горутине.
func (s *state) render() {
	s.busy = true
	s.status = "режу клипы..."
	go func() {
		defer func() {
			s.busy = false
			s.status = "готово"
			if s.invalidate != nil {
				s.invalidate()
			}
		}()
		s.doRender()
	}()
}

func (s *state) doRender() {
	tools, err := ffmpeg.Find("", "")
	if err != nil {
		s.status = "ffmpeg не найден"
		return
	}
	for _, j := range s.jobs {
		s.logLine("вырезаю clip-%03d.mp4 %s-%s", j.Index, clock(j.Start), clock(j.End))
		if err := ffmpeg.Run(tools.FFmpeg, j.Args...); err != nil {
			s.status = fmt.Sprintf("клип %d: %v", j.Index, err)
			return
		}
	}
	listPath := filepath.Join(s.outDir, "concat.txt")
	if err := os.WriteFile(listPath, []byte(cut.ConcatList(s.jobs)), 0o644); err != nil {
		s.status = fmt.Sprintf("список для склейки: %v", err)
		return
	}
	reelPath := filepath.Join(s.outDir, "reel.mp4")
	s.logLine("склеиваю reel.mp4")
	if err := ffmpeg.Run(tools.FFmpeg, cut.ConcatArgs(tools.FFmpeg, listPath, reelPath, s.preset)...); err != nil {
		s.status = fmt.Sprintf("склейка: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(s.outDir, "report.txt"), []byte(report.Text("gui", s.videoPath, s.logPath, s.preset, s.duration, s.log, s.clips, s.st)), 0o644); err != nil {
		s.logLine("отчет не записан: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.outDir, "digest.json"), report.Digest("gui", s.videoPath, s.logPath, s.preset, s.videoStart, s.duration, s.log, s.clips, s.st), 0o644); err != nil {
		s.logLine("дайджест не записан: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.outDir, "events.srt"), []byte(subtitle.SRT(s.clips)), 0o644); err != nil {
		s.logLine("субтитры не записаны: %v", err)
	}
	s.status = fmt.Sprintf("готово: %s (%d клипов)", reelPath, len(s.jobs))
	s.logLine("ролик готов: %s", reelPath)
	openFolder(s.outDir)
}

func (s *state) playSelected() {
	if s.sel < 0 || s.sel >= len(s.jobs) {
		return
	}
	j := s.jobs[s.sel]
	for _, player := range []string{"vlc", "mpv", "mpv.exe"} {
		if p, err := exec.LookPath(player); err == nil {
			args := []string{j.Path}
			if player == "vlc" {
				args = append(args, "--start-time", strconv.FormatFloat(j.Start, 'f', 1, 64))
			} else {
				args = append(args, fmt.Sprintf("--start=%f", j.Start))
			}
			exec.Command(p, args...).Start()
			return
		}
	}
	openFolder(filepath.Dir(j.Path))
}

// рисование

func surface(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return surfaceSel(gtx, false, w)
}

func surfaceSel(gtx layout.Context, selected bool, w layout.Widget) layout.Dimensions {
	bg := colCard
	if selected {
		bg = colSoft
	}
	rr := clip.RRect{
		Rect: rectOf(gtx), SE: 10, SW: 10, NE: 10, NW: 10,
	}
	defer rr.Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	return w(gtx)
}

func rounded(gtx layout.Context, col color.NRGBA, w layout.Widget) layout.Dimensions {
	rr := clip.RRect{
		Rect: rectOf(gtx), SE: 8, SW: 8, NE: 8, NW: 8,
	}
	defer rr.Push(gtx.Ops).Pop()
	paint.FillShape(gtx.Ops, col, rr.Op(gtx.Ops))
	return w(gtx)
}

func rectOf(gtx layout.Context) image.Rectangle {
	return image.Rectangle{Max: gtx.Constraints.Max}
}

func thumbWidget(img image.Image) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		sz := image.Point{X: thumbW, Y: thumbH}
		rect := image.Rectangle{Max: sz}
		rr := clip.RRect{Rect: rect, SE: 6, SW: 6, NE: 6, NW: 6}
		defer rr.Push(gtx.Ops).Pop()
		if img == nil {
			paint.FillShape(gtx.Ops, colSoft, rr.Op(gtx.Ops))
			return layout.Dimensions{Size: sz}
		}
		inner := gtx
		inner.Constraints = layout.Exact(sz)
		im := widget.Image{Src: paint.NewImageOp(img), Fit: widget.Contain, Position: layout.Center}
		return im.Layout(inner)
	}
}

func clock(t float64) string {
	if t < 0 {
		t = 0
	}
	m := int(t) / 60
	sec := int(t) % 60
	return fmt.Sprintf("%02d:%02d", m, sec)
}

func parseFloat(text string, def float64) float64 {
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
	if text == "" {
		return def
	}
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return def
	}
	return v
}

func openFolder(path string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("explorer", path).Start()
	case "darwin":
		exec.Command("open", path).Start()
	default:
		exec.Command("xdg-open", path).Start()
	}
}
