#!/bin/bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your_model_id",
    "messages": [{"role": "user", "content": "Hello, how are you?"}]
  }'

  curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "your_model_id",
    "messages": [{"role": "user", "content": "Tell me a joke"}],
    "stream": true
  }'