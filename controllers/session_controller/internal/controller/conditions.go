package controller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	telepresencev1 "mr.telepresence/controller/api/v1"
)

type ConditionType string

const (
	TYPE_READY ConditionType = "Ready"
)

const RESOURCE_NOT_FOUND_REASON = "ResourceNotFound"
const RESOURCE_NOT_FOUND_MESSAGE = "Failed to get session resource"

const FAILED_GET_SESSION_PODS_REASON = "FailedGetSessionPods"
const FAILED_GET_SESSION_PODS_MESSAGE = "Failed to get the session pods"

const SESSION_PODS_READY_REASON = "SessionPodsReady"
const SESSION_PODS_READY_MESSAGE = "All the session pods present the ready contidion set to true"

const SESSION_PODS_NOT_READY_REASON = "SessionPodsNotReady"
const SESSION_PODS_NOT_READY_MESSAGE = "At least one session pod presents the ready contidion set to false"

func (r *SessionReconciler) SetReadyCondition(
	ctx context.Context,
	session *telepresencev1.Session,
	status metav1.ConditionStatus,
	reason string,
	message string) {

	index := containsCondition(session, TYPE_READY)

	if index != -1 {
		removeConditionAtIndex(session, index)
		appendCondition(session, TYPE_READY, status, reason, message)
	} else {
		appendCondition(session, TYPE_READY, status, reason, message)
	}
}

func containsCondition(session *telepresencev1.Session, conditionType ConditionType) int {
	index := -1

	for i, condition := range session.Status.Conditions {
		if condition.Type == string(conditionType) {
			index = i
		}
	}
	return index
}

func removeConditionAtIndex(session *telepresencev1.Session, index int) {
	session.Status.Conditions = append(session.Status.Conditions[:index], session.Status.Conditions[index+1:]...)
}

func appendCondition(
	session *telepresencev1.Session,
	conditionType ConditionType,
	status metav1.ConditionStatus,
	reason string,
	message string) {

	condition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Time{Time: time.Now()},
	}

	session.Status.Conditions = append(session.Status.Conditions, condition)
}
