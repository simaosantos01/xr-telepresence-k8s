package utils

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	telepresencev1alpha1 "mr.telepresence/session/api/v1alpha1"
)

type ConditionType string

const (
	TYPE_READY ConditionType = "Ready"
)

const GET_PODS_FAILED_REASON = "FailedGetSessionPods"
const GET_PODS_FAILED_MESSAGE = "Failed to get the session pods"

const GET_SVC_FAILED_REASON = "FailedGetSessionServices"
const GET_SVC_FAILED_MESSAGE = "Failed to get the session services"

const PODS_READY_REASON = "SessionPodsReady"
const PODS_READY_MESSAGE = "All the session pods present the ready contidion set to true"

const PODS_NOT_READY_REASON = "SessionPodsNotReady"
const PODS_NOT_READY_MESSAGE = "At least one session pod presents the ready contidion set to false"

const PODS_RECONCILED_REASON = "PodsHaveBeenReconciled"
const PODS_RECONCILED_MESSAGE = "Pods have been reconciled successfully"

func SetReadyCondition(
	session *telepresencev1alpha1.Session,
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

func containsCondition(session *telepresencev1alpha1.Session, conditionType ConditionType) int {
	index := -1

	for i, condition := range session.Status.Conditions {
		if condition.Type == string(conditionType) {
			index = i
		}
	}
	return index
}

func removeConditionAtIndex(session *telepresencev1alpha1.Session, index int) {
	session.Status.Conditions = append(session.Status.Conditions[:index], session.Status.Conditions[index+1:]...)
}

func appendCondition(
	session *telepresencev1alpha1.Session,
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
