package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
)

var (
	jsonOutput bool
	moduleDir  string
	dataDir    string
	timeoutSec int
)

func main() {
	fs := flag.NewFlagSet("fluxnet", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&jsonOutput, "json", false, "output JSON (schema=1)")
	fs.StringVar(&moduleDir, "module-dir", "", "module directory override")
	fs.StringVar(&dataDir, "data-dir", "", "data directory override")
	fs.IntVar(&timeoutSec, "timeout", 0, "command timeout in seconds")
	_ = fs.Parse(os.Args[1:])

	args := fs.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	layout := paths.New(moduleDir, dataDir)

	switch args[0] {
	case "service":
		cmdService(layout, args[1:])
	case "config":
		cmdConfig(layout, args[1:])
	case "subscription":
		cmdSubscription(layout, args[1:])
	case "worker":
		cmdWorker(layout, args[1:])
	case "health":
		cmdHealth(layout, args[1:])
	case "app-list":
		cmdAppList(layout, args[1:])
	case "version":
		cmdVersion()
	case "help", "-h", "--help":
		usage()
	default:
		result.Err(jsonOutput, "usage.invalid", "未知命令: "+args[0]+"，使用 fluxnet help 查看帮助")
		os.Exit(2)
	}
}

func usage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`FluxNet - Android 透明代理模块 CLI

用法:
  %s [--json] [--module-dir <路径>] [--data-dir <路径>] [--timeout <秒>] <命令> [参数...]

命令:
  service start|stop|restart|status  服务生命周期管理
  config apply                       组装运行配置
  subscription add|update|list|remove|switch  订阅管理
  worker start|stop                  后台 Worker
  health                             健康检查
  app-list update|show|upgrade       应用名单管理
  version                            版本信息
  help                               帮助

默认输出为人类可读文本，--json 切换 schema=1 JSON 输出。
`, exe)
}

func cmdVersion() {
	info, _ := debug.ReadBuildInfo()
	rev := "unknown"
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			rev = s.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
			break
		}
	}
	goVer := info.GoVersion
	if jsonOutput {
		result.WriteJSON(os.Stdout, result.Success("version", "FluxNet "+rev, map[string]any{
			"revision": rev,
			"go":       goVer,
		}))
	} else {
		fmt.Printf("FluxNet %s  (Go %s)\n", rev, goVer)
	}
}

// ---- Stubs for remaining subcommands (implemented in later commits) ----

func cmdService(layout *paths.Layout, args []string) {
	if len(args) == 0 {
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet service start|stop|restart|status")
		return
	}
	result.Err(jsonOutput, "not_implemented", "service "+args[0]+" 尚未实现")
}

func cmdConfig(layout *paths.Layout, args []string) {
	result.Err(jsonOutput, "not_implemented", "config 尚未实现")
}

func cmdSubscription(layout *paths.Layout, args []string) {
	result.Err(jsonOutput, "not_implemented", "subscription 尚未实现")
}

func cmdWorker(layout *paths.Layout, args []string) {
	result.Err(jsonOutput, "not_implemented", "worker 尚未实现")
}

func cmdHealth(layout *paths.Layout, args []string) {
	result.Err(jsonOutput, "not_implemented", "health 尚未实现")
}

func cmdAppList(layout *paths.Layout, args []string) {
	result.Err(jsonOutput, "not_implemented", "app-list 尚未实现")
}
