package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ExtractReadyConditionStatusFromPod(pod *corev1.Pod) *corev1.ConditionStatus {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return &condition.Status
		}
	}
	return nil
}

func PodsAreReady(podList *corev1.PodList) bool {
	for _, pod := range podList.Items {
		status := ExtractReadyConditionStatusFromPod(&pod)

		if status != nil && (*status == corev1.ConditionFalse || *status == corev1.ConditionUnknown) {
			return false
		}
	}

	return true
}

func SpawnPod(
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
	pod *corev1.Pod) error {

	logger := log.FromContext(ctx)

	// set controller reference for garbage collection
	if err := ctrl.SetControllerReference(session, pod, scheme); err != nil {
		logger.Error(err, "unable to set controller reference for pod", "session", session.Name, "pod", pod.GetName())
		return err
	}

	if err := k8sclient.Create(ctx, pod); err != nil {
		logger.Error(err, "unable to create pod", "session", session.Name, "pod", pod.GetName())
		return err
	}

	return nil
}
