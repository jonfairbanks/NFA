package main

import (
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/MorpheusAIs/Morpheus-Lumerin-Node/cli/chat"
	clientPkg "github.com/MorpheusAIs/Morpheus-Lumerin-Node/cli/chat/client"
)

type (
	ChatRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
	}

	SessionRequest = clientPkg.SessionRequest
)

var (
	client           *clientPkg.ApiGatewayClient
	currentSessionId string
	myAgentId        string //use this for model id (we're assuming the marketplace will treat agents the same as models)
)

func main() {

	// Load .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get agent ID from .env
	myAgentId = os.Getenv("AGENT_ID")
	if myAgentId == "" {
		log.Fatal("AGENT_ID not set in .env file")
	}

	r := gin.Default()

	r.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		modelId := c.PostForm("model")
		//TODO import client
		sessionRequest := &SessionRequest{
			ModelId: modelId,
		}

		//todo import client
		session, err := client.OpenSession(c, sessionRequest)

		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		currentSessionId = session.SessionId

		// Simulate streaming response
		//TODO stream from morpheus provider and pass stream back to client
		stream := func(c *gin.Context) {
			c.Stream(func(w io.Writer) bool {
				c.SSEvent("message", map[string]interface{}{
					"id":      "chatcmpl-123",
					"object":  "chat.completion.chunk",
					"created": 1677652288,
					"model":   modelId,
					"choices": []map[string]interface{}{
						{
							"delta": map[string]interface{}{
								"content": "Hello, how can I assist you today?",
							},
							"index":         0,
							"finish_reason": nil,
						},
					},
				})
				return false
			})
		}

		stream(c)
	})

	r.Run(":8080")
}
