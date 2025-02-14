package main

import (
	"encoding/json"
	"net/http"
	"os"

	"mr.telepresence/clientPolling/types"

	"github.com/gin-gonic/gin"
)

func getPollingData(c *gin.Context) {

	data, err := os.ReadFile("data.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read data file"})
		return
	}

	var pollingData types.PollingData
	if err := json.Unmarshal(data, &pollingData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse JSON"})
		return
	}

	c.JSON(http.StatusOK, pollingData)
}

func main() {
	router := gin.Default()
	router.GET("/", getPollingData)

	router.Run(":8080")
}
