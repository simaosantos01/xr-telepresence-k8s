/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	telepresencev1 "mr.telepresence/controller/api/v1"
)

// SessionReconciler reconciles a Session object
type SessionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=telepresence.mr.telepresence,resources=sessions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=telepresence.mr.telepresence,resources=sessions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=telepresence.mr.telepresence,resources=sessions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Session object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("controller triggered", "name", req.Name, "namespace", req.Namespace)

	var session telepresencev1.Session
	if err := r.Get(ctx, req.NamespacedName, &session); err != nil {
		logger.Error(err, "unable to get session resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle pod creation and verify session readiness

	constructSessionPods := func(session *telepresencev1.Session, ctx context.Context, req *ctrl.Request) (*[]corev1.Pod, bool, error) {
		var result []corev1.Pod // list of pods to create

		var sessionPods corev1.PodList
		if err := r.List(ctx, &sessionPods, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: session.Name}); err != nil {
			log.Log.Error(err, "unable to get session pods")
			return nil, false, err
		}

		// Although all the pods do exist the session state may not be ready
		if len(sessionPods.Items) == len(session.Spec.Services.Items) {
			sessionIsReady := true

			for _, pod := range sessionPods.Items { // todo: check pod ready condition
				if pod.Status.Phase != corev1.PodRunning {
					return &result, false, nil
				}
			}

			if err := r.setCondition(ctx, session, TypeReady, metav1.ConditionTrue, "AllPodsRunning", "Session is ready to accept clients"); err != nil {

			}

			return &result, sessionIsReady, nil
		}

		for i, service := range session.Spec.Services.Items {
			podName := session.Name + "-" + service.Template.Spec.Containers[0].Name
			found := false

			for _, pod := range sessionPods.Items {
				if pod.Name == podName {
					found = true
				}
			}

			if !found {
				log.Log.Info("constructing pod", "session", session.Name, "pod", podName)
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{"service": service.Template.Spec.Containers[0].Name},
						Annotations: make(map[string]string),
						Name:        podName,
						Namespace:   session.Namespace,
					},
					Spec: *session.Spec.Services.Items[i].Template.Spec.DeepCopy(),
				}

				pod.Spec.Hostname = podName
				pod.Spec.Subdomain = "clientpolling"

				if err := ctrl.SetControllerReference(session, pod, r.Scheme); err != nil {
					return nil, false, err
				}

				result = append(result, *pod)
			}
		}

		return &result, false, nil
	}

	podsToCreate, sessionIsReady, err := constructSessionPods(&session, ctx, &req)

	if err != nil {
		logger.Error(err, "unable to construct session pods")
		return ctrl.Result{}, err
	}

	for _, pod := range *podsToCreate {
		if err := r.Create(ctx, &pod); err != nil {
			logger.Error(err, "unable to create pod", "session", session.Name, "pod", pod.Name)
			return ctrl.Result{}, err
		}
		logger.Info("pod created", "session", session.Name, "pod", pod.Name)
	}

	// Evaluate if session must be deleted

	if sessionIsReady {
		delete, clientCount, err := r.evaluateSession(&session)
		logger.Info("client count", "count", clientCount)
		if err != nil {
			logger.Error(err, "unable to evaluate session", "session", session.Name)
			return ctrl.Result{}, err

		} else if delete {
			logger.Info("deleting session", "session", session.Name)

			if err := r.Delete(ctx, &session); err != nil {
				logger.Error(err, "unable to delete session", "session", session.Name)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		} else {
			// update 'ClientCount' status field
			session.Status.ClientCount = clientCount

			if err := r.Client.Status().Update(ctx, &session); err != nil {
				logger.Error(err, "unable to update status field 'ClientCount'", "session", session.Name)
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil
}

var (
	ownerKey = ".metadata.controller"
	apiGVStr = telepresencev1.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, ownerKey, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)

		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Session" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&telepresencev1.Session{}).
		Owns(&corev1.Pod{}).
		Named("session").
		Complete(r)
}
