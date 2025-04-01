package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
	"mr.telepresence/controller/internal/controller/utils"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *SessionReconciler) ReconcileSessionPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	logger := log.FromContext(ctx)

	var sessionPods corev1.PodList
	fieldSelector := ctrlClient.MatchingFields{ownerField: session.Name, podTypeField: "session"}

	if err := r.List(ctx, &sessionPods, ctrlClient.InNamespace(namespace), fieldSelector); err != nil {
		utils.SetReadyCondition(session, metav1.ConditionUnknown, utils.GET_PODS_FAILED_REASON,
			utils.GET_PODS_FAILED_MESSAGE)

		logger.Error(err, "unable to get session pods", "session", session.Name)
		return err
	}

	if len(session.Spec.SessionServices) == len(sessionPods.Items) {
		ready := utils.PodsAreReady(&sessionPods)

		if ready {
			utils.SetReadyCondition(session, metav1.ConditionTrue, utils.PODS_READY_REASON, utils.PODS_READY_MESSAGE)
		} else {
			utils.SetReadyCondition(session, metav1.ConditionFalse, utils.PODS_NOT_READY_REASON,
				utils.PODS_NOT_READY_MESSAGE)
		}
	} else {
		if err := restorePods(r.Client, r.Scheme, ctx, session, sessionPods.Items,
			session.Spec.SessionServices); err != nil {

			utils.SetReadyCondition(session, metav1.ConditionFalse, utils.PODS_NOT_READY_REASON,
				utils.PODS_NOT_READY_MESSAGE)

			return err
		}
		utils.SetReadyCondition(session, metav1.ConditionUnknown, utils.PODS_RECONCILED_REASON,
			utils.PODS_RECONCILED_MESSAGE)
	}

	return nil
}

func restorePods(
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
	foundPods []corev1.Pod,
	requiredPods []telepresencev1.Pod,
) error {
	foundPodsMap := make(map[string]struct{}, len(foundPods))

	for _, pod := range foundPods {
		foundPodsMap[pod.Name] = struct{}{}
	}

	for _, pod := range requiredPods {
		key := session.Name + "-" + pod.Name

		if _, exists := foundPodsMap[key]; !exists {
			if err := spawnPod(pod, k8sclient, scheme, ctx, session); err != nil {
				return err
			}
		}
	}

	return nil
}

func spawnPod(
	pod telepresencev1.Pod,
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
) error {
	pod.Name = session.Name + "-" + pod.Name
	pod.Labels["telepresence"] = "true"
	pod.Labels["type"] = "session"
	pod.Labels["svc"] = pod.Name
	corev1Pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Labels: pod.Labels, Namespace: "default"},
		Spec: pod.Spec}
	return utils.SpawnPod(k8sclient, scheme, ctx, session, corev1Pod)
}
