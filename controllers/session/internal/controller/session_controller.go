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
	"k8s.io/apimachinery/pkg/runtime"
	gcv1alpha1 "mr.telepresence/gc/api/v1alpha1"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	"mr.telepresence/session/internal/controller/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SessionReconciler reconciles a Session object
type SessionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.mr.telepresence,resources=sessions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.mr.telepresence,resources=sessions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.mr.telepresence,resources=sessions/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.mr.telepresence,resources=gcregistrations,verbs=list;create;delete

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

	var session sessionv1alpha1.Session
	if err := r.Get(ctx, req.NamespacedName, &session); err != nil && errors.IsNotFound(err) {
		return ctrl.Result{}, nil

	} else if err != nil {
		logger.Error(err, "unable to get session resource")
		return ctrl.Result{}, err
	}

	var gcRegistrations gcv1alpha1.GCRegistrationList
	opts := client.MatchingFields{utils.GCRegistrationSessionField: session.Name}

	if err := r.List(ctx, &gcRegistrations, client.InNamespace(gcNamespace), opts); err != nil {
		logger.Error(err, "unable to get GC registrations", "session", session.Name)
		return ctrl.Result{}, err
	}

	if len(session.Spec.Clients) != 0 && len(session.Spec.SessionPodTemplates.Items) > 0 {
		if err := r.ReconcileSessionPods(ctx, req.Namespace, &session); err != nil {
			r.Status().Update(ctx, &session)
			return ctrl.Result{}, err
		}
	}

	var clientPodsToSpawn []corev1.Pod
	if len(session.Spec.ClientPodTemplates.Items) > 0 {
		var err error
		clientPodsToSpawn, err = r.ReconcileClientPods(ctx, req.Namespace, &session, gcRegistrations.Items)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	oldStatusHash := session.Annotations["statusHash"]
	newStatusHash := utils.HashStatus(&session.Status)

	if utils.StatusHasChanged(oldStatusHash, newStatusHash) {
		utils.SetStatusHashAnnotation(newStatusHash, &session)
		errStr := "unable to update session resource"

		if err := r.Update(ctx, &session); err != nil {
			logger.Error(err, errStr, "session", session.Name)
			return ctrl.Result{}, err
		}

		if err := r.Status().Update(ctx, &session); err != nil {
			logger.Error(err, errStr, "session", session.Name)
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

const (
	gcNamespace = "mr.telepresence.gc" // Namespace that holds the GC Registrations
)

// SetupWithManager sets up the controller with the Manager.
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := utils.SetupManagerFieldIndexer(context.Background(), mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&sessionv1alpha1.Session{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool { return true },
			DeleteFunc: func(e event.DeleteEvent) bool { return false },
			UpdateFunc: func(e event.UpdateEvent) bool { return utils.SessionUpdateFunc(e) },
		})).
		Owns(&corev1.Pod{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool { return false },
			DeleteFunc: func(e event.DeleteEvent) bool { return true },
			UpdateFunc: func(e event.UpdateEvent) bool { return utils.PodUpdateFunc(e) },
		})).
		Named("session").
		Complete(r)
}
