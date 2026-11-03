// Package onebot 实现 OneBot v11 的正向 / 反向 WebSocket 接入。
package onebot

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// ID 兼容数字与字符串形式的 QQ 号。
type ID int64

func (i *ID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" || s == "null" {
		*i = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*i = ID(v)
	return nil
}

func (i ID) Int64() int64 { return int64(i) }

func (i ID) String() string { return strconv.FormatInt(int64(i), 10) }

type Event struct {
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	SubType     string          `json:"sub_type"`
	SelfID      ID              `json:"self_id"`
	UserID      ID              `json:"user_id"`
	GroupID     ID              `json:"group_id"`
	MessageID   ID              `json:"message_id"`
	Message     json.RawMessage `json:"message"`
	RawMessage  string          `json:"raw_message"`
	Time        int64           `json:"time"`
	Echo        string          `json:"echo"`
	RetCode     int             `json:"ret_code"`
	Status      string          `json:"status"`
	Wording     string          `json:"wording"`
	Data        json.RawMessage `json:"data"`
}

type segment struct {
	Type string                     `json:"type"`
	Data map[string]json.RawMessage `json:"data"`
}

// Text 只接受纯文本消息（所有消息段均为 text），并原样返回拼接结果。
// 含图片、@、表情等任何非 text 段都不算“绝对匹配”。
func Text(ev *Event) (string, bool) {
	if len(ev.Message) == 0 || string(ev.Message) == "null" {
		if ev.RawMessage != "" && !strings.Contains(ev.RawMessage, "[CQ:") {
			return ev.RawMessage, true
		}
		return "", false
	}
	var segs []segment
	if err := json.Unmarshal(ev.Message, &segs); err == nil {
		if len(segs) == 0 {
			return "", false
		}
		var b strings.Builder
		for _, s := range segs {
			if s.Type != "text" {
				return "", false
			}
			raw, ok := s.Data["text"]
			if !ok {
				return "", false
			}
			var t string
			if err := json.Unmarshal(raw, &t); err != nil {
				return "", false
			}
			b.WriteString(t)
		}
		return b.String(), true
	}
	var s string
	if err := json.Unmarshal(ev.Message, &s); err == nil {
		if strings.Contains(s, "[CQ:") {
			return "", false
		}
		return s, true
	}
	return "", false
}

type Handler func(ctx context.Context, s Session, ev *Event)

type Session interface {
	Name() string
	Call(ctx context.Context, action string, params any) error
}
