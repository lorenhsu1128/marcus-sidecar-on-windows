#!/bin/bash
# Background screenshot timer - takes screenshots at regular intervals
# Usage: ./screenshot-timer.sh <interval_seconds> <count> [prefix]
#
# Run this in background, then interact with sidecar manually or via agent

INTERVAL="${1:-2}"
COUNT="${2:-5}"
PREFIX="${3:-capture}"
OUTPUT_DIR="$(dirname "$0")/../docs/screenshots"

mkdir -p "$OUTPUT_DIR"

echo "Taking $COUNT screenshots every ${INTERVAL}s..."
echo "Press Ctrl+C to stop early"

for i in $(seq 1 $COUNT); do
    FILENAME="${PREFIX}-$(printf '%02d' $i)-$(date +%H%M%S).png"
    screencapture -x "$OUTPUT_DIR/$FILENAME"
    echo "[$i/$COUNT] Captured: $FILENAME"
    
    if [ $i -lt $COUNT ]; then
        sleep $INTERVAL
    fi
done

echo "Done! Screenshots saved to $OUTPUT_DIR"
