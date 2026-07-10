#!/usr/bin/env bash
# Runs the app locally with vars from .env exported into its environment.
# Usage: ./run-dev.sh [args passed through to the app]
set -euo pipefail
cd "$(dirname "$0")"

if [[ ! -f .env ]]; then
    echo "run-dev.sh: no .env file found in $(pwd)" >&2
    exit 1
fi

set -a
source .env
set +a

exec go run ./cmd/overture-downloader "$@"

# nats pub downloads.requests --force-stdin -H "Job-Id:test-$(date +%s)" < test-request.json