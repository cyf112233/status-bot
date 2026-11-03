package main

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"statusbot/internal/config"
	"statusbot/internal/onebot"
)

type fakeRenderer struct{ calls int }

func (f *fakeRenderer) PNG(ctx context.Context) ([]byte, error) {
	f.calls++
	return []byte("img"), nil
}

type call struct {
	action string
	params map[string]any
}

type fakeSession struct {
	name  string
	calls []call
}

func (f *fakeSession) Name() string { return f.name }

func (f *fakeSession) Call(ctx context.Context, action string, params any) error {
	p, _ := params.(map[string]any)
	f.calls = append(f.calls, call{action: action, params: p})
	return nil
}

func newTestBot(commands []string, users config.QQList, cooldown int) (*Bot, *fakeRenderer) {
	store := config.NewStore(&config.Config{
		Commands: commands,
		Users:    users,
		Render:   config.Render{Cooldown: cooldown, Concurrency: 1, CacheSeconds: 3, Queue: 4},
	})
	r := &fakeRenderer{}
	return NewBot(store, r, log.New(io.Discard, "", 0)), r
}

func textEvent(id int64, user int64, text string) *onebot.Event {
	msg := `[{"type":"text","data":{"text":"` + text + `"}}]`
	return &onebot.Event{
		PostType: "message", MessageType: "group", SelfID: 1,
		UserID: onebot.ID(user), GroupID: 99, MessageID: onebot.ID(id),
		Message: []byte(msg), Time: time.Now().Unix(),
	}
}

func TestExactMatchAndImageReply(t *testing.T) {
	b, r := newTestBot([]string{"状态"}, nil, 0)
	s := &fakeSession{name: "test"}
	b.Handle(context.Background(), s, textEvent(1, 2, "状态"))

	if r.calls != 1 || len(s.calls) != 1 {
		t.Fatalf("render=%d calls=%d", r.calls, len(s.calls))
	}
	if s.calls[0].action != "send_group_msg" {
		t.Fatalf("action = %s", s.calls[0].action)
	}
	if s.calls[0].params["group_id"].(int64) != 99 {
		t.Fatalf("group_id 错误")
	}
	msg := s.calls[0].params["message"].([]map[string]any)
	data := msg[0]["data"].(map[string]any)
	file := data["file"].(string)
	if !strings.HasPrefix(file, "base64://") {
		t.Fatalf("file = %s", file)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(file, "base64://"))
	if err != nil || string(raw) != "img" {
		t.Fatalf("图片编码错误: %v %q", err, raw)
	}
}

func TestNoMatchOnExtraSpace(t *testing.T) {
	b, r := newTestBot([]string{"状态"}, nil, 0)
	s := &fakeSession{name: "test"}
	b.Handle(context.Background(), s, textEvent(1, 2, "状态 "))
	if r.calls != 0 || len(s.calls) != 0 {
		t.Fatal("带空格不应触发")
	}
}

func TestNoMatchWithAtSegment(t *testing.T) {
	b, r := newTestBot([]string{"状态"}, nil, 0)
	s := &fakeSession{name: "test"}
	ev := textEvent(1, 2, "状态")
	ev.Message = []byte(`[{"type":"at","data":{"qq":"1"}},{"type":"text","data":{"text":"状态"}}]`)
	b.Handle(context.Background(), s, ev)
	if r.calls != 0 {
		t.Fatal("@ 消息不应触发")
	}
}

func TestWhitelist(t *testing.T) {
	b, r := newTestBot([]string{"状态"}, config.QQList{100}, 0)
	s := &fakeSession{name: "test"}
	b.Handle(context.Background(), s, textEvent(1, 200, "状态"))
	if r.calls != 0 {
		t.Fatal("非白名单用户不应触发")
	}
	b.Handle(context.Background(), s, textEvent(2, 100, "状态"))
	if r.calls != 1 {
		t.Fatal("白名单用户应触发")
	}
}

func TestDedupAndCooldown(t *testing.T) {
	b, r := newTestBot([]string{"状态"}, nil, 0)
	s := &fakeSession{name: "test"}
	b.Handle(context.Background(), s, textEvent(1, 2, "状态"))
	b.Handle(context.Background(), s, textEvent(1, 2, "状态"))
	if r.calls != 1 {
		t.Fatalf("同一消息应去重，render=%d", r.calls)
	}

	b2, r2 := newTestBot([]string{"状态"}, nil, 5)
	s2 := &fakeSession{name: "test"}
	b2.Handle(context.Background(), s2, textEvent(10, 2, "状态"))
	b2.Handle(context.Background(), s2, textEvent(11, 2, "状态"))
	if r2.calls != 1 {
		t.Fatalf("冷却未生效，render=%d", r2.calls)
	}
}

func TestPrivateReply(t *testing.T) {
	b, _ := newTestBot([]string{"状态"}, nil, 0)
	s := &fakeSession{name: "test"}
	ev := textEvent(1, 2, "状态")
	ev.MessageType = "private"
	b.Handle(context.Background(), s, ev)
	if len(s.calls) != 1 || s.calls[0].action != "send_private_msg" {
		t.Fatalf("私聊回复异常: %+v", s.calls)
	}
	if s.calls[0].params["user_id"].(int64) != 2 {
		t.Fatal("user_id 错误")
	}
}
