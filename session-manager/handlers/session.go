package handlers

import (
	"net/http"

	stdErrors "errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8sClient "mr.telepresence/session-manager/k8s-client"
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
		sessionClient, ok := h.clusterClientMap[body.SessionPodsCluster]
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

	for _, sessionClient := range h.clusterClientMap {
		if _, err := sessionClient.Sessions("default").Create(session, ctx); err != nil && !errors.IsAlreadyExists(err) {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}

	ctx.Header("Location", ctx.Request.URL.Path+"/"+sessionName)
	ctx.JSON(http.StatusCreated, nil)
}

func (h *Handler) GetSession(ctx *gin.Context) {
	sessionId := ctx.Param("sessionId")
	session, err := findSession(ctx, sessionId, h.clusterClientMap)
	if err != nil && errorIsSessionNotFound(err) {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	} else if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	}

	ctx.JSON(http.StatusOK, session)
}

func findSession(
	ctx *gin.Context,
	sessionId string,
	clusterClientMap map[string]*k8sClient.SessionClient,
) (*sessionv1alpha1.Session, error) {

	sessionSum := &sessionv1alpha1.Session{
		ObjectMeta: metav1.ObjectMeta{Name: sessionId},
		Spec: sessionv1alpha1.SessionSpec{
			Clients: make(map[string]bool),
		},
		Status: sessionv1alpha1.SessionStatus{
			Conditions: []metav1.Condition{},
			Clients:    make(map[string]sessionv1alpha1.ClientStatus),
		},
	}

	for _, cluterClient := range clusterClientMap {
		session, err := cluterClient.Sessions("default").Get(ctx, sessionId, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return nil, stdErrors.New("session not found")

		} else if err != nil {
			return nil, err
		}

		sessionSum.Spec.TimeoutSeconds = session.Spec.TimeoutSeconds
		sessionSum.Spec.ReutilizeTimeoutSeconds = session.Spec.ReutilizeTimeoutSeconds

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

	return sessionSum, nil
}

func (h *Handler) GetSessions(ctx *gin.Context) {
	sessionsLocation := []map[string]string{}

	for _, sessionClient := range h.clusterClientMap {
		sessions, err := sessionClient.Sessions("default").List(metav1.ListOptions{}, ctx)
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		for _, session := range sessions.Items {
			location := make(map[string]string)
			location["session"] = session.Name
			location["uri"] = ctx.Request.URL.Path + "/" + session.Name
			sessionsLocation = append(sessionsLocation, location)
		}

		break
	}

	ctx.JSON(http.StatusOK, sessionsLocation)
}

func (h *Handler) DeleteSession(ctx *gin.Context) {
	sessionId := ctx.Param("sessionId")

	deleted := false
	for _, sessionClient := range h.clusterClientMap {
		if err := sessionClient.Sessions("default").Delete(sessionId, metav1.DeleteOptions{}, ctx); err != nil &&
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
