package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	telepresencev1 "mr.telepresence/controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConditionType string

const (
	TypeReady ConditionType = "Ready"
)

type ConditionsAware interface {
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
}

func (r *SessionReconciler) setCondition(ctx context.Context, session *telepresencev1.Session, conditionType ConditionType,
	status metav1.ConditionStatus, reason string, message string) error {

	if !containsCondition(session, reason) {
		return appendCondition(ctx, r.Client, session, conditionType, status, reason, message)
	}
	return nil
}

func containsCondition(session *telepresencev1.Session, reason string) bool {

	output := false
	for _, condition := range session.Status.Conditions {
		if condition.Reason == reason {
			output = true
		}
	}
	return output
}

func appendCondition(ctx context.Context, client client.Client, object client.Object, conditionType ConditionType,
	status metav1.ConditionStatus, reason string, message string) error {

	conditionsAware, conversionSuccessful := (object).(ConditionsAware) // type assertion

	if conversionSuccessful {
		condition := metav1.Condition{
			Type:               string(conditionType),
			Status:             status,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: metav1.Time{Time: time.Now()},
		}
		conditionsAware.SetConditions(append(conditionsAware.GetConditions(), condition))

		return client.Status().Update(ctx, object)
	}
	return fmt.Errorf("status cannot be set, resource doesn't support conditions")
}
