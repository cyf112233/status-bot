package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"statusbot/internal/config"
	"statusbot/internal/onebot"
	"statusbot/internal/render"
	"statusbot/internal/sysinfo"
)

const version = "1.0.0"

func main() {
	configFlag := flag.String("config", "", "配置文件路径（默认程序目录或当前目录下的 config.yml）")
	renderFlag := flag.String("render", "", "只生成一张示例状态图到指定文件后退出")
	showVersion := flag.Bool("version", false, "打印版本号")
	flag.Parse()

	if *showVersion {
		fmt.Println("statusbot " + version)
		return
	}

	path := resolveConfigPath(*configFlag)
	cfg, err := config.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			if werr := os.WriteFile(path, []byte(config.Template), 0o600); werr == nil {
				fmt.Printf("未找到配置文件，已生成示例：%s\n", path)
				if *renderFlag == "" {
					fmt.Println("请填写 connections 后重新启动。")
					return
				}
				cfg, err = config.Load(path)
			}
		}
		if err != nil {
			log.Fatalf("加载配置失败: %v", err)
		}
	}

	lg := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)
	lg.SetPrefix("[statusbot] ")
	if err := render.Preload(); err != nil {
		lg.Fatalf("字体加载失败: %v", err)
	}

	if *renderFlag != "" {
		sampler := sysinfo.New(time.Second, cfg.Render.TopProcesses, cfg.Render.MaxDisks)
		start := time.Now()
		snap := sampler.SampleOnce(context.Background())
		png, err := render.Render(snap, cfg.Render)
		if err != nil {
			lg.Fatalf("渲染失败: %v", err)
		}
		if err := os.WriteFile(*renderFlag, png, 0o644); err != nil {
			lg.Fatalf("写入文件失败: %v", err)
		}
		lg.Printf("已生成 %s（%.1f KiB，耗时 %d ms）", *renderFlag, float64(len(png))/1024, time.Since(start).Milliseconds())
		return
	}

	lg.Printf("statusbot %s 启动，配置：%s", version, path)
	logConfigSummary(lg, cfg)

	store := config.NewStore(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sampler := sysinfo.New(time.Second, cfg.Render.TopProcesses, cfg.Render.MaxDisks)
	sampler.Start(ctx)
	engine := render.NewEngine(store, sampler)
	bot := NewBot(store, engine, lg)

	var wg sync.WaitGroup
	forwardCount := 0
	for _, c := range cfg.Connections {
		if c.Type != "forward" || !c.IsEnabled() {
			continue
		}
		forwardCount++
		conn := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			onebot.RunForward(ctx, conn, bot.Handle, lg)
		}()
	}
	if err := onebot.StartReverses(ctx, cfg.Connections, bot.Handle, lg); err != nil {
		lg.Printf("反向连接启动失败: %v", err)
	}
	if forwardCount == 0 && len(cfg.Connections) == 0 {
		lg.Printf("提示：配置中没有连接，程序将保持空转")
	}

	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		sig := <-sigs
		if sig == syscall.SIGHUP {
			newCfg, err := config.Load(path)
			if err != nil {
				lg.Printf("热加载失败，继续使用旧配置: %v", err)
				continue
			}
			store.Set(newCfg)
			engine.Reset()
			lg.Printf("配置已热加载（命令 / 白名单 / 渲染缓存生效；连接与并发数变更需重启）")
			continue
		}
		lg.Printf("收到 %v，正在退出", sig)
		cancel()
		break
	}
	wg.Wait()
	time.Sleep(150 * time.Millisecond)
	lg.Printf("已退出")
}

func logConfigSummary(lg *log.Logger, cfg *config.Config) {
	forward, reverse := 0, 0
	for _, c := range cfg.Connections {
		if !c.IsEnabled() {
			continue
		}
		switch c.Type {
		case "forward":
			forward++
		case "reverse":
			reverse++
		}
	}
	users := "所有人"
	if len(cfg.Users) > 0 {
		users = fmt.Sprintf("%d 个白名单用户", len(cfg.Users))
	}
	lg.Printf("连接：%d 正向 / %d 反向；命令 %d 条；可用：%s；渲染并发 %d，缓存 %ds",
		forward, reverse, len(cfg.Commands), users, cfg.Render.Concurrency, cfg.Render.CacheSeconds)
}

func resolveConfigPath(p string) string {
	if p != "" {
		if filepath.IsAbs(p) {
			return p
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
		if exe, err := os.Executable(); err == nil {
			cand := filepath.Join(filepath.Dir(exe), p)
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
			return cand
		}
		return p
	}
	// 默认优先程序同目录，其次当前目录
	if exe, err := os.Executable(); err == nil {
		cand := filepath.Join(filepath.Dir(exe), "config.yml")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	if _, err := os.Stat("config.yml"); err == nil {
		return "config.yml"
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "config.yml")
	}
	return "config.yml"
}
