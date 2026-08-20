#!/system/bin/sh
# ============================================================
# FluxNet - 公共函数库
# 被模块内其它脚本 source, 不单独执行
# 运行环境: busybox ash (KernelSU Standalone Mode)
# ============================================================

# ---------- 配置加载 ----------
# 模块脚本始终位于安装目录；用户可编辑设置固定在数据目录 config/。
load_config() {
  MODDIR="${MODDIR:-${0%/*}/..}"
  DATA_DIR="/data/adb/fluxnet"
  SCRIPTS_DIR="$MODDIR/scripts"
  CONFIG_DIR="$DATA_DIR/config"

  # 用户配置文件优先数据目录，不存在回退模块目录
  if [ -f "$CONFIG_DIR/fluxnet.config" ]; then
    USER_CONFIG_FILE="$CONFIG_DIR/fluxnet.config"
  else
    USER_CONFIG_FILE="$MODDIR/config/fluxnet.config"
  fi

  # shellcheck disable=SC1090
  . "$USER_CONFIG_FILE" 2>/dev/null || {
    echo "[Error] 配置文件加载失败: $USER_CONFIG_FILE"
    return 1
  }

  # 路径导出
  flux_data="${flux_data:-$DATA_DIR}"
  bin_path="${sing_box_bin_path:-${flux_data}/bin/sing-box}"
  atp_bin="${atp_bin_path:-${flux_data}/bin/atp}"
  fluxnet_bin="$MODDIR/bin/fluxnet"

  config_root="${flux_data}/config"
  local_config_path="${config_root}/local"
  remote_config_path="${config_root}/remote"
  run_path="${flux_data}/run"
  runtime_config_dir="${config_root}/run"
  runtime_config_path="${runtime_config_dir}/config.json"
  runtime_tproxy_dir="${runtime_config_dir}/tproxy"
  logs_path="${flux_data}/logs"

  subscription_file="${config_root}/subscription.json"
  inbound_template_path="$MODDIR/config/inbounds/tpl"

  force_proxy_apps_file="${config_root}/force_proxy_app.txt"
  force_bypass_apps_file="${config_root}/force_bypass_app.txt"

  # atp 配置模板: 数据目录优先, 不存在回退模块目录
  if [ -f "$CONFIG_DIR/tproxy.conf" ]; then
    tproxy_conf_path="$CONFIG_DIR/tproxy.conf"
  else
    tproxy_conf_path="$MODDIR/config/tproxy.conf"
  fi

  service_pid_file="${run_path}/sing-box.pid"
  worker_pid_file="${run_path}/worker.pid"
  config_changed_marker="${run_path}/.config-changed"
  manual_marker="${flux_data}/manual"

  # 默认值
  bin_name="${bin_name:-sing-box}"
  proxy_mode="${proxy_mode:-tun}"
  kernel_channel="${kernel_channel:-delusions6515-pre}"
  kernel_abi="${kernel_abi:-arm64-v8a}"
  autostart="${autostart:-1}"
}

# ---------- 状态检测 ----------
service_process_matches() {
  _pid="$1"
  [ -r "/proc/$_pid/cmdline" ] || return 1
  _cmdline=$(tr '\000' ' ' < "/proc/$_pid/cmdline" 2>/dev/null)
  case "$_cmdline" in
    *"$bin_path"*"run"*"-c"*) return 0 ;;
    *"$bin_path"*"-c"*) return 0 ;;
  esac
  return 1
}

service_running() {
  _service_pid=""
  if [ -f "$service_pid_file" ]; then
    _service_pid=$(cat "$service_pid_file" 2>/dev/null)
  fi
  case "$_service_pid" in
    *[!0-9]*|'') _service_pid="" ;;
  esac
  if [ -n "$_service_pid" ] \
    && kill -0 "$_service_pid" 2>/dev/null \
    && service_process_matches "$_service_pid"; then
    return 0
  fi
  rm -f "$service_pid_file"
  return 1
}

autostart_enabled() {
  [ ! -f "$manual_marker" ]
}

# ---------- tproxy 配置路径 ----------
resolve_tproxy_conf() {
  if [ -f "$CONFIG_DIR/tproxy.conf" ]; then
    echo "$CONFIG_DIR/tproxy.conf"
  else
    echo "$MODDIR/config/tproxy.conf"
  fi
}

# ---------- JSON 输出 (Shell 级别, 供 WebUI 等使用) ----------
json_escape() {
  printf '%s' "$1" | awk 'BEGIN { ORS="" }
    {
      gsub(/\\/, "\\\\")
      gsub(/"/, "\\\"")
      gsub(/\r/, "\\r")
      gsub(/\n/, "\\n")
      print
    }'
}
