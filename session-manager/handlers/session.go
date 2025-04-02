package handlers

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	telepresencev1 "mr.telepresence/controller/api/v1"
)

func (h *Handler) RegisterSession(ctx *gin.Context) {
	var body RegisterSessionBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session := telepresencev1.Session{
		ObjectMeta: metav1.ObjectMeta{
			Name: body.Name,
		},
		Spec: telepresencev1.SessionSpec{
			Services: corev1.PodTemplateList{
				Items: []corev1.PodTemplate{},
			},
			BackgroundServices: corev1.PodTemplateList{
				Items: []corev1.PodTemplate{},
			},
			Clients:        []string{},
			TimeoutSeconds: 120,
		},
	}

	_, err := h.clientset.Sessions("default").Create(&session, ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, session)
}

func (h *Handler) GetSession(ctx *gin.Context) {
	name, ok := ctx.GetQuery("name")
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "missing param 'name'"})
		return
	}

	session, err := h.clientset.Sessions("default").Get(name, metav1.GetOptions{}, ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, session)
}

func (h *Handler) GetAll(ctx *gin.Context) {
	sessions, err := h.clientset.Sessions("default").List(metav1.ListOptions{}, ctx)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, sessions)
}
