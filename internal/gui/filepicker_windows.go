//go:build windows

package gui

import (
	"runtime"
	"syscall"
	"unsafe"
)

// Здесь только системный диалог Windows выбора файла.
// Пишем через winapi напрямую, чтобы не тащить в сборку лишние библиотеки.

var (
	comdlg32            = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

const (
	ofnExplorer     = 0x00080000
	ofnFileMustExit = 0x00000800
	ofnPathMustExit = 0x00001000
)

func utf16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

// pickFileDialog возвращает путь к выбранному файлу или пустую строку.
// filter вида "Видео (*.mp4)\x00*.mp4\x00Все файлы (*.*)\x00*.*\x00\x00".
func pickFileDialog(title, filter string) string {
	buf := make([]uint16, 2048)
	ofn := openFileNameW{
		lStructSize:  uint32(unsafe.Sizeof(openFileNameW{})),
		lpstrFilter:  utf16(filter),
		nFilterIndex: 1,
		lpstrFile:    &buf[0],
		nMaxFile:     uint32(len(buf)),
		lpstrTitle:   utf16(title),
		flags:        ofnExplorer | ofnFileMustExit | ofnPathMustExit,
	}
	ret, _, _ := procGetOpenFileName.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(ofn.lpstrFilter)
	runtime.KeepAlive(ofn.lpstrTitle)
	runtime.KeepAlive(buf)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
