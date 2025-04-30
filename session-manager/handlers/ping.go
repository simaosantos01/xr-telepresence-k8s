package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) GetPingServers(ctx *gin.Context) {
	pingServers := make(map[string]string)

	for cluster, clientset := range h.clusterClientsetMap {
		svc, err := clientset.CoreV1().Services("default").Get(ctx, "udp-ping", v1.GetOptions{})
		pingServers[cluster] = "problem getting ip"

		if err == nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
			ipAddress := svc.Status.LoadBalancer.Ingress[0].IP
			port := svc.Status.LoadBalancer.Ingress[0].Ports[0].Port

			pingServers[cluster] = fmt.Sprintf("udp://%s:%d", ipAddress, port)
		}
	}

	ctx.JSON(http.StatusOK, pingServers)
}
