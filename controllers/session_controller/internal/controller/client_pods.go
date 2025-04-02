package controller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	telepresencev1 "mr.telepresence/controller/api/v1"
)

var defaultTime = metav1.Time{}

func (r *SessionReconciler) ReconcileClientPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	clientStatusMap := make(map[string]telepresencev1.ClientStatus, len(session.Status.Clients))
	for _, client := range session.Status.Clients {
		if !clientHasExpired(client.LastSeen, session.Spec.TimeoutSeconds) {
			clientStatusMap[client.Client] = client
		}
	}

	newClients := []string{}
	now := metav1.NewTime(time.Now())
	clientSpecSet := make(map[string]struct{})

	for _, client := range session.Spec.Clients {
		value, ok := clientStatusMap[client.Id]

		clientSpecSet[client.Id] = struct{}{}

		if ok && !client.Connected && value.LastSeen == defaultTime {
			// client lost connection
			value.LastSeen = now
		} else if !ok && client.Connected {
			// new client found
			newClients = append(newClients, client.Id)
		} else if client.Connected {
			// client may have reconected
			value.LastSeen = defaultTime
		}
	}

	// remove clients from status that are not present in spec
	for k, _ := range clientStatusMap {
		if _, ok := clientSpecSet[k]; !ok {
			delete(clientStatusMap, k)
		}
	}

	return nil
}

func clientHasExpired(lastSeen metav1.Time, timeoutSeconds int) bool {
	return true
}
