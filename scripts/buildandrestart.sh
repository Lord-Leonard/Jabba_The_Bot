#!/usr/bin/env bash
set -euo pipefail

# build_and_restart.sh
# Modes:
# - server-build (default): build and restart on the server where this script runs
# - local-upload: build on this machine, upload binary over SSH, then restart remote service

MODE="${MODE:-server-build}"                       # server-build | local-upload
PROJECT_DIR="${PROJECT_DIR:-$HOME/projects/Jabba_The_Bot}"
SERVICE_NAME="${SERVICE_NAME:-teamspeak-bot}"
BIN_PATH="${BIN_PATH:-/opt/jabba/bot}"           # full path to the installed binary
GO_BUILD_PKG="${GO_BUILD_PKG:-./cmd/teamspeak/}" # package to build ('.' = current module)
GO_BUILD_TAGS="${GO_BUILD_TAGS:-}"               # optional: e.g. "prod"
GO_LDFLAGS="${GO_LDFLAGS:-}"                     # optional: e.g. "-s -w"
USE_SUDO="${USE_SUDO:-0}"                        # set to 0 if you already run as root

# local-upload mode settings
REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-$USER}"
REMOTE_PORT="${REMOTE_PORT:-22}"
REMOTE_SSH_KEY="${REMOTE_SSH_KEY:-}"             # optional: path to private key
LOCAL_GOOS="${LOCAL_GOOS:-linux}"
LOCAL_GOARCH="${LOCAL_GOARCH:-amd64}"
LOCAL_CGO_ENABLED="${LOCAL_CGO_ENABLED:-0}"

need_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "Missing command: $1" >&2; exit 1; }; }
run_root() { if [[ "$USE_SUDO" == "1" ]]; then sudo "$@"; else "$@"; fi; }
shell_escape_sq() { sed "s/'/'\\''/g" <<<"$1"; }

build_local_tmpbin() {
    local out_path="$1"
    local -a build_args

    echo "==> Building locally: $GO_BUILD_PKG ($LOCAL_GOOS/$LOCAL_GOARCH)"
    build_args=(build -o "$out_path")
    [[ -n "$GO_BUILD_TAGS" ]] && build_args+=(-tags "$GO_BUILD_TAGS")
    [[ -n "$GO_LDFLAGS" ]] && build_args+=(-ldflags "$GO_LDFLAGS")
    build_args+=("$GO_BUILD_PKG")

    GOOS="$LOCAL_GOOS" GOARCH="$LOCAL_GOARCH" CGO_ENABLED="$LOCAL_CGO_ENABLED" go "${build_args[@]}"
}

if [[ "$MODE" == "server-build" ]]; then
    need_cmd go
    need_cmd systemctl

    echo "==> Mode:      $MODE"
    echo "==> Project:   $PROJECT_DIR"
    echo "==> Service:   $SERVICE_NAME"
    echo "==> Bin path:  $BIN_PATH"
    echo

    cd "$PROJECT_DIR"

    echo "==> Updating deps (go mod tidy/download)..."
    go mod tidy
    go mod download

    echo "==> Building..."
    tmpbin="$(mktemp)"
    trap 'rm -f "$tmpbin"' EXIT

    build_args=(build -o "$tmpbin")
    [[ -n "$GO_BUILD_TAGS" ]] && build_args+=(-tags "$GO_BUILD_TAGS")
    [[ -n "$GO_LDFLAGS" ]] && build_args+=(-ldflags "$GO_LDFLAGS")
    build_args+=("$GO_BUILD_PKG")

    go "${build_args[@]}"

    echo "==> Stopping service..."
    run_root systemctl stop "$SERVICE_NAME"

    echo "==> Installing binary to $BIN_PATH..."
    run_root install -d -m 0755 "$(dirname "$BIN_PATH")"
    run_root install -m 0755 "$tmpbin" "$BIN_PATH"

    echo "==> Starting service..."
    run_root systemctl start "$SERVICE_NAME"

    echo "==> Status:"
    run_root systemctl --no-pager --full status "$SERVICE_NAME" || true

    echo
    echo "Done."
    exit 0
fi

if [[ "$MODE" != "local-upload" ]]; then
    echo "Unsupported MODE='$MODE' (expected 'server-build' or 'local-upload')." >&2
    exit 1
fi

if [[ -z "$REMOTE_HOST" ]]; then
    echo "REMOTE_HOST is required in local-upload mode." >&2
    exit 1
fi

need_cmd go
need_cmd ssh
need_cmd scp

echo "==> Mode:        $MODE"
echo "==> Local proj:  $PROJECT_DIR"
echo "==> Remote host: $REMOTE_USER@$REMOTE_HOST:$REMOTE_PORT"
echo "==> Service:     $SERVICE_NAME"
echo "==> Bin path:    $BIN_PATH"
echo

cd "$PROJECT_DIR"

echo "==> Updating deps (go mod tidy/download)..."
go mod tidy
go mod download

tmpbin="$(mktemp)"
remote_tmp="/tmp/$(basename "$BIN_PATH").new.$$"
trap 'rm -f "$tmpbin"' EXIT

build_local_tmpbin "$tmpbin"

ssh_args=(-p "$REMOTE_PORT")
scp_args=(-P "$REMOTE_PORT")
if [[ -n "$REMOTE_SSH_KEY" ]]; then
    ssh_args+=(-i "$REMOTE_SSH_KEY")
    scp_args+=(-i "$REMOTE_SSH_KEY")
fi

echo "==> Uploading binary to remote temp path: $remote_tmp"
scp "${scp_args[@]}" "$tmpbin" "$REMOTE_USER@$REMOTE_HOST:$remote_tmp"

echo "==> Restarting remote service with new binary..."
ssh_cmd="BIN_PATH='$(shell_escape_sq "$BIN_PATH")' SERVICE_NAME='$(shell_escape_sq "$SERVICE_NAME")' REMOTE_TMP='$(shell_escape_sq "$remote_tmp")' USE_SUDO='$(shell_escape_sq "$USE_SUDO")' bash -s"
ssh "${ssh_args[@]}" "$REMOTE_USER@$REMOTE_HOST" "$ssh_cmd" <<'EOF'
set -euo pipefail
run_root() { if [[ "$USE_SUDO" == "1" ]]; then sudo "$@"; else "$@"; fi; }

run_root install -d -m 0755 "$(dirname "$BIN_PATH")"
run_root systemctl stop "$SERVICE_NAME"
run_root install -m 0755 "$REMOTE_TMP" "$BIN_PATH"
run_root rm -f "$REMOTE_TMP"
run_root systemctl start "$SERVICE_NAME"
run_root systemctl --no-pager --full status "$SERVICE_NAME" || true
EOF

echo
echo "Done."
