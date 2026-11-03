package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestTemplateIsValid(t *testing.T) {
	c, err := Load(writeConfig(t, Template))
	if err != nil {
		t.Fatalf("模板配置无法加载: %v", err)
	}
	if len(c.Commands) != 3 {
		t.Fatalf("commands = %d, want 3", len(c.Commands))
	}
	if c.Render.Concurrency != 1 || c.Render.CacheSeconds != 3 {
		t.Fatalf("默认渲染参数异常: %+v", c.Render)
	}
	if len(c.Users) != 0 {
		t.Fatalf("默认应为所有人可用")
	}
	if len(c.Connections) != 1 || c.Connections[0].Type != "forward" {
		t.Fatalf("默认连接异常: %+v", c.Connections)
	}
}

func TestUsersAcceptNumbersAndStrings(t *testing.T) {
	c, err := Load(writeConfig(t, `
commands: ["状态"]
users: [123456, "987654", 123456]
connections: []
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Users) != 2 || c.Users[0] != 123456 || c.Users[1] != 987654 {
		t.Fatalf("users = %v", c.Users)
	}
	if !c.Users.Contains(987654) || c.Users.Contains(1) {
		t.Fatalf("白名单判断异常")
	}
}

func TestDuplicateReverseRouteRejected(t *testing.T) {
	_, err := Load(writeConfig(t, `
commands: ["状态"]
connections:
  - type: reverse
    name: a
    port: 8080
    path: /ws
  - type: reverse
    name: b
    port: 8080
    path: /ws
`))
	if err == nil {
		t.Fatal("重复端口/路径应当报错")
	}
}

func TestReverseDefaultsAndDedup(t *testing.T) {
	c, err := Load(writeConfig(t, `
commands: ["状态", "状态", "", "查状态"]
connections:
  - type: reverse
    port: 9000
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Commands) != 2 {
		t.Fatalf("commands = %v", c.Commands)
	}
	if c.Connections[0].Path != "/" || c.Connections[0].Name != "reverse-1" {
		t.Fatalf("默认值异常: %+v", c.Connections[0])
	}
}

func TestForwardURLValidation(t *testing.T) {
	_, err := Load(writeConfig(t, `
connections:
  - type: forward
    url: http://127.0.0.1:3000
`))
	if err == nil {
		t.Fatal("非 ws 地址应报错")
	}
}
