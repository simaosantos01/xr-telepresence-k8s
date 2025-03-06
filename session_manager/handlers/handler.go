package handlers

import (
	"flag"
	"path/filepath"
	k8sclient "telepresence-k8s/session-manager/k8sClient"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Handler struct {
	clientset *k8sclient.SessionClient
}

func ConfigHandler() (*Handler, error) {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	k8sclient.AddToScheme(scheme.Scheme)

	clientset, err := k8sclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Handler{clientset: clientset}, nil
}
