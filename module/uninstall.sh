#!/system/bin/sh
# ============================================================
# FluxNet - 卸载脚本
# 停止服务并清理 iptables 规则; 用户数据 (/data/adb/fluxnet/) 保留
# ============================================================

MODDIR=${0%/*}
SCRIPTS_DIR="$MODDIR/scripts"
. "$SCRIPTS_DIR/lib.sh"
load_config

# 停止 Worker (会顺带停止 sing-box)
if [ -f "$fluxnet_bin" ]; then
  "$fluxnet_bin" worker stop >/dev/null 2>&1
  "$fluxnet_bin" service stop >/dev/null 2>&1
fi

# 兜底清理透明代理规则
if [ -f "$atp_bin" ] && [ -d "$runtime_tproxy_dir" ]; then
  sh "$atp_bin" -d "$runtime_tproxy_dir" stop >/dev/null 2>&1
fi

# 停止 inotifyd 监控
for pid in $(pidof inotifyd 2>/dev/null); do
  grep -q 'fluxnet\|sing-box' "/proc/$pid/cmdline" 2>/dev/null && kill "$pid" 2>/dev/null
done

echo "- FluxNet 服务已停止"
echo "- 数据保留在 /data/adb/fluxnet (如需彻底删除请手动删除)"
