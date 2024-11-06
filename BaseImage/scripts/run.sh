#!/bin/bash
docker run -d -p 8080:8080 \
  -e MARKETPLACE_URL=http://your-marketplace-url \
  -e MODEL_ID=your_model_id \
  -e SESSION_DURATION=1h \
  nfa-base