package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *SessionReconciler) ReconcileBackgroundPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	logger := log.FromContext(ctx)

	// deletes pods tied to non existing clients
	if err := gargabageCollection(r.Client, ctx, namespace, session); err != nil {
		return err
	}

	for _, client := range session.Spec.Clients {

		var backgroundPods corev1.PodList
		if err := r.List(ctx, &backgroundPods, ctrlClient.InNamespace(namespace),
			ctrlClient.MatchingFields{clientKey: client}); err != nil {

			logger.Error(err, "unable to get client background pods", "session", session.Name, "client", client)
			return err
		}

		if len(session.Spec.BackgroundPods) == len(backgroundPods.Items) {
			// all background pods exist, lets verify if all pods are ready
			ready := PodsAreReady(&backgroundPods)

			if err := updateClientBackgroundPodsStatus(r.Client, ctx, session, client, ready); err != nil {
				return err
			}
		} else {
			// some or all background pods are missing
			if err := updateClientBackgroundPodsStatus(r.Client, ctx, session, client, false); err != nil {
				return err
			}

			objectMeta := metav1.ObjectMeta{
				Namespace: namespace,
				// label used for matching the selector in background-headless-service.yaml
				Labels:      map[string]string{"purpose": "background"},
				Annotations: map[string]string{"client": client},
			}

			if err := RestorePods(r.Client, r.Scheme, ctx, namespace, session, &backgroundPods, session.Spec.BackgroundPods,
				&objectMeta, &client); err != nil {

				return err
			}
		}
	}

	return nil
}

func gargabageCollection(
	k8sclient ctrlClient.Client,
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	clientExistsInSpec := func(client string, clients []string) bool {
		for _, cl := range clients {
			if cl == client {
				return true
			}
		}

		return false
	}

	logger := log.FromContext(ctx)

	var toRemoveFromStatus []int // positions to be removed from the session.status.clients
	var removedClients []string  // name of the clients that have been removed

	for i, client := range session.Status.Clients {
		if !clientExistsInSpec(client.Client, session.Spec.Clients) {
			toRemoveFromStatus = append(toRemoveFromStatus, i)
			removedClients = append(removedClients, client.Client)
		}
	}

	for _, index := range toRemoveFromStatus {
		session.Status.Clients = append(session.Status.Clients[:index], session.Status.Clients[index+1:]...)
	}

	if err := k8sclient.Status().Update(ctx, session); err != nil {
		logger.Error(err, "unable to update client background pods status", "session", session.Name)
		return err
	}

	for _, client := range removedClients {
		var backgroundPods corev1.PodList
		if err := k8sclient.List(ctx, &backgroundPods, ctrlClient.InNamespace(namespace),
			ctrlClient.MatchingFields{clientKey: client}); err != nil {

			logger.Error(err, "unable to get client background pods", "session", session.Name, "client", client)
			return err
		}

		for i := range backgroundPods.Items {
			if err := k8sclient.Delete(ctx, &backgroundPods.Items[i]); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "unable to delete client background pods", "session", session.Name, "client", client)
				return err
			}
		}
	}

	return nil
}

func updateClientBackgroundPodsStatus(
	k8sclient ctrlClient.Client,
	ctx context.Context,
	session *telepresencev1.Session,
	client string,
	ready bool) error {

	logger := log.FromContext(ctx)
	var toUpdate *telepresencev1.ClientBackgroundPodsStatus

	for i := range session.Status.Clients {
		if session.Status.Clients[i].Client == client {
			toUpdate = &session.Status.Clients[i]
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
