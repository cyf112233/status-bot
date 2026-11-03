package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"statusbot/internal/config"
	"statusbot/internal/onebot"
	"statusbot/internal/render"
	"statusbot/internal/sysinfo"
)

func TestEndToEndForward(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	yamlBody := `
commands: ["状态"]
users: []
render:
  width: 1080
  concurrency: 1
  cache_seconds: 3
  queue: 4
  cooldown: 0
  top_processes: 5
  max_disks: 6
  max_interfaces: 3
connections: []
`
	if err := os.WriteFile(cfgPath, []byte(yamlBody), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	actionCh := make(chan []byte, 1)
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{
			"post_type":    "message",
			"message_type": "group",
			"self_id":      1,
			"user_id":      10001,
			"group_id":     20002,
			"message_id":   30003,
			"message":      []map[string]any{{"type": "text", "data": map[string]any{"text": "状态"}}},
		})
		_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		_, data, err := conn.ReadMessage()
		if err == nil {
			actionCh <- data
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := config.NewStore(cfg)
	sampler := sysinfo.New(time.Second, 5, 6)
	sampler.Start(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for sampler.Snapshot() == nil && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if sampler.Snapshot() == nil {
		t.Fatal("采样未就绪")
	}

	lg := log.New(io.Discard, "", 0)
	engine := render.NewEngine(store, sampler)
	bot := NewBot(store, engine, lg)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	go onebot.RunForward(ctx, config.Connection{Type: "forward", Name: "e2e", URL: url}, bot.Handle, lg)

	var data []byte
	select {
	case data = <-actionCh:
	case <-time.After(20 * time.Second):
		t.Fatal("未收到发送动作")
	}

	var action struct {
		Action string `json:"action"`
		Params struct {
			GroupID int64 `json:"group_id"`
			Message []struct {
				Type string `json:"type"`
				Data struct {
					File string `json:"file"`
				} `json:"data"`
			} `json:"message"`
		} `json:"params"`
	}
	if err := json.Unmarshal(data, &action); err != nil {
		t.Fatalf("动作解析失败: %v (%s)", err, data)
	}
	if action.Action != "send_group_msg" || action.Params.GroupID != 20002 {
		t.Fatalf("动作异常: %+v", action)
	}
	if len(action.Params.Message) != 1 || action.Params.Message[0].Type != "image" {
		t.Fatalf("消息段异常: %+v", action.Params.Message)
	}
	file := action.Params.Message[0].Data.File
	if !strings.HasPrefix(file, "base64://") {
		t.Fatalf("图片格式异常: %s", file)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(file, "base64://"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1080 || img.Bounds().Dy() < 800 {
		t.Fatalf("图片尺寸异常: %v", img.Bounds())
	}
}
