#!/system/bin/sh
# ============================================================
# FluxNet - 开机启动 (late_start service 模式)
# 系统启动完成后启动 FluxNet Worker
# ============================================================

MODDIR=${0%/*}

# 等待系统启动完成后再拉起 Worker
(
  until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 3
  done
  "$MODDIR/bin/fluxnet" worker start &
) &
