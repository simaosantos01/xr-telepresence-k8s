package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1alpha1 "mr.telepresence/session/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ExtractReadyConditionStatusFromPod(pod *corev1.Pod) corev1.ConditionStatus {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status
		}
	}
	return corev1.ConditionStatus("")
}

func PodsAreReady(podList *corev1.PodList) bool {
	for _, pod := range podList.Items {
		status := ExtractReadyConditionStatusFromPod(&pod)

		if status != "" && (status == corev1.ConditionFalse || status == corev1.ConditionUnknown) {
			return false
		}
	}

	return true
}

func SpawnPod(
	ctx context.Context,
	rClient client.Client,
	scheme *runtime.Scheme,
	session *telepresencev1alpha1.Session,
	pod *corev1.Pod,
) error {
	logger := log.FromContext(ctx)

	pod.Labels["telepresence"] = "true"
	pod.Labels["svc"] = pod.Name
	pod.Namespace = "default"

	// set controller reference for garbage collection
	if err := ctrl.SetControllerReference(session, pod, scheme); err != nil {
		logger.Error(err, "unable to set controller reference for pod", "session", session.Name, "pod", pod.Name)
		return err
	}

	if err := rClient.Create(ctx, pod); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "unable to create pod", "session", session.Name, "pod", pod.Name)
		return err
	}

	return nil
}
