package concurrency

import (
	"context"
	"sync"
)

var (
	defaultLimit = 10
)

func GoWaitWithLimit(ctx context.Context, limit int, fns ...GoFunc) {
	if limit <= 0 {
		limit = defaultLimit
	}

	wg, ch := sync.WaitGroup{}, make(chan struct{}, limit)
	wg.Add(len(fns))
	for _, fn := range fns {
		ch <- struct{}{}

		Go(ctx, func(ctx context.Context) {
			defer func() {
				<-ch
				wg.Done()
			}()
			fn(ctx)
		})
	}
	wg.Wait()
}
