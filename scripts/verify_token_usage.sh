#!/bin/bash
set -e

# Build and start the environment (if not already running)
echo "Starting environment..."
podman compose up -d

# Wait for services to be healthy
echo "Waiting for services..."
sleep 10

# Perform an AddEpisode request to trigger LLM usage (Extraction)
# Perform an AddEpisode request (via /messages) to trigger LLM usage
echo "Triggering LLM usage via AddMessages API..."
curl -s -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "test-group",
    "saga": "test-saga",
    "messages": [
      {"role": "user", "content": "Alice is a software engineer."}
    ]
  }' > /dev/null

# Wait briefly for logs to flush
sleep 2

# Check logs for token usage
echo "Checking logs for token usage..."
# We grep for "LLM Usage" which is the format used in internal/llm/openai.go
LOG_output=$(podman compose logs app 2>&1 | grep "LLM Usage" | tail -n 1)

if [ -n "$LOG_output" ]; then
  echo "SUCCESS: Token usage detected automatically!"
  echo "Latest usage log: $LOG_output"
else
  echo "FAILURE: No token usage log found."
  exit 1
fi
