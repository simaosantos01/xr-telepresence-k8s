package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
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
	ctx context.Context,
	rClient client.Client,
	scheme *runtime.Scheme,
	session *telepresencev1.Session,
	pod telepresencev1.Pod,
) error {

	logger := log.FromContext(ctx)

	pod.Name = session.Name + "-" + pod.Name
	pod.Labels["telepresence"] = "true"
	pod.Labels["svc"] = pod.Name

	corev1Pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Labels: pod.Labels, Namespace: "default"},
		Spec: pod.Spec}

	// set controller reference for garbage collection
	if err := ctrl.SetControllerReference(session, corev1Pod, scheme); err != nil {
		logger.Error(err, "unable to set controller reference for pod", "session", session.Name, "pod", corev1Pod.Name)
		return err
	}

	if err := rClient.Create(ctx, corev1Pod); err != nil {
		logger.Error(err, "unable to create pod", "session", session.Name, "pod", corev1Pod.Name)
		return err
	}

	return nil
}
