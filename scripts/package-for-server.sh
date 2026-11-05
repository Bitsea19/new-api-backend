#!/usr/bin/env bash
# Package the backend source for upload to a server that builds its own image.
#
# The server runs `docker compose up -d --build`, which compiles the Go binary
# inside a container. So the archive has to contain the full source tree, not
# just a binary: go.mod, go.sum, the local relaykit/tokenkit modules, the
# Dockerfile and the source of every package.
#
# Usage (Git Bash on Windows, or any Linux shell):
#   ./scripts/package-for-server.sh [output-dir]
#
# The default output directory is the parent of this repository, so the archive
# lands next to the repository rather than inside it.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${1:-$(dirname "$REPO_ROOT")}"
ARCHIVE="$OUTPUT_DIR/new-api-backend-src.tar.gz"

cd "$REPO_ROOT"

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "Note: the working tree has uncommitted changes; the archive contains them."
fi

# Take every tracked file with its current content from the working tree, so
# uncommitted edits are included. Untracked files are skipped.
#
# The dashboard is a separate repository and is deliberately not included.
#   VERSION       read by the Dockerfile for the version stamp
#   relaykit      local module referenced by a `replace` directive in go.mod
#   tokenkit      local module referenced by a `replace` directive in go.mod
git ls-files -z \
    | tar --null --files-from=- --transform='s,^,new-api-backend/,' \
        -czf "$ARCHIVE"

echo "Wrote $ARCHIVE"
echo
echo "Upload it to the server, then:"
echo "  tar -xzf new-api-backend-src.tar.gz -C /opt"
echo "  cd /opt/new-api-backend"
echo "  docker compose up -d --build"
