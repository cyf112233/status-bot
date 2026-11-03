// Package config 负责加载与校验 config.yml。
package config

import (
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

//go:embed default.yml
var defaultConfig string

// Template 是首次运行自动生成的示例配置。
var Template = defaultConfig

// Connection 描述一个 OneBot 连接。
// type=forward 时连接 url；type=reverse 时监听 port + path。
type Connection struct {
	Type    string `yaml:"type"`
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
	Token   string `yaml:"token"`
	Enabled *bool  `yaml:"enabled"`
}

func (c Connection) IsEnabled() bool { return c.Enabled == nil || *c.Enabled }

// Render 控制出图与并发保护。
type Render struct {
	Width         int `yaml:"width"`
	Concurrency   int `yaml:"concurrency"`
	CacheSeconds  int `yaml:"cache_seconds"`
	Queue         int `yaml:"queue"`
	Cooldown      int `yaml:"cooldown"`
	TopProcesses  int `yaml:"top_processes"`
	MaxDisks      int `yaml:"max_disks"`
	MaxInterfaces int `yaml:"max_interfaces"`
}

// QQList 允许写 [123456] 或 ["123456"]。
type QQList []int64

func (l *QQList) UnmarshalYAML(node *yaml.Node) error {
	if node.Tag == "!!null" {
		*l = nil
		return nil
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("users 必须是 QQ 号列表")
	}
	out := make(QQList, 0, len(node.Content))
	for _, n := range node.Content {
		if n.Kind != yaml.ScalarNode {
			return fmt.Errorf("users 中存在非法值")
		}
		v, err := strconv.ParseInt(strings.TrimSpace(n.Value), 10, 64)
		if err != nil || v <= 0 {
			return fmt.Errorf("无效的 QQ 号: %q", n.Value)
		}
		out = append(out, v)
	}
	*l = out
	return nil
}

func (l QQList) Contains(id int64) bool {
	for _, v := range l {
		if v == id {
			return true
		}
	}
	return false
}

type Config struct {
	Render      Render       `yaml:"render"`
	Commands    []string     `yaml:"commands"`
	Users       QQList       `yaml:"users"`
	Connections []Connection `yaml:"connections"`
}

// Store 支持热更新（SIGHUP），读路径无锁。
type Store struct{ p atomic.Pointer[Config] }

func NewStore(c *Config) *Store {
	s := &Store{}
	s.p.Store(c)
	return s
}

func (s *Store) Get() *Config  { return s.p.Load() }
func (s *Store) Set(c *Config) { s.p.Store(c) }

// Load 读取并规范化配置。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	r := &c.Render
	if r.Width == 0 {
		r.Width = 1080
	}
	if r.Concurrency <= 0 {
		r.Concurrency = 1
	}
	if r.CacheSeconds <= 0 {
		r.CacheSeconds = 3
	}
	if r.Queue <= 0 {
		r.Queue = 64
	}
	if r.Cooldown < 0 {
		r.Cooldown = 0
	}
	if r.TopProcesses <= 0 {
		r.TopProcesses = 10
	}
	if r.TopProcesses > 12 {
		r.TopProcesses = 12
	}
	if r.MaxDisks <= 0 {
		r.MaxDisks = 6
	}
	if r.MaxDisks > 10 {
		r.MaxDisks = 10
	}
	if r.MaxInterfaces <= 0 {
		r.MaxInterfaces = 3
	}
	if r.MaxInterfaces > 6 {
		r.MaxInterfaces = 6
	}

	seenCmd := make(map[string]struct{}, len(c.Commands))
	dst := c.Commands[:0]
	for _, cmd := range c.Commands {
		if cmd == "" {
			continue
		}
		if _, ok := seenCmd[cmd]; ok {
			continue
		}
		seenCmd[cmd] = struct{}{}
		dst = append(dst, cmd)
	}
	c.Commands = dst

	seenUser := make(map[int64]struct{}, len(c.Users))
	users := c.Users[:0]
	for _, u := range c.Users {
		if u <= 0 {
			continue
		}
		if _, ok := seenUser[u]; ok {
			continue
		}
		seenUser[u] = struct{}{}
		users = append(users, u)
	}
	c.Users = users

	for i := range c.Connections {
		conn := &c.Connections[i]
		conn.Type = strings.ToLower(strings.TrimSpace(conn.Type))
		if conn.Path == "" {
			conn.Path = "/"
		}
		if !strings.HasPrefix(conn.Path, "/") {
			conn.Path = "/" + conn.Path
		}
		if strings.TrimSpace(conn.Name) == "" {
			conn.Name = fmt.Sprintf("%s-%d", conn.Type, i+1)
		}
	}
}

func (c *Config) validate() error {
	if c.Render.Width < 720 || c.Render.Width > 2160 {
		return fmt.Errorf("render.width 需在 720-2160 之间")
	}
	names := map[string]bool{}
	reverseRoutes := map[string]string{}
	for i, conn := range c.Connections {
		if !conn.IsEnabled() {
			continue
		}
		if names[conn.Name] {
			return fmt.Errorf("连接名称重复: %s", conn.Name)
		}
		names[conn.Name] = true
		switch conn.Type {
		case "forward":
			u, err := url.Parse(conn.URL)
			if err != nil || (u.Scheme != "ws" && u.Scheme != "wss") || u.Host == "" {
				return fmt.Errorf("连接 %s: 正向连接需要合法的 ws:// 或 wss:// 地址", conn.Name)
			}
		case "reverse":
			if conn.Port < 1 || conn.Port > 65535 {
				return fmt.Errorf("连接 %s: 反向连接端口无效", conn.Name)
			}
			key := fmt.Sprintf("%d%s", conn.Port, conn.Path)
			if other, ok := reverseRoutes[key]; ok {
				return fmt.Errorf("连接 %s 与 %s 使用了相同的监听端口和路径", conn.Name, other)
			}
			reverseRoutes[key] = conn.Name
		default:
			return fmt.Errorf("连接 %d: type 只能是 forward 或 reverse", i+1)
		}
	}
	return nil
}
