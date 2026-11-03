package onebot

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"statusbot/internal/config"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestReverseTokenAndPath(t *testing.T) {
	port := freePort(t)
	events := make(chan *Event, 1)
	conns := []config.Connection{{
		Type: "reverse", Name: "rev", Port: port, Path: "/ws", Token: "secret",
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := StartReverses(ctx, conns,
		func(ctx context.Context, s Session, ev *Event) { events <- ev },
		log.New(io.Discard, "", 0)); err != nil {
		t.Fatal(err)
	}

	base := "http://127.0.0.1:" + itoa(port)

	resp, err := http.Get(base + "/ws?access_token=wrong")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("错误 token 应返回 401，实际 %d", resp.StatusCode)
	}

	resp, err = http.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d", resp.StatusCode)
	}

	wsURL := "ws://127.0.0.1:" + itoa(port) + "/ws?access_token=secret"
	dialer := websocket.Dialer{HandshakeTimeout: 3 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"post_type":    "message",
		"message_type": "private",
		"self_id":      1,
		"user_id":      42,
		"message_id":   43,
		"message":      "状态",
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-events:
		if ev.UserID.Int64() != 42 {
			t.Fatalf("事件异常: %+v", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("未收到反向事件")
	}

	resp, err = http.Get(base + "/other")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("错误路径应 404，实际 %d", resp.StatusCode)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
