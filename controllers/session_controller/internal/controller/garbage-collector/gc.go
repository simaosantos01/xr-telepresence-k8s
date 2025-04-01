package garbagecollector

import (
	"context"

	event "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func FreeWorkload(ctx context.Context, k8sclient ctrlClient.Client, session *telepresencev1.Session) error {
	var noClientsEvent *event.Event
	namespacedName := types.NamespacedName{Name: "no-clients", Namespace: "default"}

	if err := k8sclient.Get(ctx, namespacedName, noClientsEvent); err != nil && !errors.IsNotFound(err) {
		return err

	} else if err != nil {
		noClientsEvent = nil
	}

	numOfClients := len(session.Spec.Clients)

	if numOfClients != 0 && noClientsEvent != nil {
		deleteEvent("no-clients")

	} else if numOfClients == 0 && noClientsEvent == nil {
		createEvent("no-clients", session)

	} else if numOfClients == 0 && expired(noClientsEvent, session.Spec.TimeoutSeconds) {
		// clean up everyhing and return
	}

	return nil
}

func createEvent(name string, session *telepresencev1.Session) error {
	return nil
}

func deleteEvent(name string) error {
	return nil
}

func expired(event *event.Event, timeSeconds int) bool {
	return true
}
