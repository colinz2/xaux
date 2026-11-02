#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Doubao websocket configuration
export DOUBAO_WS_URL="wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async"
export DOUBAO_RESOURCE_ID="volc.seedasr.sauc.duration"

# API Key authentication; AK/SK is not required.
# Replace the placeholder with your API Key.
export DOUBAO_API_KEY="your-api-key"

if [[ ! -x "./proxy" ]]; then
    echo "[ERROR] proxy not found or not executable. Build it first:" >&2
    echo "go build -o proxy ./cmd/proxy" >&2
    exit 1
fi

echo "Starting xaux proxy with Doubao ASR..."
exec ./proxy --asr_provider=doubao "$@"
