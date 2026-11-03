package onebot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var echoSeq uint64

type wsSession struct {
	name    string
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func newSession(name string, conn *websocket.Conn) *wsSession {
	return &wsSession{name: name, conn: conn}
}

func (s *wsSession) Name() string { return s.name }

func (s *wsSession) Call(ctx context.Context, action string, params any) error {
	payload := map[string]any{
		"action": action,
		"params": params,
		"echo":   fmt.Sprintf("statusbot-%d", atomic.AddUint64(&echoSeq, 1)),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return s.conn.WriteMessage(websocket.TextMessage, b)
}

func (s *wsSession) run(ctx context.Context, h Handler, lg *log.Logger) error {
	s.conn.SetReadLimit(8 << 20)
	stop := make(chan struct{})
	defer close(stop)

	go func() {
		select {
		case <-ctx.Done():
			s.conn.Close()
		case <-stop:
		}
	}()

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				s.writeMu.Lock()
				err := s.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
				s.writeMu.Unlock()
				if err != nil {
					s.conn.Close()
					return
				}
			}
		}
	}()

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			return err
		}
		if len(data) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(data, &ev); err != nil {
			continue
		}
		if ev.Echo != "" {
			if ev.RetCode != 0 {
				lg.Printf("[%s] 接口返回错误 code=%d %s", s.name, ev.RetCode, ev.Wording)
			}
			continue
		}
		if ev.PostType == "" {
			continue
		}
		e := ev
		go func() {
			defer func() {
				if r := recover(); r != nil {
					lg.Printf("[%s] 处理事件异常: %v", s.name, r)
				}
			}()
			h(ctx, s, &e)
		}()
	}
}
