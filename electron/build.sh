#!/bin/bash

set -e

echo "Building New API Electron App..."

# The frontend lives in its own repository. Point FRONTEND_DIR at that
# checkout; it must be a sibling of this repository by default.
FRONTEND_DIR="${FRONTEND_DIR:-../new-api-web}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Step 1: Building frontend..."
cd "$FRONTEND_DIR"
bun install --frozen-lockfile
DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(git describe --tags --always) bun run build

# The embed directive's path is fixed at web/dist, so the frontend's output
# is staged there.
echo "Staging frontend bundle in web/dist..."
rm -rf "$REPO_ROOT/web/dist"
mkdir -p "$REPO_ROOT/web/dist"
cp -r "$FRONTEND_DIR"/dist/. "$REPO_ROOT/web/dist/"

echo "Step 2: Building Go backend..."
cd "$REPO_ROOT"

if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Building for macOS..."
    CGO_ENABLED=1 go build -pgo="$(go list -m -f '{{.Dir}}' github.com/Calcium-Ion/moejs)/default.pgo" -ldflags="-s -w" -o new-api
    cd electron
    npm install
    npm run build:mac
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "Building for Linux..."
    CGO_ENABLED=1 go build -pgo="$(go list -m -f '{{.Dir}}' github.com/Calcium-Ion/moejs)/default.pgo" -ldflags="-s -w" -o new-api
    cd electron
    npm install
    npm run build:linux
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" || "$OSTYPE" == "win32" ]]; then
    echo "Building for Windows..."
    CGO_ENABLED=1 go build -pgo="$(go list -m -f '{{.Dir}}' github.com/Calcium-Ion/moejs)/default.pgo" -ldflags="-s -w" -o new-api.exe
    cd electron
    npm install
    npm run build:win
else
    echo "Unknown OS, building for current platform..."
    CGO_ENABLED=1 go build -pgo="$(go list -m -f '{{.Dir}}' github.com/Calcium-Ion/moejs)/default.pgo" -ldflags="-s -w" -o new-api
    cd electron
    npm install
    npm run build
fi

echo "Build complete! Check electron/dist/ for output."
