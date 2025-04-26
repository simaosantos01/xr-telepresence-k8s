package handlers

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// func (h *Handler) ConnectClient(ctx *gin.Context) {
// 	var body PatchClientsBody

// 	if err := ctx.ShouldBindJSON(&body); err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	session, err := h.clientset.Sessions("default").Get(body.SessionName, metav1.GetOptions{}, ctx)
// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	clientExists := false
// 	for _, client := range session.Spec.Clients {
// 		if client == body.ClientName {
// 			clientExists = true
// 			break
// 		}
// 	}

// 	if clientExists {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": "client already exists"})
// 		return
// 	}

// 	session.Spec.Clients = append(session.Spec.Clients, body.ClientName)
// 	_, err = h.clientset.Sessions("default").UpdateClients(body.SessionName, metav1.PatchOptions{}, session, ctx)
// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, session)
// }

// func (h *Handler) DisconnectClient(ctx *gin.Context) {
// 	var body PatchClientsBody

// 	if err := ctx.ShouldBindJSON(&body); err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	session, err := h.clientset.Sessions("default").Get(body.SessionName, metav1.GetOptions{}, ctx)
// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	removed := false
// 	for index, client := range session.Spec.Clients {
// 		if client == body.ClientName {
// 			removed = true
// 			session.Spec.Clients = append(session.Spec.Clients[:index], session.Spec.Clients[index+1:]...)
// 			break
// 		}
// 	}

// 	if !removed {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": "client does not exist"})
// 		return
// 	}

// 	_, err = h.clientset.Sessions("default").UpdateClients(body.SessionName, metav1.PatchOptions{}, session, ctx)
// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, session)
// }
