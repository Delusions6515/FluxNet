#!/system/bin/sh
# FluxNet WebUI command gateway. Keep this list explicit: the WebView never
# receives a free-form root shell channel.

MODDIR=${MODDIR:-${0%/*}/..}
FLUXNET="$MODDIR/bin/fluxnet"

[ -x "$FLUXNET" ] || {
  echo '{"schema":1,"ok":false,"code":"webui.binary_missing","message":"FluxNet CLI 不存在"}'
  exit 1
}

command=${1:-}

case "$command" in
  service-status)
    exec "$FLUXNET" --json service status
    ;;
  service-start)
    exec "$FLUXNET" --json service request start
    ;;
  service-stop)
    exec "$FLUXNET" --json service request stop
    ;;
  service-restart)
    exec "$FLUXNET" --json service request restart
    ;;
  health)
    exec "$FLUXNET" --json health
    ;;
  logs)
    exec "$FLUXNET" --json service logs
    ;;
  config-show)
    exec "$FLUXNET" --json config show
    ;;
  config-set)
    [ "$#" = 3 ] || exit 2
    exec "$FLUXNET" --json config set "$2" "$3"
    ;;
  config-inbound-read)
    [ "$#" = 2 ] || exit 2
    exec "$FLUXNET" --json config inbound read "$2"
    ;;
  config-inbound-write)
    [ "$#" = 3 ] || exit 2
    exec "$FLUXNET" --json config inbound write "$2" "$3"
    ;;
  config-tproxy-read)
    [ "$#" = 1 ] || exit 2
    exec "$FLUXNET" --json config tproxy read
    ;;
  config-tproxy-write)
    [ "$#" = 2 ] || exit 2
    exec "$FLUXNET" --json config tproxy write "$2"
    ;;
  app-list-replace)
    [ "$#" = 3 ] || exit 2
    exec "$FLUXNET" --json app-list replace "$2" "$3"
    ;;
  app-list-force-replace)
    [ "$#" = 3 ] || exit 2
    exec "$FLUXNET" --json app-list force-replace "$2" "$3"
    ;;
  app-list-upgrade)
    [ "$#" = 1 ] || exit 2
    exec "$FLUXNET" --json app-list upgrade
    ;;
  app-list-catalog)
    [ "$#" = 1 ] || exit 2
    exec "$FLUXNET" --json app-list catalog
    ;;
  kernel-status)
    [ "$#" = 1 ] || exit 2
    exec "$FLUXNET" --json kernel status
    ;;
  kernel-set-channel)
    [ "$#" = 3 ] || exit 2
    exec "$FLUXNET" --json kernel set-channel "$2" "$3"
    ;;
  kernel-upgrade)
    [ "$#" = 1 ] || exit 2
    exec "$FLUXNET" --json kernel upgrade
    ;;
  kernel-verify)
    [ "$#" = 1 ] || exit 2
    exec "$FLUXNET" --json kernel verify
    ;;
  subscription-list)
    exec "$FLUXNET" --json subscription list
    ;;
  subscription-add-remote)
    [ "$#" = 3 ] || exit 2
    name=$(printf '%s' "$2" | base64 -d) || exit 2
    url=$(printf '%s' "$3" | base64 -d) || exit 2
    exec "$FLUXNET" --json subscription add "$url" "$name"
    ;;
  subscription-update|subscription-switch|subscription-remove|local-create|local-read)
    [ "$#" = 2 ] || exit 2
    case "$command" in
      subscription-update) exec "$FLUXNET" --json subscription update "$2" ;;
      subscription-switch) exec "$FLUXNET" --json subscription switch "$2" ;;
      subscription-remove) exec "$FLUXNET" --json subscription remove "$2" ;;
      local-create) exec "$FLUXNET" --json subscription local create "$2" ;;
      local-read) exec "$FLUXNET" --json subscription local read "$2" ;;
    esac
    ;;
  local-write)
    [ "$#" = 3 ] || exit 2
    exec "$FLUXNET" --json subscription local write "$2" "$3"
    ;;
  *)
    echo '{"schema":1,"ok":false,"code":"webui.invalid_command","message":"不支持的 WebUI 命令"}'
    exit 2
    ;;
esac
