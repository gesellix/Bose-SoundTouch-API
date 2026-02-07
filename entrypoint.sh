#!/bin/sh

# Start Python backend on port 8001
# Note: Go frontend expects it on 8001 by default
echo "Starting Python backend on port 8001..."
fastapi run soundcork/main.py --port 8001 --host 0.0.0.0 &

# Start Go frontend on the primary PORT
echo "Starting Go frontend on port ${PORT}..."
./bose-soundtouch-api-bin &

# Wait for processes
wait
