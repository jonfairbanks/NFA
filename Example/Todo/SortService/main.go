package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SortRequest struct {
	Strings []string `json:"strings" binding:"required"`
}

type SortResponse struct {
	SortedStrings []string `json:"sorted_strings"`
}

func main() {
	r := gin.Default()

	r.POST("/sort", func(c *gin.Context) {
		var req SortRequest

		// Bind the incoming JSON to the SortRequest struct
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Sort the strings
		sorting.Sort(req.Strings)

		// Return the sorted list
		c.JSON(http.StatusOK, SortResponse{SortedStrings: req.Strings})
	})

	r.Run(":8080") // Run the server on port 8080
}
