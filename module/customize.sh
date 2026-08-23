#!/sbin/sh
# ============================================================
# FluxNet - 安装脚本
# 由 Magisk / KernelSU / APatch 安装器在解压并设置默认权限后 source
# ============================================================

# 仅支持在管理器内安装
if [ "$BOOTMODE" != "true" ]; then
  abort "! 请使用 Magisk/KernelSU/APatch 管理器安装本模块"
fi

# 执行按钮 (action.sh) 最低管理器版本要求:
#   Magisk >= 27008, KernelSU >= 10670, APatch >= 11039
if [ "${APATCH:-}" = "true" ]; then
  if [ "${APATCH_VER_CODE:-0}" -lt 11039 ]; then
    abort "! 请升级 APatch 后再安装 (需要 APatch >= 11039)"
  fi
elif [ "${KSU:-}" = "true" ]; then
  if [ "${KSU_VER_CODE:-0}" -lt 10670 ]; then
    abort "! 请升级 KernelSU 后再安装 (需要 >= 10670)"
  fi
else
  if [ "${MAGISK_VER_CODE:-0}" -lt 27008 ]; then
    abort "! 请升级 Magisk 后再安装 (需要 Magisk >= 27008)"
  fi
fi

# 公共函数库
. "$MODPATH/scripts/lib.sh"

DATA_DIR=/data/adb/fluxnet
BIN_DIR=$DATA_DIR/bin
CONFIG_DIR=$DATA_DIR/config
LOCAL_CONFIG_DIR=$CONFIG_DIR/local
REMOTE_CONFIG_DIR=$CONFIG_DIR/remote
RUN_DIR=$DATA_DIR/run
LOGS_DIR=$DATA_DIR/logs

ui_print "- 初始化数据目录 $DATA_DIR"
mkdir -p "$BIN_DIR" "$CONFIG_DIR" "$LOCAL_CONFIG_DIR" "$REMOTE_CONFIG_DIR" "$RUN_DIR" "$LOGS_DIR"

# ---------- 音量键交互 ----------
GETEVENT=$(command -v getevent 2>/dev/null || echo /system/bin/getevent)
TIMEOUT_BIN=$(command -v timeout 2>/dev/null || echo "")

ask_cover_bin() {
  ui_print "- 音量+ 覆盖 / 其他键 跳过 (5 秒无按键自动跳过)"
  local line=""
  while :; do
    if [ -n "$TIMEOUT_BIN" ]; then
      line=$("$TIMEOUT_BIN" 5 "$GETEVENT" -qlc 1 2>/dev/null)
    else
      line=$("$GETEVENT" -qlc 1 2>/dev/null)
    fi
    case "$line" in
      *KEY_VOLUMEUP*DOWN*)   return 0 ;;
      *KEY_VOLUMEDOWN*DOWN*) return 1 ;;
      "")                    return 1 ;;
      *) : ;;
    esac
  done
}

# ---------- 二进制 ----------
# fluxnet: 始终从包内覆盖模块目录
ui_print "- 安装 fluxnet CLI"
[ -f "$MODPATH/bin/fluxnet" ] || abort "! fluxnet 二进制缺失"

# sing-box / atp: 首次直接复制，升级时询问
# build.sh 已将它们下载到 MODPATH/bin/ 暂存
HAS_SINGBOX=0
HAS_ATP=0
[ -f "$MODPATH/bin/sing-box" ] && HAS_SINGBOX=1
[ -f "$MODPATH/bin/atp" ] && HAS_ATP=1

if [ ! -f "$BIN_DIR/sing-box" ]; then
  if [ "$HAS_SINGBOX" = "1" ]; then
    ui_print "- 首次安装: 复制 sing-box"
    cp -f "$MODPATH/bin/sing-box" "$BIN_DIR/sing-box"
    chmod 755 "$BIN_DIR/sing-box"
  else
    ui_print "! 模块未内置 sing-box, 请稍后通过 build.sh 获取"
  fi
else
  if [ "$HAS_SINGBOX" = "1" ]; then
    ui_print "- 检测到已有 sing-box"
    if ask_cover_bin; then
      ui_print "- 已选择覆盖: 用包内版本覆盖现有 sing-box"
      cp -f "$MODPATH/bin/sing-box" "$BIN_DIR/sing-box"
      cp -f "$MODPATH/bin/kernel_version" "$BIN_DIR/kernel_version"
      chmod 755 "$BIN_DIR/sing-box"
    else
      ui_print "- 已选择跳过: 保留现有 sing-box"
    fi
  fi
fi

if [ ! -f "$BIN_DIR/atp" ]; then
  if [ "$HAS_ATP" = "1" ]; then
    ui_print "- 首次安装: 复制 atp"
    cp -f "$MODPATH/bin/atp" "$BIN_DIR/atp"
    chmod 755 "$BIN_DIR/atp"
  fi
else
  if [ "$HAS_ATP" = "1" ]; then
    ui_print "- 检测到已有 atp"
    if ask_cover_bin; then
      ui_print "- 已选择覆盖: 用包内版本覆盖现有 atp"
      cp -f "$MODPATH/bin/atp" "$BIN_DIR/atp"
      cp -f "$MODPATH/bin/atp_version" "$BIN_DIR/atp_version"
      chmod 755 "$BIN_DIR/atp"
    else
      ui_print "- 已选择跳过: 保留现有 atp"
    fi
  fi
fi

