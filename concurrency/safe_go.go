package concurrency

import (
	"context"
	"log/slog"
	"runtime"
)

type GoFunc func()

func SafeGo(ctx context.Context, fn GoFunc) {
	defer func() {
		if r := recover(); r != nil {
			buf := [4096]byte{}
			n := runtime.Stack(buf[:], false)
			slog.ErrorContext(ctx, "Go Program Panic", slog.Any("stack", string(buf[:n])))
		}
		fn()
	}()
}
