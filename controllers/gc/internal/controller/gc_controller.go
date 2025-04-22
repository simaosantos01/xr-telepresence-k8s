package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	gcv1alpha1 "mr.telepresence/gc/api/v1alpha1"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type GCReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.mr.telepresence,resources=gcregistrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.mr.telepresence,resources=sessions,verbs=get
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;delete

func (r *GCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("controller triggered", "name", req.Name, "namespace", req.Namespace)

	var gcRegistrations gcv1alpha1.GCRegistrationList
	if err := r.List(ctx, &gcRegistrations, client.InNamespace(namespace)); err != nil {
		logger.Error(err, "unable to get gc registrations")
		return ctrl.Result{}, err
	}

	for _, registration := range gcRegistrations.Items {
		var session sessionv1alpha1.Session
		namespacedName := types.NamespacedName{Namespace: registration.Spec.Session.Namespace,
			Name: registration.Spec.Session.Name}

		if err := r.Get(ctx, namespacedName, &session); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "unable to get session resource")
			return ctrl.Result{}, err

		} else if err == nil {
			var pod corev1.Pod
			namespacedName := types.NamespacedName{Namespace: registration.Spec.Session.Namespace, Name: registration.Name}

			if err := r.Get(ctx, namespacedName, &pod); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "unable to get pod")
				return ctrl.Result{}, err

			} else if err != nil {
				// it means that the pod does not exist and the registration exists
				if err := r.Delete(ctx, &registration); err != nil && !errors.IsNotFound(err) {
					logger.Error(err, "unable to delete registration")
					return ctrl.Result{}, err
				}
			} else if registrationHasExpired(&registration, &session) {
				// it means that the pod exists and the registration expired
				if err := r.handleExpiredRegistration(ctx, &registration, &pod, &session); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{RequeueAfter: time.Second * time.Duration(requeueAfterSec)}, nil
}

func (r *GCReconciler) handleExpiredRegistration(
	ctx context.Context,
	registration *gcv1alpha1.GCRegistration,
	pod *corev1.Pod,
	session *sessionv1alpha1.Session,
) error {
	logger := log.FromContext(ctx)

	if registration.Spec.Type == gcv1alpha1.ReutilizeTimeout ||
		(registration.Spec.Type == gcv1alpha1.Timeout && session.Spec.ReutilizeTimeoutSeconds == 0) {

		if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "unable to delete pod")
			return err
		}
		if err := r.Delete(ctx, registration); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "unable to delete registration")
			return err
		}
	} else {
		registration.Spec.Type = gcv1alpha1.ReutilizeTimeout
		if err := r.Update(ctx, registration); err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "unable to update registration spec")
			return err
		}
	}
	return nil
}

func registrationHasExpired(registration *gcv1alpha1.GCRegistration, session *sessionv1alpha1.Session) bool {
	now := time.Now()

	timeoutDuration := time.Second * time.Duration(session.Spec.TimeoutSeconds)

	if registration.Spec.Type == gcv1alpha1.Timeout {
		return registration.ObjectMeta.CreationTimestamp.Add(timeoutDuration).Before(now)

	} else {
		reutilizeTimeoutDuration := time.Second * time.Duration(session.Spec.ReutilizeTimeoutSeconds)
		reutilizeTimeoutDuration += timeoutDuration
		return registration.ObjectMeta.CreationTimestamp.Add(reutilizeTimeoutDuration).Before(now)
	}
}

const (
	requeueAfterSec = 10
	namespace       = "mr-telepresence-gc"
)

func (r *GCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcv1alpha1.GCRegistration{}).
		Named("gc").
		Complete(r)
}
