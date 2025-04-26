package k8sClient

import (
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
)

type SessionClient struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*SessionClient, error) {
	c.ContentConfig.GroupVersion = &sessionv1alpha1.GroupVersion
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
