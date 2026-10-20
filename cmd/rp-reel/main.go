// rp-reel - GUI для нарезки RP-видео по журналу событий сервера.
package main

import (
	"github.com/hilsetapp/rp-reel/internal/gui"
)

var version = "dev"

func main() {
	gui.Run(version)
}
