// Package evtlog читает журнал событий RP-сервера.
//
// Формат - JSONL, по одной событию на строку, пишет плагин rpreel:
//
//	{"ts":1729440000,"type":"kill","player":"Nick","other":"Victim","weapon":"34"}
//
// Допустим и один массив событий в квадратных скобках. Порядок строк
// не важен: события сортируются по времени.
package evtlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Типы событий, которые понимает приложение без дополнительной настройки.
const (
	Kill   = "kill"
	Death  = "death"
	Wreck  = "wreck" // уничтожена техника
	Arrest = "arrest"
	Rob    = "rob" // ограбление
	Admin  = "admin"
	Join   = "join"
	Leave  = "leave"
	Custom = "custom"
)

// Event - одно событие на сервере. TS - время в секундах Unix.
type Event struct {
	TS     float64 `json:"ts"`
	Type   string  `json:"type"`
	Player string  `json:"player,omitempty"`
	Other  string  `json:"other,omitempty"` // вторая сторона: жертва, напарник
	Weapon string  `json:"weapon,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Z      float64 `json:"z,omitempty"`
	Label  string  `json:"label,omitempty"` // свободный текст для type=custom
}

// Log - прочитанный журнал.
type Log struct {
	Events  []Event
	Skipped int      // строк, которые не удалось разобрать
	Errors  []string // описания плохих строк, для отчета
}

// Load читает журнал из файла. Пустые строки игнорируются.
func Load(path string) (*Log, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse разбирает журнал из потока: JSONL или один массив.
func Parse(r io.Reader) (*Log, error) {
	br := bufio.NewReader(r)
	first, err := br.Peek(1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(first) > 0 && first[0] == '[' {
		var events []Event
		if err := json.NewDecoder(br).Decode(&events); err != nil {
			return nil, fmt.Errorf("массив событий не разобрался: %w", err)
		}
		return &Log{Events: events}, nil
	}

	log := &Log{}
	sc := bufio.NewScanner(br)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(text), &e); err != nil {
			log.Skipped++
			log.Errors = append(log.Errors, fmt.Sprintf("строка %d: %v", line, err))
			continue
		}
		if e.TS == 0 {
			log.Skipped++
			log.Errors = append(log.Errors, fmt.Sprintf("строка %d: нет поля ts", line))
			continue
		}
		log.Events = append(log.Events, e)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(log.Events, func(i, j int) bool {
		return log.Events[i].TS < log.Events[j].TS
	})
	return log, nil
}

// TypeName - человекочитаемое имя типа события.
func TypeName(t string) string {
	switch t {
	case Kill:
		return "убийство"
	case Death:
		return "смерть"
	case Wreck:
		return "разбивка"
	case Arrest:
		return "арест"
	case Rob:
		return "ограбление"
	case Admin:
		return "админ"
	case Join:
		return "вход"
	case Leave:
		return "выход"
	case Custom:
		return "событие"
	default:
		return strings.ToLower(t)
	}
}

// Caption - короткая подпись события для таймлайна и субтитров.
func (e Event) Caption() string {
	who := e.Player
	if who == "" {
		who = "?"
	}
	switch e.Type {
	case Kill:
		if e.Other != "" {
			return fmt.Sprintf("убийство: %s → %s", who, e.Other)
		}
		return "убийство: " + who
	case Death:
		return "смерть: " + who
	case Wreck:
		if e.Other != "" {
			return fmt.Sprintf("разбивка: %s → %s", who, e.Other)
		}
		return "разбивка: " + who
	case Arrest:
		if e.Other != "" {
			return fmt.Sprintf("арест: %s → %s", who, e.Other)
		}
		return "арест: " + who
	case Rob:
		if e.Other != "" {
			return fmt.Sprintf("ограбление: %s → %s", who, e.Other)
		}
		return "ограбление: " + who
	case Custom:
		if e.Label != "" {
			return e.Label
		}
		return "событие: " + who
	default:
		return TypeName(e.Type) + ": " + who
	}
}
