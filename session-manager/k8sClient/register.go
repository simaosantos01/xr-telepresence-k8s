package k8sclient

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(telepresencev1.GroupVersion, &telepresencev1.Session{}, &telepresencev1.SessionList{})

	metav1.AddToGroupVersion(scheme, telepresencev1.GroupVersion)
	return nil
}
