//go:build !windows

package gui

// На Linux и macOS пока пишем путь руками: системный диалог
// подключим позже, а журнал и журнал событий всё равно выбирает
// админ сервера один раз.
func pickFileDialog(title, filter string) string {
	return ""
}
