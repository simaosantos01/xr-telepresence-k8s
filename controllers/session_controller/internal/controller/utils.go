package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IndexSessionPods(obj ctrlClient.Object) []string {
	pod := obj.(*corev1.Pod)
	owner := metav1.GetControllerOf(pod)

	if owner == nil {
		return nil
	}

	if owner.APIVersion != apiGVStr || owner.Kind != "Session" {
		return nil
	}

	purpose, ok := pod.Labels["purpose"]
	if !ok {
		return nil

	} else if purpose != "session" {
		return nil
	}

	return []string{owner.Name}
}

func IndexBackgroundPods(obj ctrlClient.Object) []string {
	pod := obj.(*corev1.Pod)
	owner := metav1.GetControllerOf(pod)

	if owner == nil {
		return nil
	}

	if owner.APIVersion != apiGVStr || owner.Kind != "Session" {
		return nil
	}

	client, ok := pod.Annotations["client"]
	if !ok {
		return nil
	}

	return []string{client}
}

func PodsAreReady(podList *corev1.PodList) bool {
	for _, pod := range podList.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady &&
				(condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown) {

				return false
			}
		}
	}

	return true
}

func SpawnResource(
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
	resource ctrlClient.Object) error {

	logger := log.FromContext(ctx)

	// set controller reference for garbage collection
	if err := ctrl.SetControllerReference(session, resource, scheme); err != nil {
		logger.Error(err, "unable to set controller reference for pod", "session", session.Name, "pod", resource.GetName())
		return err
	}

	if err := k8sclient.Create(ctx, resource); err != nil {
		logger.Error(err, "unable to create pod", "session", session.Name, "pod", resource.GetName())
		return err
	}

	return nil
}
