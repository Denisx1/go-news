// Package parallel — вспомогательные функции для конкурентной работы.
package parallel

import (
	"context"
	"sync"
	"time"
)

// ForEach вызывает fn(0..n-1); одновременно выполняется не больше workers вызовов.
func ForEach(ctx context.Context, workers, n int, fn func(i int)) {
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		if ctx.Err() != nil {
			<-sem
			break
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i)
		}(i)
	}
	wg.Wait()
}

// Sleep — пауза, которую можно прервать отменой контекста.
func Sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
