package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1alpha1 "mr.telepresence/session/api/v1alpha1"
	"mr.telepresence/session/internal/controller/utils"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *SessionReconciler) ReconcileSessionPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1alpha1.Session,
) error {

	logger := log.FromContext(ctx)

	var sessionPods corev1.PodList
	fieldSelector := client.MatchingFields{utils.PodOwnerField: session.Name, utils.PodTypeField: "session"}

	if err := r.List(ctx, &sessionPods, client.InNamespace(namespace), fieldSelector); err != nil {
		utils.SetReadyCondition(session, metav1.ConditionUnknown, utils.GET_PODS_FAILED_REASON,
			utils.GET_PODS_FAILED_MESSAGE)

		logger.Error(err, "unable to get session pods", "session", session.Name)
		return err
	}

	if len(session.Spec.SessionPodTemplates.Items) == len(sessionPods.Items) {
		ready := utils.PodsAreReady(&sessionPods)

		if ready {
			utils.SetReadyCondition(session, metav1.ConditionTrue, utils.PODS_READY_REASON, utils.PODS_READY_MESSAGE)
		} else {
			utils.SetReadyCondition(session, metav1.ConditionFalse, utils.PODS_NOT_READY_REASON,
				utils.PODS_NOT_READY_MESSAGE)
		}
	} else {
		if err := restorePods(ctx, r.Client, r.Scheme, session, sessionPods.Items,
			session.Spec.SessionPodTemplates.Items); err != nil {

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
	ctx context.Context,
	rClient client.Client,
	scheme *runtime.Scheme,
	session *telepresencev1alpha1.Session,
	foundPods []corev1.Pod,
	podTemplates []corev1.PodTemplate,
) error {
	foundPodsMap := make(map[string]struct{}, len(foundPods))

	for _, pod := range foundPods {
		foundPodsMap[pod.Name] = struct{}{}
	}

	for _, template := range podTemplates {
		key := session.Name + "-" + template.Name

		if _, exists := foundPodsMap[key]; !exists {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   key,
					Labels: map[string]string{"type": "session"},
				},
				Spec: template.Template.Spec,
			}

			if err := utils.SpawnPod(ctx, rClient, scheme, session, pod); err != nil {
				return err
			}
		}
	}

	return nil
}
