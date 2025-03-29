package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
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

	var ingress netv1.Ingress
	namespacedName := types.NamespacedName{Name: "ingress", Namespace: "default"}
	ingressScaffolded := false
	ingressUpdated := false
	if err := r.Client.Get(ctx, namespacedName, &ingress); err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "unable to get ingress resource")
		return ctrl.Result{}, err

	} else if err != nil {
		ingressScaffolded = true
		ingress = *scaffoldIngress()
	}

	podsMap := make(map[string]struct{}, len(pods.Items))
	servicesMap := make(map[string]struct{}, len(services.Items))

	for _, pod := range pods.Items {
		podsMap[pod.Name] = struct{}{}
	}

	for _, service := range services.Items {
		key := service.Name[:len(service.Name)-3] // trim the "-svc" from the service name to match the pod name

		if _, ok := podsMap[key]; !ok { // service exists but does not match any pod, lets delete it

			if err := r.Delete(ctx, &service); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "unable to delete service", "service", service)
				return ctrl.Result{}, err
			}

			if !ingressUpdated {
				ingressUpdated = deleteFromIngress(&ingress, &service)
			}

		} else {
			servicesMap[service.Name] = struct{}{}
		}
	}

	for _, pod := range pods.Items {
		key := pod.Name + "-svc"

		if _, ok := servicesMap[key]; !ok { // the pod does not have a corresponding service
			service, err := r.spawnService(ctx, req.Namespace, &pod)

			if err != nil && !errors.IsAlreadyExists(err) {
				return ctrl.Result{}, err
			}

			if !ingressUpdated {
				ingressUpdated = publishToIngress(&ingress, &pod, service)
			}
		}
	}

	if ingressScaffolded {
		if err := r.Create(ctx, &ingress); err != nil {
			logger.Error(err, "unable to create ingress")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if ingressUpdated {
		if err := r.Update(ctx, &ingress); err != nil {
			logger.Error(err, "unable to update ingress")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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

	if len(servicePorts) == 0 {
		servicePorts = append(servicePorts, corev1.ServicePort{Protocol: corev1.ProtocolSCTP, Port: 8080})
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
				DeleteFunc:  func(e event.DeleteEvent) bool { return true },
				UpdateFunc:  func(e event.UpdateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Named("network").
		Complete(r)
}
