package render

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"statusbot/internal/config"
	"statusbot/internal/sysinfo"
)

// ErrBusy 表示排队人数超过上限。
var ErrBusy = errors.New("渲染队列已满")

// Engine 提供带缓存与并发闸门的出图能力。
type Engine struct {
	cfg     *config.Store
	sampler *sysinfo.Sampler
	sem     chan struct{}
	pending int32

	mu         sync.Mutex
	cachePNG   []byte
	cacheAt    time.Time
	cacheWidth int
}

func NewEngine(cfg *config.Store, sampler *sysinfo.Sampler) *Engine {
	limit := cfg.Get().Render.Concurrency
	if limit < 1 {
		limit = 1
	}
	return &Engine{
		cfg:     cfg,
		sampler: sampler,
		sem:     make(chan struct{}, limit),
	}
}

// Reset 在配置热更新后清空缓存。
func (e *Engine) Reset() {
	e.mu.Lock()
	e.cachePNG = nil
	e.cacheAt = time.Time{}
	e.mu.Unlock()
}

func (e *Engine) PNG(ctx context.Context) ([]byte, error) {
	r := e.cfg.Get().Render
	if png := e.cached(r); png != nil {
		return png, nil
	}

	limit := r.Concurrency
	if limit < 1 {
		limit = 1
	}
	maxPending := limit + r.Queue
	if int(atomic.AddInt32(&e.pending, 1)) > maxPending {
		atomic.AddInt32(&e.pending, -1)
		return nil, ErrBusy
	}
	defer atomic.AddInt32(&e.pending, -1)

	select {
	case e.sem <- struct{}{}:
		defer func() { <-e.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if png := e.cached(r); png != nil {
		return png, nil
	}
	snap := e.sampler.Snapshot()
	if snap == nil {
		return nil, errors.New("等待采样数据")
	}
	png, err := Render(snap, r)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	e.cachePNG = png
	e.cacheAt = time.Now()
	e.cacheWidth = r.Width
	e.mu.Unlock()
	return png, nil
}

func (e *Engine) cached(r config.Render) []byte {
	if r.CacheSeconds <= 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cachePNG == nil || e.cacheWidth != r.Width {
		return nil
	}
	if time.Since(e.cacheAt) > time.Duration(r.CacheSeconds)*time.Second {
		return nil
	}
	return e.cachePNG
}