# 清理暂存二进制 (不留在模块目录)
rm -f "$MODPATH/bin/sing-box" "$MODPATH/bin/atp" 2>/dev/null

# ---------- 配置文件 (仅在不存在时写入) ----------

# fluxnet.config: 从模块模板复制，按设备 ABI 设置默认 kernel_abi
if [ ! -f "$CONFIG_DIR/fluxnet.config" ]; then
  ui_print "- 写入默认配置文件 fluxnet.config"
  cp -f "$MODPATH/config/fluxnet.config" "$CONFIG_DIR/fluxnet.config"
  ABI=$(getprop ro.product.cpu.abi 2>/dev/null)
  case "$ABI" in
    arm64-v8a|armeabi-v7a|x86_64|x86)
      sed -i "s/^kernel_abi=.*/kernel_abi=\"$ABI\"/" "$CONFIG_DIR/fluxnet.config"
      ui_print "- 已按设备架构设置 kernel_abi=$ABI"
      ;;
    *) ui_print "! 无法识别设备 ABI ($ABI), 保持默认 arm64-v8a" ;;
  esac
fi

# 名单已迁移到独立文件；旧值不保留，避免更新后仍参与运行时配置。
sed -i \
  -e '/^[[:space:]]*proxy_apps_list=/d' \
  -e '/^[[:space:]]*bypass_apps_list=/d' \
  -e '/^[[:space:]]*auto_proxy_apps_enable=/d' \
  "$CONFIG_DIR/fluxnet.config"
if ! grep -q '^[[:space:]]*auto_mode=' "$CONFIG_DIR/fluxnet.config"; then
  printf '\nauto_mode=0\n' >> "$CONFIG_DIR/fluxnet.config"
fi

# subscription.json: 默认索引
if [ ! -f "$CONFIG_DIR/subscription.json" ]; then
  ui_print "- 写入订阅索引 subscription.json"
  cat > "$CONFIG_DIR/subscription.json" << 'SUBEOF'
{
  "active": "default",
  "subscriptions": [
    {"name":"default","type":"local","filename":"default.json","url":null,"updated_at":null}
  ]
}
SUBEOF
fi

# local/default.json: 最小可用完整配置
if [ ! -f "$LOCAL_CONFIG_DIR/default.json" ]; then
  ui_print "- 写入默认完整配置 local/default.json"
  cat > "$LOCAL_CONFIG_DIR/default.json" << 'DCEOF'
{
  "log": {"level": "info"},
  "dns": {
    "servers": [
      {"tag": "local", "type": "udp", "server": "223.5.5.5"}
    ]
  },
  "inbounds": [],
  "outbounds": [
    {"tag": "direct", "type": "direct"},
  ],
  "route": {
    "rules": [],
    "final": "direct",
    "auto_detect_interface": true
  }
}
DCEOF
fi

# 构建期内置的 v2rayNG 原始清单每次模块更新都刷新；其余名单由用户维护。
cp -f "$MODPATH/config/proxy_package_name" "$CONFIG_DIR/proxy_package_name"
if [ ! -f "$CONFIG_DIR/proxy_app.txt" ]; then
  cp -f "$MODPATH/config/proxy_app.txt" "$CONFIG_DIR/proxy_app.txt"
fi
if [ ! -f "$CONFIG_DIR/bypass_app.txt" ]; then
  cp -f "$MODPATH/config/bypass_app.txt" "$CONFIG_DIR/bypass_app.txt"
fi
if [ ! -f "$CONFIG_DIR/force_proxy_app.txt" ]; then
  cp -f "$MODPATH/config/force_proxy_app.txt" "$CONFIG_DIR/force_proxy_app.txt"
fi
if [ ! -f "$CONFIG_DIR/force_bypass_app.txt" ]; then
  cp -f "$MODPATH/config/force_bypass_app.txt" "$CONFIG_DIR/force_bypass_app.txt"
fi

# 注意: tproxy.conf 安装时不复制到数据目录，由 fluxnet config apply 时按优先级读取

# ---------- 权限 ----------
set_perm_recursive "$MODPATH" 0 0 0755 0644
chmod 755 "$MODPATH/bin/fluxnet" 2>/dev/null
chmod 755 "$BIN_DIR/sing-box" 2>/dev/null
chmod 755 "$BIN_DIR/atp" 2>/dev/null
set_perm "$CONFIG_DIR/fluxnet.config" 0 0 0600 2>/dev/null
set_perm "$CONFIG_DIR/subscription.json" 0 0 0600 2>/dev/null
set_perm "$CONFIG_DIR/proxy_package_name" 0 0 0600 2>/dev/null
set_perm "$CONFIG_DIR/proxy_app.txt" 0 0 0600 2>/dev/null
set_perm "$CONFIG_DIR/bypass_app.txt" 0 0 0600 2>/dev/null
set_perm "$CONFIG_DIR/force_proxy_app.txt" 0 0 0600 2>/dev/null
set_perm "$CONFIG_DIR/force_bypass_app.txt" 0 0 0600 2>/dev/null
set_perm "$LOCAL_CONFIG_DIR/default.json" 0 0 0600 2>/dev/null
chmod ugo+x "$MODPATH"/*.sh "$MODPATH"/scripts/*.sh 2>/dev/null

ui_print "- 安装完成"
ui_print "- 重启后 FluxNet 将自动启动"
ui_print "- 管理器内点击 [执行] 可切换服务启停"
