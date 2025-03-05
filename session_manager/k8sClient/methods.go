package k8sclient

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	telepresencev1 "mr.telepresence/controller/api/v1"
)

type SessionInterface interface {
	List(opts metav1.ListOptions, ctx context.Context) (*telepresencev1.SessionList, error)
	Get(name string, opts metav1.GetOptions, ctx context.Context) (*telepresencev1.Session, error)
	Create(session *telepresencev1.Session, ctx context.Context) (*telepresencev1.Session, error)
	//Delete(name string, opts metav1.DeleteOptions) error
}

type sessionClient struct {
	restClient rest.Interface
	namespace  string
}

func (c *sessionClient) List(opts metav1.ListOptions, ctx context.Context) (*telepresencev1.SessionList, error) {
	result := telepresencev1.SessionList{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("sessions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *sessionClient) Get(name string, opts metav1.GetOptions, ctx context.Context) (*telepresencev1.Session, error) {
	result := telepresencev1.Session{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("sessions").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *sessionClient) Create(session *telepresencev1.Session, ctx context.Context) (*telepresencev1.Session, error) {
	result := telepresencev1.Session{}
	err := c.restClient.
		Post().
		Namespace(c.namespace).
		Resource("sessions").
		Body(session).
		Do(ctx).
		Into(&result)

	return &result, err
}
