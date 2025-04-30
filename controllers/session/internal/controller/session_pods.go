package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gcv1alpha1 "mr.telepresence/gc/api/v1alpha1"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	"mr.telepresence/session/internal/controller/utils"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *SessionReconciler) ReconcileSessionPods(
	ctx context.Context,
	namespace string,
	session *sessionv1alpha1.Session,
	gcRegistrations []gcv1alpha1.GCRegistration,
	ingressServiceExternalIp *string,
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

	connectedClients := countConnectedClients(session.Spec.Clients)
	buildPodsStatus(session, session.Spec.SessionPodTemplates.Items, *ingressServiceExternalIp)
	manageGCRegistrationsForSessionPods(ctx, r.Client, session, connectedClients, sessionPods.Items, gcRegistrations)

	if len(session.Spec.SessionPodTemplates.Items) == len(sessionPods.Items) {
		ready := utils.PodsAreReady(&sessionPods)

		if ready && ingressServiceExternalIp != nil {
			setPodsStatusToTrue(session.Status.SessionPods.PodsStatus)
			utils.SetReadyCondition(session, metav1.ConditionTrue, utils.PODS_READY_REASON, utils.PODS_READY_MESSAGE)
		} else {
			utils.SetReadyCondition(session, metav1.ConditionFalse, utils.PODS_NOT_READY_REASON,
				utils.PODS_NOT_READY_MESSAGE)
		}
	} else if connectedClients > 0 {
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
	session *sessionv1alpha1.Session,
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
			status := session.Status.SessionPods.PodsStatus[key]
			status.Ready = false
			session.Status.SessionPods.PodsStatus[key] = status

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

func buildPodsStatus(
	session *sessionv1alpha1.Session,
	templates []corev1.PodTemplate,
	ingressServiceExternalIp string,
) {
	podsStatusMap := make(map[string]sessionv1alpha1.PodStatus)

	for _, template := range templates {
		podName := session.Name + "-" + template.Name
		podStatus := buildPodStatus(podName, template.Template.Spec)
		concatIngressExternalIpToPodPaths(ingressServiceExternalIp, &podStatus)
		podsStatusMap[podName] = podStatus
	}

	session.Status.SessionPods.PodsStatus = podsStatusMap
}

func setPodsStatusToTrue(podsStatus map[string]sessionv1alpha1.PodStatus) {
	for pod, podStatus := range podsStatus {
		podStatus.Ready = true
		podsStatus[pod] = podStatus
	}
}

func manageGCRegistrationsForSessionPods(
	ctx context.Context,
	rClient client.Client,
	session *sessionv1alpha1.Session,
	connectedClients int,
	foundPods []corev1.Pod,
	gcRegistrations []gcv1alpha1.GCRegistration,
) error {
	logger := log.FromContext(ctx)

	gcRegistrationsMap := make(map[string]*gcv1alpha1.GCRegistration, len(gcRegistrations))
	for _, gcRegistration := range gcRegistrations {
		gcRegistrationsMap[gcRegistration.Name] = &gcRegistration
	}

	for _, pod := range foundPods {
		if gcRegistration, ok := gcRegistrationsMap[pod.Name]; !ok && connectedClients == 0 {
			if err := createGCRegistration(ctx, rClient, pod.Name, session.Name, session.Namespace); err != nil {
				return err
			}

		} else if ok && connectedClients != 0 {
			if err := rClient.Delete(ctx, gcRegistration); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "unable to delete gc registration", "session", session.Name)
				return err
			}
		}
	}
	return nil
}

func countConnectedClients(specClients map[string]bool) int {
	count := 0

	for _, connected := range specClients {
		if connected {
			count++
		}
	}

	return count
}
