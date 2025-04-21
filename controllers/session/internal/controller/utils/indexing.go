package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gcv1alpha1 "mr.telepresence/gc/api/v1alpha1"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PodOwnerField              = "ownerField"
	PodTypeField               = "podTypeField"
	GCRegistrationSessionField = "gcRegistrationSessionField"
)

func SetupManagerFieldIndexer(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, PodOwnerField,
		func(o client.Object) []string {

			return indexPodByOwner(o)
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, PodTypeField,
		func(o client.Object) []string {

			return indexPodByType(o)
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &gcv1alpha1.GCRegistration{},
		GCRegistrationSessionField,

		func(o client.Object) []string {

			return indexGCRegistrationBySession(o)
		}); err != nil {
		return err
	}
	return nil
}

func indexPodByOwner(obj client.Object) []string {
	owner := metav1.GetControllerOf(obj)

	if owner == nil {
		return nil
	}

	if owner.APIVersion != sessionv1alpha1.GroupVersion.String() || owner.Kind != "Session" {
		return nil
	}

	return []string{owner.Name}
}

func indexPodByType(obj client.Object) []string {
	pod := obj.(*corev1.Pod)

	podType, ok := pod.Labels["type"]
	if !ok {
		return nil
	}

	return []string{podType}
}

func indexGCRegistrationBySession(obj client.Object) []string {
	gcRegistration := obj.(*gcv1alpha1.GCRegistration)
	return []string{gcRegistration.Spec.Session.Name}
}
