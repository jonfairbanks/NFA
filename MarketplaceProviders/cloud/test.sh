PROJECT_ID=101868473812
ENDPOINT_ID=9079161191267303424
INFERENCE_TOKEN=19dcf7e801e37e40e078d639cfeab6c7a3403ce0

curl \
-X POST \
-H "Authorization: Bearer 19dcf7e801e37e40e078d639cfeab6c7a3403ce0" \
-H "Content-Type: application/json" \
"https://9079161191267303424.us-central1-fasttryout.prediction.vertexai.goog/v1/projects/${PROJECT_ID}/locations/us-central1/endpoints/${ENDPOINT_ID}/chat/completions" \
-d '{
    "messages": [{
      "role": "user",
      "content": "Write a story about a magic backpack."
    }]
  }'

#   curl -X POST \
#     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
#     -H "Content-Type: application/json" \
#   https://us-central1-aiplatform.googleapis.com/v1beta1/projects/graphical-bus-433219-g5/locations/us-central1/endpoints/9079161191267303424/chat/completions \
#   -d '{
#     "messages": [{
#       "role": "user",
#       "content": "Write a story about a magic backpack."
#     }]
#   }'