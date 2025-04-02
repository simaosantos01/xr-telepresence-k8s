package garbagecollector

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	event "k8s.io/api/events/v1"
	"k8s.io/client-go/tools/record"
	telepresencev1 "mr.telepresence/controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FreeWorkload(
	ctx context.Context,
	rClient client.Client,
	recorder record.EventRecorder,
	session *telepresencev1.Session,
	sessionPods *corev1.PodList,
) (time.Duration, error) {

	duration := time.Duration(0)

	var events event.EventList
	fieldSelector := client.MatchingFields{"eventRegardingField": session.Name}
	if err := rClient.List(ctx, &events, fieldSelector); err != nil {
		return duration, err
	}

	sessionScopedEvent, _ := extractEvents(events.Items)

	numOfClients := len(session.Spec.Clients)

	if numOfClients != 0 && sessionScopedEvent.Name != "" {
		// the session scoped event exists but there are clients in the session

		if err := rClient.Delete(ctx, &sessionScopedEvent); err != nil {
			return duration, err
		}

	} else if numOfClients == 0 && sessionScopedEvent.Name == "" {
		// no clients in the session and the session scoped event does not exist

		recorder.Event(session, "Normal", "NoClientsInSession", "Test")

		duration := time.Duration(session.Spec.TimeoutSeconds)
		return duration, nil

	} else if numOfClients == 0 && expired(&sessionScopedEvent, session.Spec.TimeoutSeconds) {
		// no clients in the session and the timeout is reached

		for _, pod := range sessionPods.Items {
			if err := rClient.Delete(ctx, &pod); err != nil {
				return duration, err
			}
		}

		if err := rClient.Delete(ctx, &sessionScopedEvent); err != nil {
			return duration, err
		}
	}

	return duration, nil
}

func extractEvents(events []event.Event) (event.Event, []event.Event) {
	sessionScopedEvent := event.Event{}
	podScopedEvents := []event.Event{}

	for _, event := range events {
		if event.Reason == "NoClientsInSession" {
			sessionScopedEvent = event
		} else {
			podScopedEvents = append(podScopedEvents, event)
		}
	}

	return sessionScopedEvent, podScopedEvents
}

func expired(event *event.Event, timeoutSeconds int) bool {
	expiryTime := event.EventTime.Time.Add(time.Duration(timeoutSeconds) * time.Second)
	return time.Now().After(expiryTime)
}
