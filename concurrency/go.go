package concurrency

import (
	"context"
	"log/slog"
	"runtime"
)

type AsyncContext struct {
	context.Context
}

func (c *AsyncContext) Done() <-chan struct{} {
	return nil
}

func Go(ctx context.Context, fn GoFunc) {
	go func(ctx context.Context) {
		defer func() {
			if err := recover(); err != nil {
				var buf [4096]byte
				n := runtime.Stack(buf[:], false)
				slog.ErrorContext(ctx, "[Go] System Panic, err: %v, detail: %v", err, buf[:n])
			}
		}()

		fn(ctx)
	}(&AsyncContext{ctx})
}
