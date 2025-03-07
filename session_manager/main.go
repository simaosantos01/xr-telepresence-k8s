package main

import (
	handlers "telepresence-k8s/session-manager/handlers"

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
		v1.POST("/session", handler.RegisterSession)
		v1.GET("/session", handler.GetSession)
		v1.GET("/session/all", handler.GetAll)

		v1.PATCH("/client/connect", handler.ConnectClient)
		v1.PATCH("/client/disconnect", handler.DisconnectClient)
	}

	router.Run(":8080")
}
