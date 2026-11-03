package onebot

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"statusbot/internal/config"
)

func TestForwardReceivesEventAndSendsToken(t *testing.T) {
	events := make(chan *Event, 1)
	authOK := make(chan bool, 1)
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authOK <- r.Header.Get("Authorization") == "Bearer tok"
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{
			"post_type":    "message",
			"message_type": "group",
			"self_id":      1,
			"user_id":      7,
			"group_id":     8,
			"message_id":   9,
			"message":      []map[string]any{{"type": "text", "data": map[string]any{"text": "状态"}}},
		})
		<-time.After(400 * time.Millisecond)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/onebot/v11/ws"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go RunForward(ctx, config.Connection{Type: "forward", Name: "test", URL: url, Token: "tok"},
		func(ctx context.Context, s Session, ev *Event) { events <- ev },
		log.New(io.Discard, "", 0))

	select {
	case ev := <-events:
		if ev.UserID.Int64() != 7 || ev.GroupID.Int64() != 8 || ev.MessageType != "group" {
			t.Fatalf("事件内容异常: %+v", ev)
		}
		if text, ok := Text(ev); !ok || text != "状态" {
			t.Fatalf("文本解析异常: %q %v", text, ok)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("未收到事件")
	}
	select {
	case ok := <-authOK:
		if !ok {
			t.Fatal("未携带 Bearer token")
		}
	case <-time.After(time.Second):
		t.Fatal("服务端未收到连接")
	}
}
