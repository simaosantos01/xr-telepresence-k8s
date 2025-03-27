package controller

// import (
// 	"context"

// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	telepresencev1 "mr.telepresence/controller/api/v1"
// 	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
// 	"sigs.k8s.io/controller-runtime/pkg/log"
// )

// func (r *SessionReconciler) ReconcileSessionServices(
// 	ctx context.Context,
// 	namespace string,
// 	session *telepresencev1.Session) error {

// 	logger := log.FromContext(ctx)

// 	var sessionPods corev1.PodList
// 	if err := r.List(ctx, &sessionPods, ctrlClient.InNamespace(namespace),
// 		ctrlClient.MatchingFields{ownerKey: session.Name}); err != nil {

// 		r.SetReadyCondition(ctx, session, metav1.ConditionUnknown, FAILED_GET_SESSION_PODS_REASON,
// 			FAILED_GET_SESSION_PODS_MESSAGE)

// 		logger.Error(err, "unable to get session pods", "session", session.Name)
// 		return err
// 	}

// 	if len(session.Spec.SessionPods) == len(sessionPods.Items) {
// 		// all session pods exist, lets verify if all pods are ready
// 		ready := PodsAreReady(&sessionPods)

// 		if ready {
// 			r.SetReadyCondition(ctx, session, metav1.ConditionTrue, SESSION_PODS_READY_REASON,
// 				SESSION_PODS_READY_MESSAGE)
// 		} else {
// 			r.SetReadyCondition(ctx, session, metav1.ConditionFalse, SESSION_PODS_NOT_READY_REASON,
// 				SESSION_PODS_NOT_READY_MESSAGE)
// 		}
// 	} else {
// 		r.SetReadyCondition(ctx, session, metav1.ConditionFalse, SESSION_PODS_NOT_READY_REASON,
// 			SESSION_PODS_NOT_READY_MESSAGE)

// 		objectMeta := metav1.ObjectMeta{
// 			Namespace: namespace,
// 			// label used for matching the selector in background-headless-service.yaml
// 			Labels: map[string]string{"purpose": "session"},
// 		}

// 		if err := RestorePods(r.Client, r.Scheme, ctx, session, &sessionPods, session.Spec.SessionPods, &objectMeta,
// 			nil); err != nil {

// 			return err
// 		}
// 	}

// 	return nil
// }
