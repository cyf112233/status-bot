package onebot

import (
	"encoding/json"
	"testing"
)

func TestTextStrict(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
		ok   bool
	}{
		{"single", `{"message":[{"type":"text","data":{"text":"状态"}}]}`, "状态", true},
		{"multi", `{"message":[{"type":"text","data":{"text":"查"}},{"type":"text","data":{"text":"状态"}}]}`, "查状态", true},
		{"at segment", `{"message":[{"type":"at","data":{"qq":"1"}},{"type":"text","data":{"text":"状态"}}]}`, "", false},
		{"image", `{"message":[{"type":"image","data":{"file":"x"}},{"type":"text","data":{"text":"状态"}}]}`, "", false},
		{"plain string", `{"message":"状态"}`, "状态", true},
		{"cq string", `{"message":"[CQ:at,qq=1] 状态"}`, "", false},
		{"raw fallback", `{"raw_message":"状态"}`, "状态", true},
		{"empty", `{"message":[]}`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ev Event
			if err := json.Unmarshal([]byte(c.body), &ev); err != nil {
				t.Fatal(err)
			}
			got, ok := Text(&ev)
			if got != c.want || ok != c.ok {
				t.Fatalf("Text() = %q,%v want %q,%v", got, ok, c.want, c.ok)
			}
		})
	}
}

func TestFlexibleID(t *testing.T) {
	var ev Event
	if err := json.Unmarshal([]byte(`{"user_id":"123456789","group_id":42}`), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.UserID.Int64() != 123456789 || ev.GroupID.Int64() != 42 {
		t.Fatalf("ID 解析失败: %d %d", ev.UserID, ev.GroupID)
	}
}
