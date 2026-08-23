#!/bin/bash
# ============================================================
# FluxNet - 模块构建脚本
# Go 交叉编译 + 获取 sing-box/atp + 打包 zip
#
# 用法:
#   ./build.sh                  # 默认 arm64-v8a / delusions6515-pre
#   ./build.sh 1.0.0            # 指定版本名覆盖 tag
#   ./build.sh 1.0.0 out.zip    # 指定版本名和输出路径
#
# 环境变量:
#   TARGET_ABI        目标 ABI: arm64-v8a (默认且唯一支持)
#   KERNEL_CHANNEL    内核渠道: delusions6515-pre(默认)|delusions6515-stable|
#                               ref1nd-pre|ref1nd-stable|official-stable|official-pre
#   OUT_DIR           输出目录 (默认 ./build)
#   SKIP_VERSION_CHECK 设为 1 跳过缓存版本对比 (强制重新下载)
# ============================================================
set -euo pipefail

REPO_DIR=$(cd "$(dirname "$0")" && pwd)
MODULE_DIR="$REPO_DIR/module"
GO_DIR="$REPO_DIR/go"
WEBUI_DIR="$REPO_DIR/webui"
OUT_DIR="${OUT_DIR:-$REPO_DIR/build}"
TARGET_ABI="${TARGET_ABI:-arm64-v8a}"
KERNEL_CHANNEL="${KERNEL_CHANNEL:-delusions6515-pre}"
VERSION="${1:-}"
OUT_ZIP="${2:-}"

info() { echo "[*] $1"; }
warn() { echo "[!] $1"; }
die()  { echo "[Error] $1"; exit 1; }

download() {
  curl -fsSL --connect-timeout 10 --max-time 600 --retry 3 --retry-delay 2 "$1" -o "$2"
}

api_get() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl -fsSL --max-time 30 -H "Authorization: Bearer $GITHUB_TOKEN" "https://api.github.com$1" 2>/dev/null
  elif command -v gh >/dev/null 2>&1; then
    gh api "$1" 2>/dev/null
  else
    curl -fsSL --max-time 30 "https://api.github.com$1" 2>/dev/null
  fi
}

# ---------- ABI 映射 ----------
case "$TARGET_ABI" in
  arm64-v8a) SINGBOX_ARCH="arm64"; GOARCH="arm64"; GOARM="" ;;
  *) die "不支持的 TARGET_ABI: $TARGET_ABI" ;;
esac
info "目标: $TARGET_ABI (sing-box: $SINGBOX_ARCH, go: $GOARCH${GOARM:+ ARMv$GOARM})  渠道: $KERNEL_CHANNEL"

# ---------- 0. 构建模块 WebUI ----------
VITE_BIN="$WEBUI_DIR/node_modules/.bin/vite"
[ -x "$VITE_BIN" ] || die "WebUI 依赖未安装，请先执行 pnpm --dir webui install --frozen-lockfile"
info "构建 WebUI ..."
(cd "$WEBUI_DIR" && "$VITE_BIN" build)
[ -f "$WEBUI_DIR/dist/index.html" ] || die "WebUI 构建未生成 index.html"

# ---------- 渠道 -> 仓库 ----------
case "$KERNEL_CHANNEL" in
  delusions6515-*)  KERNEL_REPO="Delusions6515/sing-box-releases" ;;
  ref1nd-*)         KERNEL_REPO="reF1nd/sing-box-releases" ;;
  official-*)       KERNEL_REPO="SagerNet/sing-box" ;;
  *) die "不支持的 KERNEL_CHANNEL: $KERNEL_CHANNEL" ;;
esac
case "$KERNEL_CHANNEL" in
  *-pre)     WANT_PRE=1 ;;
  *-stable)  WANT_PRE=0 ;;
esac

