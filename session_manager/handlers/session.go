package handlers

import (
	"net/http"

	k8sclient "telepresence-k8s/session-manager/k8sClient"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
)

func RegisterSession(ctx *gin.Context) {
	value, exists := ctx.Get("clientset")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not found"})
		return
	}

	clientset := value.(*k8sclient.SessionClient)

	sessions, err := clientset.Sessions("default").List(metav1.ListOptions{}, ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	//var session SessionId

	// if err := ctx.ShouldBindJSON(&session); err != nil {
	// 	ctx.Error(err)
	// 	ctx.AbortWithStatus(http.StatusBadRequest)
	// 	return
	// }
	ctx.JSON(http.StatusOK, sessions)
}
