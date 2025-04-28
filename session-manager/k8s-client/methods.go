package k8sClient

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
)

type SessionInterface interface {
	List(opts metav1.ListOptions, ctx context.Context) (*sessionv1alpha1.SessionList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*sessionv1alpha1.Session, error)
	Create(session *sessionv1alpha1.Session, ctx context.Context) (*sessionv1alpha1.Session, error)
	PatchClients(ctx context.Context, sessionName string, session *sessionv1alpha1.Session, patchData []byte, opts metav1.PatchOptions) (*sessionv1alpha1.Session, error)
	Delete(name string, opts metav1.DeleteOptions, ctx context.Context) error
}

type sessionClient struct {
	restClient rest.Interface
	namespace  string
}

func (c *sessionClient) List(opts metav1.ListOptions, ctx context.Context) (*sessionv1alpha1.SessionList, error) {
	result := sessionv1alpha1.SessionList{}
	err := c.restClient.
		Get().
		Namespace(c.namespace).
		Resource("sessions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *sessionClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*sessionv1alpha1.Session, error) {
	result := sessionv1alpha1.Session{}
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

func (c *sessionClient) Create(session *sessionv1alpha1.Session, ctx context.Context) (*sessionv1alpha1.Session, error) {
	result := sessionv1alpha1.Session{}
	err := c.restClient.
		Post().
		Namespace(c.namespace).
		Resource("sessions").
		Body(session).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *sessionClient) PatchClients(
	ctx context.Context,
	sessionName string,
	session *sessionv1alpha1.Session,
	patchData []byte,
	opts metav1.PatchOptions,
) (*sessionv1alpha1.Session, error) {

	if patchData == nil {
		var err error
		patchData, err = json.Marshal(map[string]interface{}{
			"spec": map[string]interface{}{
				"clients": session.Spec.Clients,
			},
		})

		if err != nil {
			return nil, err
		}
	}

	result := sessionv1alpha1.Session{}
	err := c.restClient.
		Patch(types.MergePatchType).
		Namespace(c.namespace).
		Resource("sessions").
		Name(sessionName).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(patchData).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *sessionClient) Delete(name string, opts metav1.DeleteOptions, ctx context.Context) error {
	result := sessionv1alpha1.Session{}
	return c.restClient.
		Delete().
		Namespace(c.namespace).
		Resource("sessions").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
}
