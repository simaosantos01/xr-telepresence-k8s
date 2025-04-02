package controller

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
)

func (r *NetworkReconciler) reconcileIngress(
	ctx context.Context,
	servicesMap map[string]*corev1.Service,
) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	var ingress netv1.Ingress
	namespacedName := types.NamespacedName{Name: "ingress", Namespace: "default"}
	ingressScaffolded := false
	if err := r.Client.Get(ctx, namespacedName, &ingress); err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "unable to get ingress resource")
		return ctrl.Result{}, err

	} else if err != nil {
		ingressScaffolded = true
		ingress = *scaffoldIngress()
	}

	setIngressPaths(&ingress)
	ingressPodsSet, ingressUpdatedByGC := ingressGarbageCollection(&ingress, servicesMap)
	ingressUpdatedByPub := publishToIngress(&ingress, servicesMap, ingressPodsSet)

	if len(ingress.Spec.Rules[0].HTTP.Paths) == 0 {
		// this is required because ingress is not valid with empty paths array :(
		ingress.Spec.Rules[0].HTTP = nil
	}

	if ingressScaffolded {
		if err := r.Create(ctx, &ingress); err != nil {
			logger.Error(err, "unable to create ingress")
			return ctrl.Result{}, err
		}

	} else if ingressUpdatedByGC || ingressUpdatedByPub {
		if err := r.Update(ctx, &ingress); err != nil {
			logger.Error(err, "unable to update ingress")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func ingressGarbageCollection(
	ingress *netv1.Ingress,
	servicesMap map[string]*corev1.Service,
) (map[string]struct{}, bool) {

	ingressPodsSet := make(map[string]struct{})
	updated := false
	paths := ingress.Spec.Rules[0].HTTP.Paths

	// TODO: only http implemented
	for i := 0; i < len(paths); i++ {
		pathStr := paths[i].Path
		podNameFromPath := strings.Split(pathStr, "/")[1]
		ingressPodsSet[podNameFromPath] = struct{}{}

		if _, ok := servicesMap[podNameFromPath+"-svc"]; !ok {
			updated = true
			paths = append(paths[:i], paths[i+1:]...)
			i--
		}
	}

	ingress.Spec.Rules[0].HTTP.Paths = paths
	return ingressPodsSet, updated
}

func publishToIngress(
	ingress *netv1.Ingress,
	servicesMap map[string]*corev1.Service,
	ingressPodsSet map[string]struct{},
) bool {

	updated := false
	paths := ingress.Spec.Rules[0].HTTP.Paths
	pathType := netv1.PathTypeImplementationSpecific

	// TODO: only http implemented
	for serviceName, service := range servicesMap {
		podNameFromServiceName := serviceName[:len(serviceName)-4]

		if _, ok := ingressPodsSet[podNameFromServiceName]; !ok {
			for _, port := range service.Spec.Ports {
				updated = true
				path := "/" + podNameFromServiceName + "/" + port.Name + "(/|$)(.*)"
				paths = append(paths, netv1.HTTPIngressPath{
					PathType: &pathType,
					Path:     path,
					Backend: netv1.IngressBackend{
						Service: &netv1.IngressServiceBackend{
							Name: service.Name,
							Port: netv1.ServiceBackendPort{Number: port.Port}},
					},
				})
			}
		}
	}

	ingress.Spec.Rules[0].HTTP.Paths = paths
	return updated
}

func setIngressPaths(ingress *netv1.Ingress) {
	if ingress.Spec.Rules[0].HTTP == nil {
		ingress.Spec.Rules[0].HTTP = &netv1.HTTPIngressRuleValue{Paths: []netv1.HTTPIngressPath{}}
	}
}

func scaffoldIngress() *netv1.Ingress {
	ingress := &netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/use-regex":      "true",
				"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
			},
			Name:      "ingress",
			Namespace: "default",
		},
		Spec: netv1.IngressSpec{
			TLS:   []netv1.IngressTLS{{Hosts: []string{"localhost"}, SecretName: "tls"}},
			Rules: []netv1.IngressRule{{IngressRuleValue: netv1.IngressRuleValue{}}},
		},
	}

	return ingress
}
