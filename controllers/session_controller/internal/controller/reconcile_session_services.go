package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
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
		r.SetReadyCondition(session, metav1.ConditionUnknown, GET_PODS_FAILED_REASON, GET_PODS_FAILED_MESSAGE)
		logger.Error(err, "unable to get session pods", "session", session.Name)
		return err
	}

	var services corev1.ServiceList
	fieldSelector = ctrlClient.MatchingFields{ownerField: session.Name}

	if err := r.List(ctx, &services, ctrlClient.InNamespace(namespace), fieldSelector); err != nil {
		r.SetReadyCondition(session, metav1.ConditionUnknown, GET_SVC_FAILED_REASON, GET_SVC_FAILED_MESSAGE)
		logger.Error(err, "unable to get session services", "session", session.Name)
		return err
	}

	if len(session.Spec.SessionServices) == len(sessionPods.Items) {
		ready := PodsAreReady(&sessionPods)

		if ready {
			r.SetReadyCondition(session, metav1.ConditionTrue, PODS_READY_REASON, PODS_READY_MESSAGE)
		} else {
			r.SetReadyCondition(session, metav1.ConditionFalse, PODS_NOT_READY_REASON, PODS_NOT_READY_MESSAGE)
		}
	} else {
		if err := restorePods(r.Client, r.Scheme, ctx, session, sessionPods.Items, services.Items,
			session.Spec.SessionServices); err != nil {

			r.SetReadyCondition(session, metav1.ConditionFalse, PODS_NOT_READY_REASON, PODS_NOT_READY_MESSAGE)
			return err
		}
		r.SetReadyCondition(session, metav1.ConditionUnknown, PODS_RECONCILED_REASON, PODS_RECONCILED_MESSAGE)
	}

	return nil
}

func restorePods(
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
	foundPods []corev1.Pod,
	foundServices []corev1.Service,
	requiredPods []telepresencev1.Pod,
) error {

	foundPodsMap := make(map[string]struct{}, len(foundPods))
	foundServicesMap := make(map[string]struct{}, len(foundServices))

	for _, pod := range foundPods {
		foundPodsMap[pod.Name] = struct{}{}
	}

	for _, service := range foundServices {
		foundServicesMap[service.Name] = struct{}{}
	}

	for _, pod := range requiredPods {
		key := session.Name + "-" + pod.Name

		if _, exists := foundPodsMap[key]; !exists {
			if err := spawnPod(pod, k8sclient, scheme, ctx, session); err != nil {
				return err
			}
		}

		key = session.Name + "-" + pod.Name + "-svc"

		if _, exists := foundServicesMap[key]; !exists {
			if err := spawnService(pod, k8sclient, scheme, ctx, session); err != nil {
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
	pod.Labels["type"] = "session"
	pod.Labels["svc"] = pod.Name
	corev1Pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Labels: pod.Labels}, Spec: pod.Spec}
	return SpawnResource(k8sclient, scheme, ctx, session, corev1Pod)
}

func spawnService(
	forPod telepresencev1.Pod,
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
) error {
	servicePorts := []corev1.ServicePort{}

	for _, container := range forPod.Spec.Containers {
		for _, port := range container.Ports {
			servicePorts = append(servicePorts, corev1.ServicePort{Protocol: port.Protocol, Port: port.ContainerPort})
		}
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   session.Name + "-" + forPod.Name + "-svc",
			Labels: map[string]string{"type": "session"},
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: map[string]string{"svc": forPod.Name},
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}
	return SpawnResource(k8sclient, scheme, ctx, session, service)
}
