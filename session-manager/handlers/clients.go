package handlers

import (
	"encoding/json"
	"net/http"

	stdErrors "errors"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "mr.telepresence/session-manager/k8s-client"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
)

func (h *Handler) CreateClient(ctx *gin.Context) {
	var body CreateClientBody

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, ok := h.clusterClientMap[body.Cluster]; !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "cluster not configured"})
		return
	}

	// Find session
	sessionId := ctx.Param("sessionId")
	clusterClient, session, err := findSession(ctx, h.clusterClientMap, sessionId)

	if err != nil && errorIsSessionNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return

	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Check if client exists
	if _, ok := session.Spec.Clients[body.ClientId]; ok {
		ctx.JSON(http.StatusConflict, gin.H{"error": "client already exists"})
		return
	}

	// Create client
	session.Spec.Clients[body.ClientId] = true
	patchSession(ctx, session, clusterClient, nil)
}

func findSession(
	ctx *gin.Context,
	clusterClientMap map[string]*k8sClient.SessionClient,
	sessionId string,
) (*k8sClient.SessionClient, *sessionv1alpha1.Session, error) {

	for _, client := range clusterClientMap {
		session, err := client.Sessions("default").Get(sessionId, metav1.GetOptions{}, ctx)

		if err != nil && !errors.IsNotFound(err) {
			return nil, nil, err

		} else if err == nil {
			return client, session, nil
		}
	}

	return nil, nil, stdErrors.New("session not found")
}

func errorIsSessionNotFound(err error) bool {
	return err.Error() == "session not found"
}

func patchSession(ctx *gin.Context, session *sessionv1alpha1.Session, clusterClient *k8sClient.SessionClient, patchData []byte) {
	_, err := clusterClient.Sessions("default").PatchClients(ctx, session.Name, session, patchData, metav1.PatchOptions{})

	if err != nil && errors.IsNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})

	} else {
		ctx.JSON(http.StatusOK, nil)
	}
}

func (h *Handler) GetClient(ctx *gin.Context) {
	// Find session
	sessionId := ctx.Param("sessionId")
	_, session, err := findSession(ctx, h.clusterClientMap, sessionId)

	if err != nil && errorIsSessionNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return

	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Check if client exists
	clientId := ctx.Param("clientId")
	if _, ok := session.Spec.Clients[clientId]; !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "client does not exist"})
		return
	}

	response := make(map[string]interface{})
	response["spec"] = map[string]bool{"connected": session.Spec.Clients[clientId]}
	response["status"] = session.Status.Clients[clientId]
	ctx.JSON(http.StatusOK, response)
}

func (h *Handler) GetClients(ctx *gin.Context) {
	clientsLocation := []map[string]string{}

	// Find session
	sessionId := ctx.Param("sessionId")
	_, session, err := findSession(ctx, h.clusterClientMap, sessionId)

	if err != nil && errorIsSessionNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return

	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	for client := range session.Spec.Clients {
		location := make(map[string]string)
		location["client"] = client
		location["uri"] = ctx.Request.URL.Path + "/" + client
		clientsLocation = append(clientsLocation, location)
	}

	ctx.JSON(http.StatusOK, clientsLocation)
}

func (h *Handler) UpdateClient(ctx *gin.Context) {
	var body UpdateClientBody

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find session
	sessionId := ctx.Param("sessionId")
	clusterClient, session, err := findSession(ctx, h.clusterClientMap, sessionId)

	if err != nil && errorIsSessionNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return

	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Check if client exists
	clientId := ctx.Param("clientId")
	if _, ok := session.Spec.Clients[clientId]; !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "client does not exist"})
		return
	}

	// Update client
	session.Spec.Clients[clientId] = *body.Connected
	patchSession(ctx, session, clusterClient, nil)
}

func (h *Handler) DeleteClient(ctx *gin.Context) {
	// Find session
	sessionId := ctx.Param("sessionId")
	clusterClient, session, err := findSession(ctx, h.clusterClientMap, sessionId)

	if err != nil && errorIsSessionNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return

	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Check if client exists
	clientId := ctx.Param("clientId")
	if _, ok := session.Spec.Clients[clientId]; !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "client does not exist"})
		return
	}

	// Delete client
	patchData, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"clients": map[string]interface{}{
				clientId: nil,
			},
		},
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	patchSession(ctx, session, clusterClient, patchData)
}
