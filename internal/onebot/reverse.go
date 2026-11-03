package onebot

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"statusbot/internal/config"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// StartReverses 按端口分组启动反向 WebSocket 服务，同端口可挂多个路径与 token。
func StartReverses(ctx context.Context, conns []config.Connection, h Handler, lg *log.Logger) error {
	groups := map[int][]config.Connection{}
	var ports []int
	for _, c := range conns {
		if c.Type != "reverse" || !c.IsEnabled() {
			continue
		}
		if _, ok := groups[c.Port]; !ok {
			ports = append(ports, c.Port)
		}
		groups[c.Port] = append(groups[c.Port], c)
	}
	if len(groups) == 0 {
		return nil
	}

	started := 0
	for _, port := range ports {
		routes := groups[port]
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			lg.Printf("反向监听 :%d 失败: %v", port, err)
			continue
		}
		started++

		mux := http.NewServeMux()
		hasHealth := false
		for _, route := range routes {
			rt := route
			if rt.Path == "/healthz" {
				hasHealth = true
			}
			mux.HandleFunc(rt.Path, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != rt.Path {
					http.NotFound(w, r)
					return
				}
				if !authorized(r, rt.Token) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				lg.Printf("[%s] 反向连接已建立: %s", rt.Name, r.RemoteAddr)
				sess := newSession(rt.Name, conn)
				go func() {
					if err := sess.run(ctx, h, lg); err != nil && ctx.Err() == nil {
						lg.Printf("[%s] 反向连接断开: %v", rt.Name, err)
					}
					conn.Close()
				}()
			})
		}
		if !hasHealth {
			mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				fmt.Fprintln(w, "ok")
			})
		}

		srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		go func() {
			<-ctx.Done()
			shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(shCtx)
		}()
		go func(port int, srv *http.Server) {
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				lg.Printf("反向服务 :%d 退出: %v", port, err)
			}
		}(port, srv)

		lg.Printf("反向 WebSocket 已监听 :%d，路径 %s", port, pathList(routes))
	}

	if started == 0 {
		return errors.New("所有反向端口监听失败")
	}
	return nil
}

func pathList(routes []config.Connection) string {
	parts := make([]string, 0, len(routes))
	for _, r := range routes {
		parts = append(parts, r.Path)
	}
	return strings.Join(parts, ", ")
}

func tokenEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func authorized(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(auth) >= 7 && strings.EqualFold(auth[:7], "Bearer ") {
		auth = strings.TrimSpace(auth[7:])
	}
	if tokenEqual(auth, token) {
		return true
	}
	if tokenEqual(r.Header.Get("X-Access-Token"), token) {
		return true
	}
	return tokenEqual(r.URL.Query().Get("access_token"), token)
}
