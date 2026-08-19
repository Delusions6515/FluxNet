#!/system/bin/sh
# ============================================================
# FluxNet - 执行按钮
# 点击即 toggle: 运行中则停止, 已停止则启动
# 直接输出 CLI 日志
# ============================================================

MODDIR=${0%/*}
SCRIPTS_DIR="$MODDIR/scripts"
. "$SCRIPTS_DIR/lib.sh"
load_config

if service_running; then
  echo "FluxNet 运行中, 正在停止 ..."
  "$MODDIR/bin/fluxnet" service stop
else
  echo "FluxNet 未运行, 正在启动 ..."
  "$MODDIR/bin/fluxnet" service start
fi
