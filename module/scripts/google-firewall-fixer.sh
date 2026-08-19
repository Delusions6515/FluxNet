#!/system/bin/sh
# Derived from https://github.com/CHIZI-0618/ColorOS-Google-Firewall-Fixer
# (AGPL-3.0), modified 2026-08-14 for this module's lifecycle.

SCRIPTS_DIR=$(dirname "$0")
. "$SCRIPTS_DIR/lib.sh"
load_config

firewall_log="$run_path/google-firewall-fixer.log"
firewall_chains="fw_INPUT fw_OUTPUT fw_OUTPUT_oplus_dns zte_fw_gms"

log_fix() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$1" >>"$firewall_log"
}

remove_block_rules() { # $1=iptables command, $2=chain
  _cmd=$1
  _chain=$2
  _removed=0

  command -v "$_cmd" >/dev/null 2>&1 || return 0
  while :; do
    _line=$("$_cmd" -t filter -nvL "$_chain" --line-numbers 2>/dev/null \
      | awk '$1 ~ /^[0-9]+$/ && /REJECT|DROP/ { print $1; exit }')
    [ -n "$_line" ] || break
    "$_cmd" -t filter -D "$_chain" "$_line" 2>/dev/null || break
    _removed=$((_removed + 1))
  done
  [ "$_removed" -gt 0 ] && log_fix "$_cmd: $_chain 已删除 $_removed 条 REJECT/DROP 规则"
}

apply_fix() {
  mkdir -p "$run_path" || return 1
  if [ -f $firewall_log ]; then
    mv -f $firewall_log "$firewall_log.bak"
  fi
  log_fix "开始清理 ColorOS/RedMagic Google 服务拦截规则"
  for _chain in $firewall_chains; do
    remove_block_rules iptables "$_chain"
    remove_block_rules ip6tables "$_chain"
  done
  log_fix "清理完成"
}

case "$1" in
  apply) apply_fix ;;
  *) echo "用法: $0 apply" >&2; exit 2 ;;
esac
