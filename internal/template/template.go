// Package template - галерея пресетов нарезки.
//
// Пресеты живут в presets/*.json, их можно добавлять своими:
// положить файл в папку templates рядом с приложением или в
// пользовательскую папку настроек.
package template

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Preset - один пресет галереи.
type Preset struct {
	Name      string   `json:"name"`
	Title     string   `json:"title"`
	Note      string   `json:"note"`
	Pre       float64  `json:"pre"`
	Post      float64  `json:"post"`
	MinLen    float64  `json:"minLen"`
	MergeGap  float64  `json:"mergeGap"`
	MaxClips  int      `json:"maxClips"`
	Reencode  bool     `json:"reencode"`
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

//go:embed presets/*.json
var embedded embed.FS

// Base - пресет по умолчанию, если галерея недоступна.
func Base() Preset {
	return Preset{
		Name: "base", Title: "Обычная нарезка",
		Note: "Окно 10 секунд до и после события, склейка без перекодирования.",
		Pre:  10, Post: 10, MinLen: 3, MergeGap: 2,
	}
}

// List возвращает все пресеты: встроенные и пользовательские
// из папки templates рядом с приложением.
func List() []Preset {
	presets := []Preset{Base()}
	for _, name := range embeddedNames() {
		if p, err := load(embedded, name); err == nil && p.Name != "base" {
			presets = append(presets, p)
		}
	}
	presets = append(presets, userPresets()...)
	return presets
}

// Get берет пресет по имени. Пустое имя - пресет по умолчанию.
func Get(name string) (Preset, error) {
	if name == "" {
		return Base(), nil
	}
	for _, p := range List() {
		if p.Name == name {
			return p, nil
		}
	}
	return Preset{}, fmt.Errorf("пресет %q не найден, есть: %s", name, names())
}

func names() string {
	var out []string
	for _, p := range List() {
		out = append(out, p.Name)
	}
	return fmt.Sprintf("%v", out)
}

func embeddedNames() []string {
	ents, err := embedded.ReadDir("presets")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			out = append(out, "presets/"+e.Name())
		}
	}
	return out
}

func load(fsys fs.FS, name string) (Preset, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Preset{}, err
	}
	var p Preset
	if err := json.Unmarshal(data, &p); err != nil {
		return Preset{}, err
	}
	return p, nil
}

func userPresets() []Preset {
	dir := "templates"
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Preset
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if p, err := load(os.DirFS(dir), e.Name()); err == nil && p.Name != "" {
			out = append(out, p)
		}
	}
	return out
}
