#!/bin/bash
set -e
set -x

echo "Starting backtor..."
backtor \
    --elasticsearch-url=$ELASTICSEARCH_URL \
    --log-level=$LOG_LEVEL

