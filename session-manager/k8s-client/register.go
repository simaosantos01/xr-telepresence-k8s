package k8sClient

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(sessionv1alpha1.GroupVersion, &sessionv1alpha1.Session{}, &sessionv1alpha1.SessionList{})

	metav1.AddToGroupVersion(scheme, sessionv1alpha1.GroupVersion)
	return nil
}
