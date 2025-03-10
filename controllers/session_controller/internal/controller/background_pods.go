package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *SessionReconciler) ReconcileBackgroundPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	logger := log.FromContext(ctx)

	for _, client := range session.Spec.Clients {

		var backgroundPods corev1.PodList
		if err := r.List(ctx, &backgroundPods, ctrlClient.InNamespace(namespace),
			ctrlClient.MatchingFields{clientKey: client}); err != nil {

			logger.Error(err, "unable to get client background pods", "session", session.Name, "client", client)
			return err
		}

		if len(session.Spec.BackgroundPods) == len(backgroundPods.Items) {
			// all background pods exist, lets verify if all pods are ready
			ready := true

			for _, pod := range backgroundPods.Items {
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.PodReady &&
						(condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown) {

						ready = false
						break
					}
				}
			}
			return updateClientBackgroundPodsStatus(r.Client, ctx, session, client, ready)

		} else {
			// some or all background pods are missing
			if err := updateClientBackgroundPodsStatus(r.Client, ctx, session, client, false); err != nil {
				return err
			}
			return restoreBackgroundPods(r.Client, r.Scheme, ctx, namespace, session, client, &backgroundPods)
		}
	}

	return nil
}

func gargabageCollector() {

}

func updateClientBackgroundPodsStatus(
	k8sclient client.Client,
	ctx context.Context,
	session *telepresencev1.Session,
	client string,
	ready bool) error {

	logger := log.FromContext(ctx)
	var toUpdate *telepresencev1.ClientBackgroundPodsStatus

	for _, cl := range session.Status.Clients {
		if cl.Client == client {
			toUpdate = &cl
		}
	}

	if toUpdate == nil {
		session.Status.Clients = append(session.Status.Clients,
			telepresencev1.ClientBackgroundPodsStatus{Client: client, Ready: ready})

	} else {
		toUpdate.Ready = ready
	}

	if err := k8sclient.Status().Update(ctx, session); err != nil {
		logger.Error(err, "unable to update client background pods status", "session", session.Name, "client", client)
		return err
	}
	return nil
}

func restoreBackgroundPods(
	k8sclient client.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session,
	client string,
	foundPods *corev1.PodList) error {

	if len(foundPods.Items) == 0 {
		// all background pods are missing
		for _, pod := range session.Spec.BackgroundPods {
			if err := spawnBackgroundPod(k8sclient, scheme, ctx, namespace, session, client, &pod); err != nil {
				return err
			}
		}
	} else {
		// lets find out which pods are missing
		for _, pod := range session.Spec.BackgroundPods {
			found := false

			for _, foundPod := range foundPods.Items {
				if pod.Name == foundPod.Name {
					found = true
					break
				}
			}

			if !found {
				// the pod wasn't found, lets spawn it
				if err := spawnBackgroundPod(k8sclient, scheme, ctx, namespace, session, client, &pod); err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func spawnBackgroundPod(
	k8sclient client.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session,
	client string,
	podToSpawn *telepresencev1.BackgroundPod) error {

	logger := log.FromContext(ctx)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      session.Name + "-" + podToSpawn.Name + "-" + "client", // e.g. session1-renderfusion-johndoe
			Namespace: namespace,
			Labels:    map[string]string{"purpose": "background"}, // label used for matching the selector in background-headless-service.yaml
		},
		Spec: podToSpawn.PodSpec,
	}

	// set controller reference for garbage collection
	if err := ctrl.SetControllerReference(session, &pod, scheme); err != nil {
		logger.Error(err, "unable to set controller reference for background pod", "session", session.Name,
			"client", client, "pod", pod)

		return err
	}

	if err := k8sclient.Create(ctx, &pod); err != nil {
		logger.Error(err, "unable to create background pod for client", "session", session.Name, "client", client,
			"pod", pod)

		return err
	}
	return nil
}
