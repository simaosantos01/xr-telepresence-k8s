package controller

import (
	"context"
	"time"

	"k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	telepresencev1alpha1 "mr.telepresence/session/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	var eventList v1.EventList
	if err := r.List(ctx, &eventList); err != nil {
		logger.Error(err, "unable to get event resources")
		return ctrl.Result{}, err
	}

	sessionMap := make(map[string]*telepresencev1alpha1.Session)

	for _, event := range eventList.Items {
		sessionName := event.Regarding.Name
		sessionNamespace := event.Regarding.Namespace
		var session telepresencev1alpha1.Session

		if _, ok := sessionMap[sessionName]; !ok {
			if err := r.Get(ctx, types.NamespacedName{Namespace: sessionNamespace, Name: sessionName}, &session); err != nil {
				logger.Error(err, "unable to get session resource")
				return ctrl.Result{}, err
			}

			sessionMap[sessionName] = &session
		} else {
			session = *sessionMap[sessionName]
		}

		if now.Sub(event.CreationTimestamp.Time) > time.Second * time.Duration(session.Spec.TimeoutSeconds) {
			
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

var (
	requeueAfterSec = 10
	regardingField  = "regarding"
)

func (r *GCReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Event{}, regardingField,
		func(o client.Object) []string {
			event := o.(*v1.Event)
			return []string{event.Regarding.Kind}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&telepresencev1alpha1.Session{},
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
