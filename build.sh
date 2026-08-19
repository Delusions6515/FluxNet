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
#   TARGET_ABI        目标 ABI: arm64-v8a(默认)|armeabi-v7a|x86_64|x86
#   KERNEL_CHANNEL    内核渠道: delusions6515-pre(默认)|delusions6515-stable|
#                               ref1nd-pre|ref1nd-stable|official-stable|official-pre
#   OUT_DIR           输出目录 (默认 ./build)
# ============================================================
set -euo pipefail

REPO_DIR=$(cd "$(dirname "$0")" && pwd)
MODULE_DIR="$REPO_DIR/module"
GO_DIR="$REPO_DIR/go"
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
  arm64-v8a)   SINGBOX_ARCH="arm64" ;;
  armeabi-v7a) SINGBOX_ARCH="arm" ;;
  x86_64)      SINGBOX_ARCH="amd64" ;;
  x86)         SINGBOX_ARCH="386" ;;
  *) die "不支持的 TARGET_ABI: $TARGET_ABI" ;;
esac
info "目标: $TARGET_ABI (sing-box: $SINGBOX_ARCH)  渠道: $KERNEL_CHANNEL"

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

# ---------- 1. 编译 Go ----------
info "编译 fluxnet (GOOS=android GOARCH=arm64 CGO_ENABLED=0) ..."
(
  cd "$GO_DIR"
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags="-s -w" -o "$MODULE_DIR/bin/fluxnet" ./cmd/fluxnet
)
info "fluxnet: $MODULE_DIR/bin/fluxnet"

# ---------- 2. 构建暂存 ----------
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
cp -r "$MODULE_DIR/." "$STAGE/"
find "$STAGE" -name .DS_Store -delete
STAGE_BIN="$STAGE/bin"

# ---------- 3. 获取 sing-box ----------
KERNEL_BIN="$STAGE_BIN/sing-box"
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
rm -rf "$TMP"
info "sing-box: $KERNEL_VER"

# ---------- 4. 获取 atp ----------
ATP_SRC="https://raw.githubusercontent.com/CHIZI-0618/AndroidTProxyShell/main/atp"
ATP_BIN="$STAGE_BIN/atp"
info "下载 AndroidTProxyShell (atp) ..."
if download "$ATP_SRC" "$ATP_BIN.tmp"; then
  mv -f "$ATP_BIN.tmp" "$ATP_BIN"
  chmod 755 "$ATP_BIN"
  info "atp: 已获取"
else
  rm -f "$ATP_BIN.tmp"
  warn "atp 下载失败, 跳过"
fi

# ---------- 5. META-INF ----------
mkdir -p "$STAGE/META-INF/com/google/android"
if download "https://raw.githubusercontent.com/topjohnwu/Magisk/master/scripts/module_installer.sh" \
  "$STAGE/META-INF/com/google/android/update-binary"; then
  printf '#MAGISK\n' > "$STAGE/META-INF/com/google/android/updater-script"
  info "已获取 module_installer.sh"
else
  rm -rf "$STAGE/META-INF"
  warn "META-INF 获取失败, 跳过"
fi

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
BUILD_TYPE="release"
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
