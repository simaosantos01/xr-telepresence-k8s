package handlers

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	telepresencev1 "mr.telepresence/controller/api/v1"
)

func (h *Handler) RegisterSession(ctx *gin.Context) {
	session := telepresencev1.Session{
		ObjectMeta: metav1.ObjectMeta{
			Name: "session-1",
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

func GetSession(ctx *gin.Context) {

}
