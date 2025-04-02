package k8sclient

import (
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	telepresencev1 "mr.telepresence/controller/api/v1"
)

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
