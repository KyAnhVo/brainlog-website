#!/usr/bin/env bash
# Builds both frontends, embeds them into the Go server, and writes server/bin/brainlog-server.
#   customer-facing-client -> served at /
#   admin-panel-client     -> served at /admin/
#   API                    -> /api/...
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

build_frontend() {
	local app="$1" name="$2"
	local dest="$root/server/internal/web/dist/$name"
	echo "==> building $app"
	(cd "$root/$app" && bun install --frozen-lockfile && bun run build)
	# Replace the previous build, keeping the committed placeholder.
	find "$dest" -mindepth 1 ! -name .gitkeep -delete
	cp -R "$root/$app/dist/." "$dest/"
	rm -f "$dest/mockServiceWorker.js" # dev-only MSW worker
}

build_frontend customer-facing-client customer
build_frontend admin-panel-client admin

echo "==> building server"
mkdir -p "$root/server/bin"
(cd "$root/server" && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/brainlog-server .)
echo "==> done: server/bin/brainlog-server"
