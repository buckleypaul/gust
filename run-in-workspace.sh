#!/bin/bash
# Helper script to run gust in the Zephyr workspace

set -e

GUST_BIN="$(cd "$(dirname "$0")" && pwd)/gust"
WORKSPACE="/Users/paulbuckley/Projects/zephyr-workspace"

if [ ! -f "$GUST_BIN" ]; then
    echo "Error: gust binary not found. Run 'go build ./cmd/gust' first."
    exit 1
fi

if [ ! -d "$WORKSPACE/.west" ]; then
    echo "Error: Zephyr workspace not found at $WORKSPACE"
    exit 1
fi

echo "Launching gust in workspace: $WORKSPACE"
echo "Press q to quit, Tab to toggle focus, ? for help"
echo ""

cd "$WORKSPACE"
exec "$GUST_BIN"
