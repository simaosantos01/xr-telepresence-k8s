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
		v1.POST("/register-session", handler.RegisterSession)
	}

	router.Run(":8080")
}
