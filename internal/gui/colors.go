package gui

import "image/color"

// Спокойные цвета без неона: светлый фон, тёмный текст, мягкие плашки.
var (
	colBg       = color.NRGBA{R: 0xF4, G: 0xF5, B: 0xF7, A: 0xFF}
	colCard     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	colLine     = color.NRGBA{R: 0xDF, G: 0xE3, B: 0xE8, A: 0xFF}
	colText     = color.NRGBA{R: 0x1B, G: 0x1F, B: 0x24, A: 0xFF}
	colDim      = color.NRGBA{R: 0x63, G: 0x6B, B: 0x75, A: 0xFF}
	colAccent   = color.NRGBA{R: 0x2F, G: 0x5D, B: 0x8C, A: 0xFF}
	colOnAccent = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	colSoft     = color.NRGBA{R: 0xEC, G: 0xF1, B: 0xF6, A: 0xFF}
	colWarnBg   = color.NRGBA{R: 0xFD, G: 0xF3, B: 0xDF, A: 0xFF}
	colWarnText = color.NRGBA{R: 0x8A, G: 0x5A, B: 0x12, A: 0xFF}
	colBadBg    = color.NRGBA{R: 0xFB, G: 0xE7, B: 0xE6, A: 0xFF}
	colBadText  = color.NRGBA{R: 0x99, G: 0x2B, B: 0x27, A: 0xFF}
	colOkText   = color.NRGBA{R: 0x1E, G: 0x63, B: 0x3A, A: 0xFF}
)

// Цвета меток событий на таймлайне.
var (
	colKill    = color.NRGBA{R: 0xB3, G: 0x26, B: 0x1E, A: 0xFF}
	colWreck   = color.NRGBA{R: 0xB2, G: 0x6A, B: 0x00, A: 0xFF}
	colArrest  = color.NRGBA{R: 0x1A, G: 0x5F, B: 0xB4, A: 0xFF}
	colRob     = color.NRGBA{R: 0x6A, G: 0x3F, B: 0xA0, A: 0xFF}
	colCustom  = color.NRGBA{R: 0x0B, G: 0x72, B: 0x85, A: 0xFF}
	colNeutral = color.NRGBA{R: 0x5A, G: 0x64, B: 0x72, A: 0xFF}
)

// EventColor - цвет метки по типу события.
func EventColor(t string) color.NRGBA {
	switch t {
	case "kill":
		return colKill
	case "wreck":
		return colWreck
	case "arrest":
		return colArrest
	case "rob":
		return colRob
	case "custom":
		return colCustom
	default:
		return colNeutral
	}
}
