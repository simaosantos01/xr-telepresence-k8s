package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

type NetworkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("controller triggered", "name", req.Name, "namespace", req.Namespace)

	var pods corev1.PodList
	labelSelector := client.MatchingLabels{"telepresence": "true"}
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace), labelSelector); err != nil {
		logger.Error(err, "unable to get pods")
		return ctrl.Result{}, err
	}

	var services corev1.ServiceList
	if err := r.List(ctx, &services, client.InNamespace(req.Namespace), labelSelector); err != nil {
		logger.Error(err, "unable to get services")
		return ctrl.Result{}, err
	}

	servicesMap := make(map[string]*corev1.Service, len(services.Items))
	for _, service := range services.Items {
		servicesMap[service.Name] = &service
	}

	if err := r.spawnServices(ctx, req.Namespace, pods.Items, servicesMap); err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileIngress(ctx, servicesMap)
}

func (r *NetworkReconciler) spawnServices(
	ctx context.Context,
	namespace string,
	pods []corev1.Pod,
	servicesMap map[string]*corev1.Service,
) error {

	for _, pod := range pods {
		key := pod.Name + "-svc"

		if _, ok := servicesMap[key]; !ok { // the pod does not have a corresponding service
			service, err := r.spawnService(ctx, namespace, &pod)

			if err != nil && !errors.IsAlreadyExists(err) {
				return err
			}

			servicesMap[key] = service
		}
	}
	return nil
}

func (r *NetworkReconciler) spawnService(
	ctx context.Context,
	namespace string,
	forPod *corev1.Pod,
) (*corev1.Service, error) {

	logger := log.FromContext(ctx)
	servicePorts := []corev1.ServicePort{}

	for _, container := range forPod.Spec.Containers {
		for _, port := range container.Ports {
			servicePorts = append(servicePorts, corev1.ServicePort{
				Protocol: port.Protocol, Port: port.ContainerPort, Name: port.Name,
			})
		}
	}

	// allocate default port (services require at least one port)
	if len(servicePorts) == 0 {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Protocol: corev1.ProtocolTCP, Port: 8080, Name: "default"})
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      forPod.Name + "-svc",
			Namespace: namespace,
			Labels:    map[string]string{"telepresence": "true"},
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: map[string]string{"svc": forPod.Name},
		},
	}

	if err := ctrl.SetControllerReference(forPod, service, r.Scheme); err != nil {
		logger.Error(err, "unable to set owner reference in service")
		return nil, err
	}

	if err := r.Create(ctx, service); err != nil {
		logger.Error(err, "unable to create service")
		return nil, err
	}
	return service, nil
}

func handleEvent(obj client.Object) []reconcile.Request {
	if val, ok := obj.GetLabels()["telepresence"]; ok && val == "true" {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
				},
			},
		}
	}
	return []reconcile.Request{}
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return handleEvent(obj)
			}),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				DeleteFunc:  func(e event.DeleteEvent) bool { return true },
				UpdateFunc:  func(e event.UpdateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return handleEvent(obj)
			}),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc:  func(e event.CreateEvent) bool { return true },
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				UpdateFunc:  func(e event.UpdateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Named("network").
		Complete(r)
}
