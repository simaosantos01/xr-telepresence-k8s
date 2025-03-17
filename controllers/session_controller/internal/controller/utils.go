package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	telepresencev1 "mr.telepresence/controller/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IndexSessionPods(obj ctrlClient.Object) []string {
	pod := obj.(*corev1.Pod)
	owner := metav1.GetControllerOf(pod)

	if owner == nil {
		return nil
	}

	if owner.APIVersion != apiGVStr || owner.Kind != "Session" {
		return nil
	}

	purpose, ok := pod.Labels["purpose"]
	if !ok {
		return nil

	} else if purpose != "session" {
		return nil
	}

	return []string{owner.Name}
}

func IndexBackgroundPods(obj ctrlClient.Object) []string {
	pod := obj.(*corev1.Pod)
	owner := metav1.GetControllerOf(pod)

	if owner == nil {
		return nil
	}

	if owner.APIVersion != apiGVStr || owner.Kind != "Session" {
		return nil
	}

	client, ok := pod.Annotations["client"]
	if !ok {
		return nil
	}

	return []string{client}
}

func PodsAreReady(podList *corev1.PodList) bool {
	for _, pod := range podList.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady &&
				(condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown) {

				return false
			}
		}
	}

	return true
}

func RestorePods(
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
	foundPods *corev1.PodList,
	requiredPods []telepresencev1.PodSpec,
	podObjectMeta *metav1.ObjectMeta,
	client *string) error {

	if len(foundPods.Items) == 0 {
		// all pods are missing
		for _, podSpec := range requiredPods {
			pod := buildPod(podObjectMeta, &podSpec, session, client)
			if err := spawnPod(k8sclient, scheme, ctx, session, pod); err != nil {
				return err
			}
		}
	} else {
		// lets find out which pods are missing
		for _, podSpec := range requiredPods {
			found := false

			for _, foundPod := range foundPods.Items {
				if podSpec.Name == foundPod.Name {
					found = true
					break
				}
			}

			if !found {
				// the pod wasn't found, lets spawn it
				pod := buildPod(podObjectMeta, &podSpec, session, client)
				if err := spawnPod(k8sclient, scheme, ctx, session, pod); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func buildPod(
	objectMeta *metav1.ObjectMeta,
	podSpec *telepresencev1.PodSpec,
	session *telepresencev1.Session,
	client *string) *corev1.Pod {

	objectMeta.Name = session.Name + "-" + podSpec.Name

	if client != nil {
		objectMeta.Name += "-" + *client
	}

	return &corev1.Pod{
		ObjectMeta: *objectMeta,
		Spec:       podSpec.PodSpec,
	}
}

func spawnPod(
	k8sclient ctrlClient.Client,
	scheme *runtime.Scheme,
	ctx context.Context,
	session *telepresencev1.Session,
	pod *corev1.Pod) error {

	logger := log.FromContext(ctx)

	// set controller reference for garbage collection
	if err := ctrl.SetControllerReference(session, pod, scheme); err != nil {
		logger.Error(err, "unable to set controller reference for pod", "session", session.Name, "pod", pod)
		return err
	}

	if err := k8sclient.Create(ctx, pod); err != nil {
		logger.Error(err, "unable to create pod", "session", session.Name, "pod", pod)
		return err
	}

	return nil
}
