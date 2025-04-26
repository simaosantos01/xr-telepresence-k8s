package handlers

import (
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
)

func (h *Handler) CreateSession(ctx *gin.Context) {
	var body RegisterSessionBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, ok := h.sessionTemplates[body.TemplateName]
	if !ok {
		message := "template name not configured"
		ctx.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	sessionName := body.TemplateName + "-" + uuid.New().String()[:4]

	session := &sessionv1alpha1.Session{
		ObjectMeta: metav1.ObjectMeta{
			Name: sessionName,
		},
		Spec: sessionv1alpha1.SessionSpec{
			SessionPodTemplates:     template.SessionPodTemplates,
			ClientPodTemplates:      template.ClientPodTemplates,
			TimeoutSeconds:          template.TimeoutSeconds,
			ReutilizeTimeoutSeconds: template.ReutilizeTimeoutSeconds,
			Clients:                 make(map[string]bool),
		},
	}

	if len(session.Spec.SessionPodTemplates.Items) != 0 {
		sessionClient, ok := h.clusterSessionClientMap[body.SessionPodsCluster]
		if !ok {
			message := "cluster not configured"
			ctx.JSON(http.StatusBadRequest, gin.H{"error": message})
			return
		}

		if _, err := sessionClient.Sessions("default").Create(session, ctx); err != nil && !errors.IsAlreadyExists(err) {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}

	session.Spec.SessionPodTemplates = corev1.PodTemplateList{Items: []corev1.PodTemplate{}}

	for _, sessionClient := range h.clusterSessionClientMap {
		if _, err := sessionClient.Sessions("default").Create(session, ctx); err != nil && !errors.IsAlreadyExists(err) {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}

	ctx.Header("Location", ctx.Request.URL.Path+"?id="+sessionName)
	ctx.JSON(http.StatusCreated, nil)
}

func (h *Handler) GetSession(ctx *gin.Context) {
	name, ok := ctx.GetQuery("id")
	if !ok {
		h.GetSessions(ctx)
		return
	}

	sessionSum := &sessionv1alpha1.Session{ObjectMeta: metav1.ObjectMeta{Name: name}}

	for _, sessionClient := range h.clusterSessionClientMap {
		session, err := sessionClient.Sessions("default").Get(name, metav1.GetOptions{}, ctx)
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		if len(session.Spec.SessionPodTemplates.Items) != 0 {
			sessionSum.Spec.SessionPodTemplates = session.Spec.SessionPodTemplates
			sessionSum.Status.Conditions = session.Status.Conditions
		}

		if len(sessionSum.Spec.ClientPodTemplates.Items) == 0 {
			sessionSum.Spec.ClientPodTemplates = session.Spec.ClientPodTemplates
		}

		for specClient, connected := range session.Spec.Clients {
			sessionSum.Spec.Clients[specClient] = connected
		}

		for statusClient, status := range session.Status.Clients {
			sessionSum.Status.Clients[statusClient] = status
		}
	}

	ctx.JSON(http.StatusOK, sessionSum)
}

func (h *Handler) GetSessions(ctx *gin.Context) {
	sessionsLocation := []map[string]string{}

	for _, sessionClient := range h.clusterSessionClientMap {
		sessions, err := sessionClient.Sessions("default").List(metav1.ListOptions{}, ctx)
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		for _, session := range sessions.Items {
			location := make(map[string]string)
			location["session"] = session.Name
			location["uri"] = ctx.Request.URL.Path + "?id=" + session.Name
			sessionsLocation = append(sessionsLocation, location)
		}

		break
	}

	ctx.JSON(http.StatusOK, sessionsLocation)
}

func (h *Handler) DeleteSession(ctx *gin.Context) {
	name, ok := ctx.GetQuery("id")
	if !ok {
		ctx.JSON(http.StatusNotFound, nil)
		return
	}

	deleted := false
	for _, sessionClient := range h.clusterSessionClientMap {
		if err := sessionClient.Sessions("default").Delete(name, metav1.DeleteOptions{}, ctx); err != nil &&
			!errors.IsNotFound(err) {

			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		} else if err == nil {
			deleted = true
		}
	}

	if !deleted {
		ctx.JSON(http.StatusNotFound, nil)
		return
	}

	ctx.JSON(http.StatusOK, nil)
}
