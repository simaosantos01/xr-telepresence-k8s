package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) GetPingServers(ctx *gin.Context) {
	pingServers := make(map[string]string)

	fmt.Printf("length %d", len(h.clusterClientsetMap))
	for cluster, clientset := range h.clusterClientsetMap {
		svc, err := clientset.CoreV1().Services("default").Get(ctx, "udp-ping", v1.GetOptions{})
		pingServers[cluster] = "problem getting ip"

		if err != nil {
			log.Println("cluster: ", cluster)
			log.Println(err.Error())
		}

		if err == nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
			ipAddress := svc.Status.LoadBalancer.Ingress[0].IP

			pingServers[cluster] = fmt.Sprintf("udp://%s:%d", ipAddress, 8080)
		}
	}

	ctx.JSON(http.StatusOK, pingServers)
}
