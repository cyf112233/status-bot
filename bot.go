package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"sync"
	"time"

	"statusbot/internal/config"
	"statusbot/internal/onebot"
	"statusbot/internal/render"
)

type imageRenderer interface {
	PNG(ctx context.Context) ([]byte, error)
}

type Bot struct {
	cfg    *config.Store
	engine imageRenderer
	log    *log.Logger

	mu       sync.Mutex
	cooldown map[int64]time.Time
	seen     map[string]time.Time
}

func NewBot(cfg *config.Store, engine imageRenderer, lg *log.Logger) *Bot {
	return &Bot{
		cfg:      cfg,
		engine:   engine,
		log:      lg,
		cooldown: map[int64]time.Time{},
		seen:     map[string]time.Time{},
	}
}

func (b *Bot) Handle(ctx context.Context, sess onebot.Session, ev *onebot.Event) {
	if ev.PostType != "message" {
		return
	}
	if ev.MessageType != "group" && ev.MessageType != "private" {
		return
	}
	if ev.SelfID != 0 && ev.UserID == ev.SelfID {
		return
	}

	text, ok := onebot.Text(ev)
	if !ok {
		return
	}
	cfg := b.cfg.Get()
	if !hasCommand(cfg.Commands, text) {
		return
	}
	uid := ev.UserID.Int64()
	if len(cfg.Users) > 0 && !cfg.Users.Contains(uid) {
		return
	}
	if !b.claim(ev) {
		return
	}
	if !b.allow(uid, cfg.Render.Cooldown) {
		return
	}

	start := time.Now()
	cctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	png, err := b.engine.PNG(cctx)
	if err != nil {
		if errors.Is(err, render.ErrBusy) {
			b.sendText(cctx, sess, ev, "状态图正在生成，请稍后再试")
			return
		}
		b.log.Printf("[%s] 出图失败: %v", sess.Name(), err)
		if ctx.Err() == nil {
			b.sendText(cctx, sess, ev, "状态图生成失败，请稍后再试")
		}
		return
	}
	if err := b.sendImage(cctx, sess, ev, png); err != nil {
		b.log.Printf("[%s] 发送图片失败: %v", sess.Name(), err)
		return
	}
	b.log.Printf("[%s] 已响应 %s 消息，图片 %.1f KiB，耗时 %d ms",
		sess.Name(), ev.MessageType, float64(len(png))/1024, time.Since(start).Milliseconds())
}

func hasCommand(cmds []string, text string) bool {
	for _, c := range cmds {
		if c == text {
			return true
		}
	}
	return false
}

// claim 对同一条消息去重，避免同一事件经多条连接重复回复。
func (b *Bot) claim(ev *onebot.Event) bool {
	if ev.MessageID == 0 {
		return true
	}
	key := ev.MessageID.String() + ":" + ev.UserID.String() + ":" + ev.GroupID.String()
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	if last, ok := b.seen[key]; ok && now.Sub(last) < 20*time.Second {
		return false
	}
	if len(b.seen) > 512 {
		for k, t := range b.seen {
			if now.Sub(t) > 20*time.Second {
				delete(b.seen, k)
			}
		}
	}
	b.seen[key] = now
	return true
}

func (b *Bot) allow(uid int64, cooldown int) bool {
	if cooldown <= 0 {
		return true
	}
	now := time.Now()
	wait := time.Duration(cooldown) * time.Second
	b.mu.Lock()
	defer b.mu.Unlock()
	if last, ok := b.cooldown[uid]; ok && now.Sub(last) < wait {
		return false
	}
	if len(b.cooldown) > 1024 {
		for k, t := range b.cooldown {
			if now.Sub(t) > wait {
				delete(b.cooldown, k)
			}
		}
	}
	b.cooldown[uid] = now
	return true
}

func (b *Bot) sendImage(ctx context.Context, sess onebot.Session, ev *onebot.Event, png []byte) error {
	file := "base64://" + base64.StdEncoding.EncodeToString(png)
	msg := []map[string]any{{"type": "image", "data": map[string]any{"file": file}}}
	switch ev.MessageType {
	case "group":
		return sess.Call(ctx, "send_group_msg", map[string]any{
			"group_id": ev.GroupID.Int64(),
			"message":  msg,
		})
	case "private":
		return sess.Call(ctx, "send_private_msg", map[string]any{
			"user_id": ev.UserID.Int64(),
			"message": msg,
		})
	}
	return nil
}

func (b *Bot) sendText(ctx context.Context, sess onebot.Session, ev *onebot.Event, text string) {
	msg := []map[string]any{{"type": "text", "data": map[string]any{"text": text}}}
	switch ev.MessageType {
	case "group":
		_ = sess.Call(ctx, "send_group_msg", map[string]any{
			"group_id": ev.GroupID.Int64(),
			"message":  msg,
		})
	case "private":
		_ = sess.Call(ctx, "send_private_msg", map[string]any{
			"user_id": ev.UserID.Int64(),
			"message": msg,
		})
	}
}
