#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$ROOT_DIR/backend/build"
LAYER_ZIP="$BUILD_DIR/ffmpeg-layer.zip"
TMP_DIR="$BUILD_DIR/tmp_ffmpeg"

mkdir -p "$BUILD_DIR"

if [ -f "$LAYER_ZIP" ] && [ "${1:-}" != "--force" ]; then
    echo "FFmpeg layer zip already exists at $LAYER_ZIP (use --force to rebuild)"
    exit 0
fi

echo "==> Building FFmpeg ARM64 Lambda Layer..."
rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR/bin"

URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz"
TARBALL="$BUILD_DIR/ffmpeg-arm64-static.tar.xz"

if [ ! -f "$TARBALL" ]; then
    echo "==> Downloading John Van Sickle static ARM64 FFmpeg release..."
    curl -fSL "$URL" -o "$TARBALL"
else
    echo "==> Using cached tarball at $TARBALL"
fi

echo "==> Extracting ffmpeg binary..."
tar -xJf "$TARBALL" -C "$TMP_DIR"

FFMPEG_SRC=$(find "$TMP_DIR" -type f -name "ffmpeg" | head -n 1)
if [ -z "$FFMPEG_SRC" ]; then
    echo "ERROR: failed to find ffmpeg in extracted archive"
    exit 1
fi

mv "$FFMPEG_SRC" "$TMP_DIR/bin/ffmpeg"
chmod +x "$TMP_DIR/bin/ffmpeg"

echo "==> Packaging layer zip: $LAYER_ZIP..."
rm -f "$LAYER_ZIP"
(
    cd "$TMP_DIR"
    zip -q -9 -r "$LAYER_ZIP" bin/ffmpeg
)

rm -rf "$TMP_DIR"
echo "==> Successfully created $LAYER_ZIP ($(du -h "$LAYER_ZIP" | cut -f1))"
