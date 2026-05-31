#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# -ne 1 ]]; then
    echo "usage: scripts/record-demo.sh <expect-script>" >&2
    exit 1
fi

EXPECT_SCRIPT="$1"
if [[ ! -f "$EXPECT_SCRIPT" ]]; then
    echo "error: expect script not found: $EXPECT_SCRIPT" >&2
    exit 1
fi
EXPECT_SCRIPT="$(cd "$(dirname "$EXPECT_SCRIPT")" && pwd)/$(basename "$EXPECT_SCRIPT")"

DEMO_NAME="$(basename "$EXPECT_SCRIPT")"
DEMO_NAME="${DEMO_NAME%.*}"
OUT="$ROOT_DIR/demos/$DEMO_NAME.gif"
TARGET="${GRPCEXP_DEMO_TARGET:-127.0.0.1:50051}"
SCRIPT_COLS="$(sed -n 's/^# demo-cols: //p' "$EXPECT_SCRIPT" | head -n 1)"
SCRIPT_ROWS="$(sed -n 's/^# demo-rows: //p' "$EXPECT_SCRIPT" | head -n 1)"
COLS="${GRPCEXP_DEMO_COLS:-${SCRIPT_COLS:-82}}"
ROWS="${GRPCEXP_DEMO_ROWS:-${SCRIPT_ROWS:-18}}"
TMP_DIR="${TMPDIR:-/tmp}/grpcexp-demo-recording"
BIN="$TMP_DIR/grpcexp"
CAST="$TMP_DIR/$DEMO_NAME.cast"
GO_CACHE="${GOCACHE:-$TMP_DIR/go-build-cache}"

for cmd in go expect asciinema agg; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "error: '$cmd' is required to record the demo" >&2
        exit 1
    fi
done

mkdir -p "$TMP_DIR" "$(dirname "$OUT")"

echo "Building grpcexp..."
(
    cd "$ROOT_DIR"
    GOCACHE="$GO_CACHE" go build -o "$BIN" ./cmd/grpcexp
)

echo "Recording terminal session..."
asciinema rec \
    --overwrite \
    --headless \
    --window-size "${COLS}x${ROWS}" \
    -i 1.4 \
    -c "GRPCEXP_DEMO_BIN=$BIN GRPCEXP_DEMO_TARGET=$TARGET expect $EXPECT_SCRIPT" \
    "$CAST"

echo "Rendering $OUT..."
agg \
    --theme github-dark \
    --cols "$COLS" \
    --rows "$ROWS" \
    --font-size 16 \
    --line-height 1.35 \
    --fps-cap 20 \
    --idle-time-limit 1.4 \
    --last-frame-duration 3 \
    --speed 0.85 \
    "$CAST" \
    "$OUT"

echo "Wrote $OUT"
