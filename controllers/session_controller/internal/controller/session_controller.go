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
	if err := r.Get(ctx, req.NamespacedName, &session); err != nil && errors.IsNotFound(err) {
		return ctrl.Result{}, nil

	} else if err != nil {
		logger.Error(err, "unable to get session resource")
		return ctrl.Result{}, err
	}

	//sessionSnapshot := session.DeepCopy()

	// if err := r.ReconcileSessionServices(ctx, req.Namespace, &session); err != nil {
	// 	r.Status().Update(ctx, &session)
	// 	return ctrl.Result{}, err
	// }

	// if err := r.ReconcileBackgroundPods(ctx, req.Namespace, &session); err != nil {
	// 	r.Status().Update(ctx, &session)
	// 	return ctrl.Result{}, err
	// }

	// if err := r.Status().Update(ctx, &session); err != nil {
	// 	logger.Error(err, "undable to update sessio status", "session", session.Name)
	// 	return ctrl.Result{}, err
	// }

	// if sessionSnapshot.Status != session.Status {

	// }

	return ctrl.Result{}, nil
}

var (
	ownerKey  = "controllerReference"
	clientKey = "clientAnnotation"
	apiGVStr  = telepresencev1.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, ownerKey,
		func(o client.Object) []string {

			return IndexSessionPods(o)
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, clientKey,
		func(o client.Object) []string {

			return IndexBackgroundPods(o)
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&telepresencev1.Session{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Named("session").
		Complete(r)
}
