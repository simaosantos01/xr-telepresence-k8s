package utils

import (
	corev1 "k8s.io/api/core/v1"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func SessionUpdateFunc(e event.UpdateEvent) bool {
	oldObj := e.ObjectOld.(*sessionv1alpha1.Session)
	newObj := e.ObjectNew.(*sessionv1alpha1.Session)

	if len(oldObj.Spec.Clients) != len(newObj.Spec.Clients) {
		return true
	}

	for specClient, connected := range newObj.Spec.Clients {
		if v, ok := oldObj.Spec.Clients[specClient]; ok {
			return v != connected
		} else {
			return true
		}
	}

	return false
}

func PodUpdateFunc(e event.UpdateEvent) bool {
	oldObj := e.ObjectOld.(*corev1.Pod)
	newObj := e.ObjectNew.(*corev1.Pod)

	return ExtractReadyConditionStatusFromPod(oldObj) != ExtractReadyConditionStatusFromPod(newObj)
}