# ---------- 获取最新版本号 ----------
latest_tag() {
  local repo="$1" want_pre="$2" tag="" url=""
  if [ "$want_pre" = "0" ]; then
    url=$(curl -sI -o /dev/null -w '%{redirect_url}' --max-time 30 \
      "https://github.com/$repo/releases/latest" 2>/dev/null)
    tag=$(basename "$url" 2>/dev/null)
    [ "$tag" != "latest" ] && [ -n "$tag" ] && { echo "$tag"; return 0; }
    tag=$(api_get "/repos/$repo/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
  else
    tag=$(api_get "/repos/$repo/releases?per_page=20" \
      | grep -oE '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"|"prerelease"[[:space:]]*:[[:space:]]*(true|false)' \
      | while read -r _line; do
          case "$_line" in
            *tag_name*) _t=$(printf '%s' "$_line" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p') ;;
            *prerelease*true*) [ -n "$_t" ] && { echo "$_t"; break; } ;;
          esac
        done)
  fi
  echo "$tag"
}

TAG=$(latest_tag "$KERNEL_REPO" "$WANT_PRE")
[ -n "$TAG" ] || die "无法获取 $KERNEL_CHANNEL 的最新版本号"
KERNEL_VER=${TAG#v}
info "内核版本: $KERNEL_VER"

# ---------- 持久化缓存目录 (由 GitHub Actions cache 保留) ----------
CACHE_DIR="$OUT_DIR/cache"
BIN_DIR="$CACHE_DIR/bin"
mkdir -p "$BIN_DIR"

# ---------- 1. 编译 Go ----------
info "编译 fluxnet (GOOS=android GOARCH=$GOARCH${GOARM:+ GOARM=$GOARM} CGO_ENABLED=0) ..."
(
  cd "$GO_DIR"
  export CGO_ENABLED=0 GOOS=android GOARCH="$GOARCH"
  [ -n "$GOARM" ] && export GOARM="$GOARM"
  go build -ldflags="-s -w" -o "$MODULE_DIR/bin/fluxnet" ./cmd/fluxnet
)
info "fluxnet: $MODULE_DIR/bin/fluxnet"

# ---------- 2. 获取 v2rayNG 代理应用名单 (带版本缓存) ----------
# 用户设备上的数据目录副本由安装脚本保留，模块内置清单随构建版本刷新。
PROXY_PACKAGE_LIST_URL="https://raw.githubusercontent.com/2dust/v2rayNG/master/V2rayNG/app/src/main/assets/proxy_package_name"
PROXY_PACKAGE_LIST="$MODULE_DIR/config/proxy_package_name"
PROXY_PACKAGE_CACHE="$CACHE_DIR/proxy_package_name"
PROXY_PACKAGE_VER_FILE="$CACHE_DIR/proxy_package_name_version"
PROXY_PACKAGE_VER=$(api_get "/repos/2dust/v2rayNG/commits/master" \
  | sed -n 's/.*"sha"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
PROXY_PACKAGE_VER="${PROXY_PACKAGE_VER:0:7}"
current_proxy_package_ver=""
[ -f "$PROXY_PACKAGE_VER_FILE" ] && current_proxy_package_ver=$(cat "$PROXY_PACKAGE_VER_FILE")

need_proxy_package=0
if [ ! -f "$PROXY_PACKAGE_CACHE" ]; then
  need_proxy_package=1
elif [ "${SKIP_VERSION_CHECK:-0}" != "1" ] \
  && [ -n "$PROXY_PACKAGE_VER" ] \
  && [ "$current_proxy_package_ver" != "$PROXY_PACKAGE_VER" ]; then
  warn "版本检查: v2rayNG 清单过旧 (缓存=${current_proxy_package_ver:-无}, 最新=$PROXY_PACKAGE_VER), 重新下载"
  need_proxy_package=1
fi

if [ "$need_proxy_package" = "1" ]; then
  info "下载 v2rayNG 代理应用名单 ..."
  download "$PROXY_PACKAGE_LIST_URL" "$PROXY_PACKAGE_CACHE.tmp" \
    || die "下载 v2rayNG 代理应用名单失败"
  if ! grep -qE '^[[:alnum:]_]+(\.[[:alnum:]_]+)+$' "$PROXY_PACKAGE_CACHE.tmp"; then
    rm -f "$PROXY_PACKAGE_CACHE.tmp"
    die "v2rayNG 代理应用名单为空或格式无效"
  fi
  mv -f "$PROXY_PACKAGE_CACHE.tmp" "$PROXY_PACKAGE_CACHE"
  [ -n "$PROXY_PACKAGE_VER" ] && echo "$PROXY_PACKAGE_VER" > "$PROXY_PACKAGE_VER_FILE"
  info "v2rayNG 清单: ${PROXY_PACKAGE_VER:-已获取} (已缓存)"
else
  info "v2rayNG 清单: 使用缓存 ${current_proxy_package_ver:-无版本号}"
fi
cp -f "$PROXY_PACKAGE_CACHE" "$PROXY_PACKAGE_LIST"

# ---------- 3. 构建暂存 ----------
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
cp -r "$MODULE_DIR/." "$STAGE/"
find "$STAGE" -name .DS_Store -delete
STAGE_BIN="$STAGE/bin"
mkdir -p "$STAGE/webroot"
cp -r "$WEBUI_DIR/dist/." "$STAGE/webroot/"
[ -f "$STAGE/webroot/index.html" ] || die "暂存 WebUI 缺少 index.html"

# ---------- 3. 获取 sing-box (带版本缓存) ----------
KERNEL_BIN="$BIN_DIR/sing-box"
KERNEL_VER_FILE="$BIN_DIR/kernel_version"
current_kernel=""
[ -f "$KERNEL_VER_FILE" ] && current_kernel=$(cat "$KERNEL_VER_FILE")

need_kernel=0
if [ ! -f "$KERNEL_BIN" ]; then
  need_kernel=1
elif [ "${SKIP_VERSION_CHECK:-0}" != "1" ] && [ "$current_kernel" != "$KERNEL_VER" ]; then
  warn "版本检查: 内核过旧 (缓存=${current_kernel:-无}, 最新=$KERNEL_VER), 重新下载"
  need_kernel=1
fi

if [ "$need_kernel" = "1" ]; then
  ASSET="sing-box-${KERNEL_VER}-android-${SINGBOX_ARCH}.tar.gz"
  URL="https://github.com/$KERNEL_REPO/releases/download/$TAG/$ASSET"
  TMP=$(mktemp -d)
  info "下载 sing-box $KERNEL_VER ..."
  download "$URL" "$TMP/$ASSET" || die "下载 sing-box 失败: $URL"
  tar -xzf "$TMP/$ASSET" -C "$TMP"
  NEW_BIN=$(find "$TMP" -type f -name sing-box | head -n 1)
  [ -n "$NEW_BIN" ] || die "解包后未找到 sing-box"
  cp -f "$NEW_BIN" "$KERNEL_BIN"
  chmod 755 "$KERNEL_BIN"
  echo "$KERNEL_VER" > "$KERNEL_VER_FILE"
  rm -rf "$TMP"
  info "sing-box: $KERNEL_VER (已缓存)"
else
  info "sing-box: 使用缓存 $KERNEL_VER"
fi

# ---------- 4. 获取 atp (带版本缓存) ----------
ATP_SRC="https://raw.githubusercontent.com/CHIZI-0618/AndroidTProxyShell/main/tproxy.sh"
ATP_BIN="$BIN_DIR/atp"
ATP_VER_FILE="$BIN_DIR/atp_version"
ATP_VER=""
current_atp=""
[ -f "$ATP_VER_FILE" ] && current_atp=$(cat "$ATP_VER_FILE")
ATP_VER=$(api_get "/repos/CHIZI-0618/AndroidTProxyShell/commits/main" \
  | sed -n 's/.*"sha"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
ATP_VER="${ATP_VER:0:7}"

need_atp=0
if [ ! -f "$ATP_BIN" ]; then
  need_atp=1
elif [ "${SKIP_VERSION_CHECK:-0}" != "1" ] && [ -n "$ATP_VER" ] && [ "$current_atp" != "$ATP_VER" ]; then
  warn "版本检查: atp 过旧 (缓存=${current_atp:-无}, 最新=$ATP_VER), 重新下载"
  need_atp=1
fi

if [ "$need_atp" = "1" ]; then
  info "下载 AndroidTProxyShell (atp) ..."
  if download "$ATP_SRC" "$ATP_BIN.tmp"; then
    mv -f "$ATP_BIN.tmp" "$ATP_BIN"
    chmod 755 "$ATP_BIN"
    [ -n "$ATP_VER" ] && echo "$ATP_VER" > "$ATP_VER_FILE"
    info "atp: ${ATP_VER:-已获取} (已缓存)"
  else
    rm -f "$ATP_BIN.tmp"
    warn "atp 下载失败, 跳过"
  fi
else
  info "atp: 使用缓存 ${current_atp:-无版本号}"
fi

# ---------- 5. META-INF (缓存复用) ----------
META_BIN="$BIN_DIR/update-binary"
if [ -f "$META_BIN" ]; then
  info "META-INF: 使用缓存"
else
  info "下载 module_installer.sh ..."
  if download "https://raw.githubusercontent.com/topjohnwu/Magisk/master/scripts/module_installer.sh" \
    "$META_BIN"; then
    info "META-INF: 已缓存"
  else
    warn "META-INF 获取失败, 跳过"
  fi
fi

# ---------- 同步缓存到暂存 ----------
if [ -f "$META_BIN" ]; then
  mkdir -p "$STAGE/META-INF/com/google/android"
  cp -f "$META_BIN" "$STAGE/META-INF/com/google/android/update-binary"
  printf '#MAGISK\n' > "$STAGE/META-INF/com/google/android/updater-script"
fi
cp -rf "$BIN_DIR/." "$STAGE_BIN/"
chmod 755 "$STAGE_BIN/sing-box" "$STAGE_BIN/atp" 2>/dev/null
[ -e "$STAGE_BIN/sing-box" ] || die "缺少 sing-box 二进制"

# ---------- 6. 版本号 ----------
VER_NAME="${VERSION:-}"
if [ -z "$VER_NAME" ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  VER_NAME=$(git describe --tags --exact-match HEAD 2>/dev/null \
    || git describe --tags --abbrev=0 2>/dev/null || true)
fi
VER_NAME="${VER_NAME:-dev}"
VER_NAME="${VER_NAME#v}"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  VER_CODE=$(git rev-list HEAD --count 2>/dev/null || echo 1)
  VER_HASH=$(git rev-parse --verify --short HEAD 2>/dev/null || echo unknown)
else
  VER_CODE=1
  VER_HASH="nogit"
fi
BUILD_TYPE="${BUILD_TYPE:-release}"
VERSION_LINE="$VER_NAME ($VER_CODE-$VER_HASH-$BUILD_TYPE)"
sed -i "s/^version=.*/version=$VERSION_LINE/; s/^versionCode=.*/versionCode=$VER_CODE/" "$STAGE/module.prop"
info "版本: $VERSION_LINE (versionCode: $VER_CODE)"

# ---------- 7. 权限 ----------
find "$STAGE" -type f \( -name '*.sh' -o -name 'update-binary' \) -exec chmod 755 {} +
find "$STAGE" -type d -exec chmod 755 {} +
chmod 755 "$STAGE_BIN/sing-box" "$STAGE_BIN/atp" 2>/dev/null
[ -e "$STAGE_BIN/sing-box" ] || die "缺少 sing-box 二进制"

# ---------- 8. 打包 ----------
mkdir -p "$OUT_DIR"
if [ -z "$OUT_ZIP" ]; then
  OUT_ZIP="$OUT_DIR/FluxNet-${VER_NAME}-${VER_CODE}-${VER_HASH}-${BUILD_TYPE}-${TARGET_ABI}.zip"
fi
rm -f "$OUT_ZIP"
(cd "$STAGE" && zip -rq "$OUT_ZIP" .)

echo
info "已生成: $OUT_ZIP"
info "内置: sing-box $KERNEL_VER ($KERNEL_CHANNEL)"
[ -f "$STAGE_BIN/atp" ] && info "内置: AndroidTProxyShell (atp)"
ls -lh "$OUT_ZIP"
