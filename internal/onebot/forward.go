package onebot

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"statusbot/internal/config"
)

// RunForward 维护一条正向 WebSocket 连接，断线后指数退避重连。
func RunForward(ctx context.Context, cfg config.Connection, h Handler, lg *log.Logger) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		u, err := url.Parse(cfg.URL)
		if err != nil {
			lg.Printf("[%s] 地址无效: %v", cfg.Name, err)
			return
		}
		header := http.Header{}
		if cfg.Token != "" {
			q := u.Query()
			q.Set("access_token", cfg.Token)
			u.RawQuery = q.Encode()
			header.Set("Authorization", "Bearer "+cfg.Token)
		}

		dialer := websocket.Dialer{
			HandshakeTimeout: 12 * time.Second,
			Proxy:            http.ProxyFromEnvironment,
		}
		conn, resp, err := dialer.DialContext(ctx, u.String(), header)
		if err != nil {
			if resp != nil {
				lg.Printf("[%s] 连接 %s 失败: %v (HTTP %d)", cfg.Name, cfg.URL, err, resp.StatusCode)
			} else {
				lg.Printf("[%s] 连接 %s 失败: %v", cfg.Name, cfg.URL, err)
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}

		lg.Printf("[%s] 已连接: %s", cfg.Name, cfg.URL)
		backoff = time.Second
		sess := newSession(cfg.Name, conn)
		err = sess.run(ctx, h, lg)
		conn.Close()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			lg.Printf("[%s] 连接断开: %v", cfg.Name, err)
		}
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
	}
}
