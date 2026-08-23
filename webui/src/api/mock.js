const state = {
  service: { pid: 2481, running: true, mode: "tun", uptime_ms: 3840000 },
  health: {
    process_alive: true,
    api_reachable: true,
    proxy_rules_active: false,
    mode: "tun",
    pid: 2481,
  },
  settings: {
    autostart: true,
    proxy_mode: "tun",
    app_proxy_enable: false,
    app_proxy_mode: "blacklist",
    auto_mode: false,
    tun_stack: "gvisor",
    auto_redirect: false,
    tun_forward: false,
    proxy_apps: [],
    bypass_apps: [],
    force_proxy_apps: [],
    force_bypass_apps: [],
  },
  subscriptions: {
    active: "default",
    subscriptions: [
      {
        name: "default",
        type: "local",
        filename: "default.json",
        updated_at: "2026-08-21T00:00:00Z",
      },
    ],
  },
  local: {
    default:
      '{\n  "inbounds": [],\n  "outbounds": [{ "type": "direct", "tag": "direct" }]\n}\n',
  },
  inbounds: {
    tun: '{\n  "type": "tun",\n  "tag": "tun-in"\n}\n',
    tproxy: '{\n  "type": "tproxy",\n  "tag": "tproxy-in"\n}\n',
    redirect: '{\n  "type": "redirect",\n  "tag": "redirect-in"\n}\n',
    ebpf: '{\n  "type": "ebpf",\n  "tag": "ebpf-in"\n}\n',
  },
  tproxy: "PROXY_TCP_PORT=1536\n",
};

const result = (code, message, data) =>
  JSON.stringify({ schema: 1, ok: true, code, message, data });

export function mockGateway(command, args = []) {
  switch (command) {
    case "service-status":
      return result("service.status", "服务状态", state.service);
    case "health":
      return result("health.check", "健康检查", state.health);
    case "config-show":
      return result("config.settings", "基础设置", state.settings);
    case "subscription-list":
      return result("subscription.list", "订阅列表", state.subscriptions);
    case "logs":
      return result("logs.list", "日志列表", {
        entries: [
          {
            timestamp: "2026-08-21T12:00:00Z",
            level: "info",
            component: "service",
            event: "started",
            result: "ok",
            message: "sing-box is running",
          },
        ],
      });
    case "service-start":
    case "service-restart":
      state.service.running = true;
      state.service.pid = 2481;
      state.health.process_alive = true;
      return result("service.started", "服务已启动", state.service);
    case "service-stop":
      state.service.running = false;
      state.service.pid = 0;
      state.health.process_alive = false;
      return result("service.stopped", "服务已停止");
    case "config-set":
      state.settings[args[0]] =
        args[1] === "1" ? true : args[1] === "0" ? false : args[1];
      if (args[0] === "proxy_mode") {
        state.service.mode = args[1];
        state.health.mode = args[1];
      }
      return result("config.updated", "设置已保存", state.settings);
    case "app-list-replace": {
      const key = args[0] === "proxy" ? "proxy_apps" : "bypass_apps";
      state.settings[key] = JSON.parse(atob(args[1]));
      return result("app.replaced", "应用名单已保存", state.settings);
    }
    case "app-list-force-replace": {
      const key =
        args[0] === "proxy" ? "force_proxy_apps" : "force_bypass_apps";
      state.settings[key] = JSON.parse(atob(args[1]));
      return result("app.force_replaced", "强制应用名单已保存", state.settings);
    }
    case "app-list-upgrade":
      return result("app.upgraded", "预置名单已更新");
    case "app-list-catalog":
      return result("app.catalog", "预置名单", {
        packages: ["com.google.android.youtube", "org.telegram.messenger"],
      });
    case "config-inbound-read":
      return result("config.inbound", "用户入站", {
        content: state.inbounds[args[0]],
      });
    case "config-inbound-write":
      state.inbounds[args[0]] = atob(args[1]);
      return result("config.inbound_written", "用户入站已保存");
    case "config-tproxy-read":
      return result("config.tproxy", "用户 tproxy.conf", {
        content: state.tproxy,
      });
    case "config-tproxy-write":
      state.tproxy = atob(args[0]);
      return result("config.tproxy_written", "用户 tproxy.conf 已保存");
    case "local-create": {
      const name = args[0];
      state.subscriptions.subscriptions.push({
        name,
        type: "local",
        filename: `${name}.json`,
      });
      state.local[name] = '{\n  "inbounds": [],\n  "outbounds": []\n}\n';
      return result("subscription.local_created", `本地订阅已创建: ${name}`, {
        name,
        content: state.local[name],
      });
    }
    case "local-read":
      return result("subscription.local_read", "本地订阅", {
        name: args[0],
        content: state.local[args[0]],
      });
    case "local-write":
      state.local[args[0]] = atob(args[1]);
      return result("subscription.local_written", "本地订阅已保存");
    case "subscription-switch":
      state.subscriptions.active = args[0];
      return result("subscription.switched", "订阅已切换");
    case "subscription-remove":
      state.subscriptions.subscriptions =
        state.subscriptions.subscriptions.filter(
          (item) => item.name !== args[0],
        );
      return result("subscription.removed", "订阅已删除");
    case "subscription-update":
      return result("subscription.updated", "订阅已更新");
    case "subscription-add-remote": {
      const name = atob(args[0]);
      state.subscriptions.subscriptions.push({
        name,
        type: "remote",
        filename: `${name}.json`,
        url: atob(args[1]),
      });
      return result("subscription.added", "订阅已添加");
    }
    default:
      return JSON.stringify({
        schema: 1,
        ok: false,
        code: "webui.invalid_command",
        message: "不支持的 WebUI 命令",
      });
  }
}

export const MOCK_PACKAGES = [
  { packageName: "com.google.android.youtube", appLabel: "YouTube" },
  { packageName: "org.telegram.messenger", appLabel: "Telegram" },
  { packageName: "com.twitter.android", appLabel: "X" },
  ...Array.from({ length: 45 }, (_, index) => ({
    packageName: `com.example.app${String(index + 1).padStart(2, "0")}`,
    appLabel: `示例应用 ${index + 1}`,
  })),
];
