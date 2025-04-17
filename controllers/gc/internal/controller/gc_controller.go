package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type GCReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.mr.telepresence,resources=sessions,verbs=get;list;watch

func (r *GCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("controller triggered", "name", req.Name, "namespace", req.Namespace)

	now := time.Now()
	requeueAfter := time.Second * time.Duration(requeueAfterSec)

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

var (
	requeueAfterSec = 10
)

func (r *GCReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Event{}, eventRegardingField,
		func(o client.Object) []string {
			event := o.(*v1.Event)
			return []string{event.Regarding.Kind}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&v1.Event{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{
					Name: obj.GetName(), Namespace: obj.GetNamespace()}}}
			}),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc:  func(e event.CreateEvent) bool { return true },
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				UpdateFunc:  func(e event.UpdateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Named("gc").
		Complete(r)
}
