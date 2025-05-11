package logging

import (
	"log/slog"
	"os"
)

var L *slog.Logger

func init() {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	L = slog.New(h)
}
