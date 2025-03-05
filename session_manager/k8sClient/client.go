package k8sclient

import (
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	telepresencev1 "mr.telepresence/controller/api/v1"
)

func k8sClient() *rest.RESTClient {
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

	AddToScheme(scheme.Scheme)

	config.ContentConfig.GroupVersion = &telepresencev1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		panic(err.Error())
	}

	return client
}

type SessionClient struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*SessionClient, error) {
	c.ContentConfig.GroupVersion = &telepresencev1.GroupVersion
	c.APIPath = "/apis"
	c.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	c.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.UnversionedRESTClientFor(c)
	if err != nil {
		return nil, err
	}

	return &SessionClient{restClient: client}, nil
}

func (c *SessionClient) Sessions(namespace string) SessionInterface {
	return &sessionClient{
		restClient: c.restClient,
		namespace:  namespace,
	}
}
