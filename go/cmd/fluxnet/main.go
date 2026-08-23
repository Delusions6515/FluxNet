package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/Delusions6515/FluxNet/internal/app"
	"github.com/Delusions6515/FluxNet/internal/config"
	"github.com/Delusions6515/FluxNet/internal/health"
	"github.com/Delusions6515/FluxNet/internal/logs"
	"github.com/Delusions6515/FluxNet/internal/paths"
	"github.com/Delusions6515/FluxNet/internal/result"
	"github.com/Delusions6515/FluxNet/internal/service"
	"github.com/Delusions6515/FluxNet/internal/subscription"
	"github.com/Delusions6515/FluxNet/internal/worker"
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
	result.SetOperationLogger(func(operation result.Result) {
		logs.RecordOperation(layout, operation)
	})

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
  service start|stop|restart|status|logs  服务生命周期管理
  config apply|show|set              运行配置和基础设置
  subscription add|update|list|remove|switch  订阅管理
  worker start|stop                  后台 Worker
  health                             健康检查
	app-list update|show|upgrade|installed  应用名单管理
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

func cmdService(layout *paths.Layout, args []string) {
	if len(args) == 0 {
		service.Status(layout, jsonOutput)
		return
	}
	switch args[0] {
	case "start":
		service.Start(layout, jsonOutput)
	case "stop":
		service.Stop(layout, jsonOutput)
	case "restart":
		service.Restart(layout, jsonOutput)
	case "logs":
		logs.Show(layout, jsonOutput)
	case "status":
		service.Status(layout, jsonOutput)
	default:
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet service start|stop|restart|status|logs")
	}
}

func cmdConfig(layout *paths.Layout, args []string) {
	if len(args) == 0 || args[0] == "apply" {
		config.Apply(layout, jsonOutput)
		return
	}
	switch args[0] {
	case "show":
		result.Text(result.Success("config.settings", "基础设置", config.ReadSettings(layout)), jsonOutput)
	case "set":
		if len(args) != 3 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet config set <key> <value>")
			return
		}
		settings, err := config.UpdateSetting(layout, args[1], args[2])
		if err != nil {
			result.Err(jsonOutput, "config.invalid_setting", err.Error())
			return
		}
		result.Text(result.Success("config.updated", "设置已保存", settings), jsonOutput)
	default:
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet config apply|show|set <key> <value>")
	}
}

func cmdSubscription(layout *paths.Layout, args []string) {
	if len(args) == 0 {
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription add|update|list|remove|switch")
		return
	}
	switch args[0] {
	case "local":
		cmdLocalSubscription(layout, args[1:])
	case "add":
		urlOrPath := ""
		name := ""
		if len(args) > 1 {
			urlOrPath = args[1]
		}
		if len(args) > 2 {
			name = args[2]
		}
		if urlOrPath == "" {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription add <url|path> [name]")
			return
		}
		subscription.Add(layout, jsonOutput, urlOrPath, name)
	case "update":
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		subscription.Update(layout, jsonOutput, name)
	case "list":
		subscription.List(layout, jsonOutput)
	case "remove":
		if len(args) < 2 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription remove <name>")
			return
		}
		subscription.Remove(layout, jsonOutput, args[1])
	case "switch":
		if len(args) < 2 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription switch <name>")
			return
		}
		subscription.Switch(layout, jsonOutput, args[1])
	default:
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription add|update|list|remove|switch|local")
	}
}

func cmdLocalSubscription(layout *paths.Layout, args []string) {
	if len(args) == 0 {
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription local create|read|write")
		return
	}
	switch args[0] {
	case "create":
		if len(args) != 2 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription local create <name>")
			return
		}
		subscription.CreateLocal(layout, jsonOutput, args[1])
	case "read":
		if len(args) != 2 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription local read <name>")
			return
		}
		subscription.ReadLocal(layout, jsonOutput, args[1])
	case "write":
		if len(args) != 3 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription local write <name> <base64-json>")
			return
		}
		subscription.WriteLocal(layout, jsonOutput, args[1], args[2])
	default:
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet subscription local create|read|write")
	}
}

func cmdWorker(layout *paths.Layout, args []string) {
	if len(args) == 0 {
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet worker start|stop")
		return
	}
	switch args[0] {
	case "start":
		worker.Start(layout, jsonOutput)
	case "stop":
		worker.Stop(layout, jsonOutput)
	case "run":
		worker.Run(layout)
	default:
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet worker start|stop")
	}
}

func cmdAppList(layout *paths.Layout, args []string) {
	if len(args) == 0 {
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet app-list update|show|upgrade|installed|replace|force-replace")
		return
	}
	switch args[0] {
	case "update":
		app.Update(layout, jsonOutput)
	case "show":
		app.Show(layout, jsonOutput)
	case "upgrade":
		app.Upgrade(layout, jsonOutput)
	case "installed":
		app.Installed(jsonOutput)
	case "replace":
		if len(args) != 3 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet app-list replace <whitelist|blacklist> <base64-json>")
			return
		}
		app.Replace(layout, jsonOutput, args[1], args[2])
	case "force-replace":
		if len(args) != 3 {
			result.Err(jsonOutput, "usage.invalid", "用法: fluxnet app-list force-replace <proxy|bypass> <base64-json>")
			return
		}
		app.ReplaceForce(layout, jsonOutput, args[1], args[2])
	default:
		result.Err(jsonOutput, "usage.invalid", "用法: fluxnet app-list update|show|upgrade|installed|replace|force-replace")
	}
}

func cmdHealth(layout *paths.Layout, args []string) {
	health.Check(layout, jsonOutput)
}
