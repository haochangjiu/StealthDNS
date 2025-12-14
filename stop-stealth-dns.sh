#!/bin/bash
# Script to stop StealthDNS service

echo "Stopping StealthDNS service..."

# Find all stealth-dns processes
PIDS=$(ps aux | grep -E "stealth-dns run" | grep -v grep | awk '{print $2}')

if [ -z "$PIDS" ]; then
    echo "No StealthDNS service process found."
    exit 0
fi

# Kill each process
for PID in $PIDS; do
    echo "Stopping process with PID: $PID"
    sudo kill $PID 2>/dev/null || kill $PID 2>/dev/null
done

# Wait a moment and check if processes are still running
sleep 1
REMAINING=$(ps aux | grep -E "stealth-dns run" | grep -v grep | awk '{print $2}')

if [ -n "$REMAINING" ]; then
    echo "Some processes are still running, attempting force kill..."
    for PID in $REMAINING; do
        echo "Force killing process with PID: $PID"
        sudo kill -9 $PID 2>/dev/null || kill -9 $PID 2>/dev/null
    done
fi

# Final check
sleep 1
FINAL=$(ps aux | grep -E "stealth-dns run" | grep -v grep)

if [ -z "$FINAL" ]; then
    echo "StealthDNS service stopped successfully."
else
    echo "Warning: Some processes may still be running:"
    echo "$FINAL"
    exit 1
fi

