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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"mr.telepresence/controller/internal/controller/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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
	if err := r.Get(ctx, req.NamespacedName, &session); err != nil && errors.IsNotFound(err) {
		return ctrl.Result{}, nil

	} else if err != nil {
		logger.Error(err, "unable to get session resource")
		return ctrl.Result{}, err
	}

	statusSnapshot := session.Status.DeepCopy()

	if len(session.Spec.Clients) != 0 && len(session.Spec.SessionServices) > 0 {
		if err := r.ReconcileSessionPods(ctx, req.Namespace, &session); err != nil {
			r.Status().Update(ctx, &session)
			return ctrl.Result{}, err
		}
	}

	var clientPodsToSpawn []telepresencev1.Pod
	if len(session.Spec.ClientServices) > 0 {
		var err error
		clientPodsToSpawn, err = r.ReconcileClientPods(ctx, req.Namespace, &session)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if StatusHasChanged(statusSnapshot, &session.Status) {
		if err := r.Status().Update(ctx, &session); err != nil {
			logger.Error(err, "unable to update session resource")
			return ctrl.Result{}, err
		}
	}

	for _, pod := range clientPodsToSpawn {
		if err := utils.SpawnPod(ctx, r.Client, r.Scheme, &session, &pod); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func StatusHasChanged(oldStatus *telepresencev1.SessionStatus, newStatus *telepresencev1.SessionStatus) bool {
	if len(oldStatus.Clients) != len(newStatus.Clients) || len(oldStatus.Conditions) != len(newStatus.Conditions) {
		return true
	}

	oldConditionsMap := make(map[utils.ConditionType]metav1.ConditionStatus)
	for _, condition := range oldStatus.Conditions {
		oldConditionsMap[utils.ConditionType(condition.Type)] = condition.Status
	}

	for _, condition := range newStatus.Conditions {
		val, ok := oldConditionsMap[utils.ConditionType(condition.Type)]

		if !ok || val != condition.Status {
			return true
		}
	}

	return true // TODO: always updating the status!
}

func IndexPodByOwner(obj client.Object) []string {
	owner := metav1.GetControllerOf(obj)

	if owner == nil {
		return nil
	}

	if owner.APIVersion != apiGVStr || owner.Kind != "Session" {
		return nil
	}

	return []string{owner.Name}
}

func IndexPodByType(obj client.Object) []string {
	pod := obj.(*corev1.Pod)

	podType, ok := pod.Labels["type"]
	if !ok {
		return nil
	}

	return []string{podType}
}

var (
	podOwnerField = "ownerField"
	podTypeField  = "podTypeField"
	apiGVStr      = telepresencev1.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podOwnerField,
		func(o client.Object) []string {

			return IndexPodByOwner(o)
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podTypeField,
		func(o client.Object) []string {

			return IndexPodByType(o)
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&telepresencev1.Session{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool { return true },
			DeleteFunc: func(e event.DeleteEvent) bool { return false },
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldObj := e.ObjectOld.(*telepresencev1.Session)
				newObj := e.ObjectNew.(*telepresencev1.Session)

				if len(oldObj.Spec.Clients) != len(newObj.Spec.Clients) {
					return true
				}

				clientMap := make(map[string]bool, len(oldObj.Spec.Clients))
				for _, client := range oldObj.Spec.Clients {
					clientMap[client.Id] = client.Connected
				}

				for _, client := range newObj.Spec.Clients {
					value, ok := clientMap[client.Id]

					if !ok || value != client.Connected {
						return true
					}
				}

				return false
			},
		})).
		Owns(&corev1.Pod{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool { return false },
			DeleteFunc: func(e event.DeleteEvent) bool { return true },
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldObj := e.ObjectOld.(*corev1.Pod)
				newObj := e.ObjectNew.(*corev1.Pod)

				return utils.ExtractReadyConditionStatusFromPod(oldObj) !=
					utils.ExtractReadyConditionStatusFromPod(newObj)
			},
		})).
		Named("session").
		Complete(r)
}
