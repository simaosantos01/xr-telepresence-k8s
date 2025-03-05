package main

import (
	"flag"
	"path/filepath"
	handlers "telepresence-k8s/session-manager/handlers"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	k8sClient "telepresence-k8s/session-manager/k8sClient"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	k8sClient.AddToScheme(scheme.Scheme)

	clientset, err := k8sClient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	router.Use(func(c *gin.Context) {
		c.Set("clientset", clientset)
		c.Next()
	})

	v1 := router.Group("session-manager/v1")
	{
		v1.POST("/register-session", handlers.RegisterSession)
	}

	router.Run(":8080")
}
