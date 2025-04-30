package main

import (
	handlers "mr.telepresence/session-manager/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	handler, err := handlers.ConfigHandler()
	if err != nil {
		panic(err.Error())
	}

	v1 := router.Group("session-manager/v1")
	{
		v1.POST("/session", handler.CreateSession)
		v1.GET("/session", handler.GetSessions)
		v1.GET("/session/:sessionId", handler.GetSession)
		v1.DELETE("/session/:sessionId", handler.DeleteSession)

		v1.POST("/session/:sessionId/client", handler.CreateClient)
		v1.GET("/session/:sessionId/client", handler.GetClients)
		v1.GET("/session/:sessionId/client/:clientId", handler.GetClient)
		v1.PATCH("/session/:sessionId/client/:clientId", handler.UpdateClient)
		v1.DELETE("session/:sessionId/client/:clientId", handler.DeleteClient)

		v1.GET("/ping", handler.GetPingServers)
	}

	router.Run(":8080")
}
