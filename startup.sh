#!/bin/bash
set -e
set -x

echo "Starting elasticblast..."
elasticblast \
    --blast-url=$BLAST_URL \
    --log-level=$LOG_LEVEL

