package controller

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
)

func deleteFromIngress(ingress *netv1.Ingress, service *corev1.Service) bool {
	updated := false
	podNameFromService := service.Name[:strings.Index(service.Name, "-")]
	paths := &ingress.Spec.Rules[0].HTTP.Paths

	//TODO: ingress requires at least one path :(

	// TODO: only http implemented
	for i := 0; i < len((*paths)); i++ {
		pathStr := (*paths)[i].Path
		podNameFromPath := strings.Split(pathStr, "/")[1]

		if podNameFromService == podNameFromPath {
			updated = true
			*paths = append((*paths)[:i], (*paths)[i+1:]...)
			i--
		}
	}

	return updated
}

func publishToIngress(ingress *netv1.Ingress, forPod *corev1.Pod, service *corev1.Service) bool {
	updated := false
	pathType := netv1.PathTypeImplementationSpecific
	ingressPathsMap := make(map[string]struct{}, len(ingress.Spec.Rules[0].HTTP.Paths))

	for _, path := range ingress.Spec.Rules[0].HTTP.Paths {
		ingressPathsMap[path.Path] = struct{}{}
	}

	// TODO: only http implemented
	for _, port := range service.Spec.Ports {
		path := "/" + forPod.Name + "/" + port.Name + "(/|$)(.*)"

		if _, ok := ingressPathsMap[path]; !ok {
			updated = true

			ingress.Spec.Rules[0].HTTP.Paths = append(ingress.Spec.Rules[0].HTTP.Paths, netv1.HTTPIngressPath{
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
	return updated
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
			TLS: []netv1.IngressTLS{{Hosts: []string{"localhost"}, SecretName: "tls"}},
			Rules: []netv1.IngressRule{{
				IngressRuleValue: netv1.IngressRuleValue{
					HTTP: &netv1.HTTPIngressRuleValue{
						Paths: []netv1.HTTPIngressPath{},
					},
				},
			}},
		},
	}

	return ingress
}
