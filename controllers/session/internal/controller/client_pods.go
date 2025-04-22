package controller

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gcv1alpha1 "mr.telepresence/gc/api/v1alpha1"

	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	"mr.telepresence/session/internal/controller/utils"
)

var defaultTime = metav1.Time{Time: time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)}

type allocationValue struct {
	PodTemplate sessionv1alpha1.ClientPodTemplate
	Pods        []pod
}

type pod struct {
	Name    string
	Clients []podClient
}

type podClient struct {
	Id        string
	Connected bool
}

func (r *SessionReconciler) ReconcileClientPods(
	ctx context.Context,
	namespace string,
	session *sessionv1alpha1.Session,
	gcRegistrations []gcv1alpha1.GCRegistration,
) ([]corev1.Pod, error) {

	logger := log.FromContext(ctx)
	now := metav1.NewTime(time.Now())

	// Find pods in the cluster
	var clientPods corev1.PodList
	fieldSelector := client.MatchingFields{utils.PodOwnerField: session.Name, utils.PodTypeField: "client"}

	if err := r.List(ctx, &clientPods, client.InNamespace(namespace), fieldSelector); err != nil {
		logger.Error(err, "unable to get client pods", "session", session.Name)
		return nil, err
	}

	if session.Status.Clients == nil {
		session.Status.Clients = make(map[string]sessionv1alpha1.ClientStatus)
	}

	// Remove clients from the status if their reconnection grace period has expired
	cleanExpiredClientsFromStatus(now, session.Status.Clients, session.Spec.TimeoutSeconds)

	// Check for new clients and lost connections
	newClients := handleClientChanges(now, session.Spec.Clients, session.Status.Clients)

	// Build the allocation map where each key represents a pod template name defined in the spec
	// The corresponding value is an object containing the pod template and a list of pods
	// currently running in the cluster that were created from that template
	allocationMap := initAllocationMap(session.Spec.ClientPodTemplates.Items)
	templatePodToReutilizeMap := templatePodMapping(clientPods.Items)

	for clientId, clientStatus := range session.Status.Clients {
		if _, ok := session.Spec.Clients[clientId]; !ok {
			// client left the session (exists in status but not in spec)
			delete(session.Status.Clients, clientId)
		} else {
			// client may be connected to the session
			isClientConnected := defaultTime.Unix() == clientStatus.LastSeenAt.Unix()
			buildAllocationMap(clientId, &clientStatus, isClientConnected, allocationMap, templatePodToReutilizeMap)
		}
	}

	// allocate the new clients
	if len(newClients) != 0 {
		// sort pods in the allocation map so that new clients are allocated to the least empty ones
		sortAllocationMap(allocationMap)
		allocateClients(allocationMap, newClients, session, templatePodToReutilizeMap)
	}

	// creates and deletes gc registrations for pods to be handled by the gc controller
	templatePodMap := templatePodMapping(clientPods.Items)
	manageGCRegistrations(ctx, r.Client, session, allocationMap, templatePodToReutilizeMap, gcRegistrations)

	// reconcile workload
	podsToSpawn := reconcilePods(allocationMap, session.Status.Clients, templatePodMap)
	return podsToSpawn, nil
}

func cleanExpiredClientsFromStatus(
	now metav1.Time,
	clients map[string]sessionv1alpha1.ClientStatus,
	timeoutSeconds int,
) {
	for k, v := range clients {
		if clientHasExpired(now, v.LastSeenAt, timeoutSeconds) {
			delete(clients, k)
		}
	}
}

func clientHasExpired(now metav1.Time, lastSeen metav1.Time, timeoutSeconds int) bool {
	if lastSeen.Time.Unix() == defaultTime.Time.Unix() {
		return false
	}

	return lastSeen.Time.Add(time.Second*time.Duration(timeoutSeconds)).Before(now.Time) ||
		lastSeen.Time.Add(time.Second*time.Duration(timeoutSeconds)).Equal(now.Time)
}

func handleClientChanges(
	now metav1.Time,
	specClients map[string]bool,
	statusClients map[string]sessionv1alpha1.ClientStatus,
) []string {
	newClients := []string{}

	for specClient, connected := range specClients {
		match, ok := statusClients[specClient]

		if ok && !connected && match.LastSeenAt.Time.Unix() == defaultTime.Time.Unix() {
			// client lost connection
			match.LastSeenAt = now
			statusClients[specClient] = match
		} else if !ok && connected {
			// new client found
			newClients = append(newClients, specClient)
		} else if connected {
			// client may have reconected
			match.LastSeenAt = defaultTime
			statusClients[specClient] = match
		}
	}

	return newClients
}

func initAllocationMap(clientPodTemplates []sessionv1alpha1.ClientPodTemplate) map[string]allocationValue {
	allocationMap := make(map[string]allocationValue, len(clientPodTemplates))

	for _, podTemplate := range clientPodTemplates {
		allocationMap[podTemplate.Name] = allocationValue{PodTemplate: podTemplate, Pods: []pod{}}
	}
	return allocationMap
}

func templatePodMapping(pods []corev1.Pod) map[string]map[string]corev1.Pod {
	templatePodMap := make(map[string]map[string]corev1.Pod)

	for _, pod := range pods {
		templateName := strings.Split(pod.Name, "-")[2]

		if _, ok := templatePodMap[templateName]; !ok {
			templatePodMap[templateName] = map[string]corev1.Pod{pod.Name: pod}
		} else {
			templatePodMap[templateName][pod.Name] = pod
		}
	}
	return templatePodMap
}

func buildAllocationMap(
	clientId string,
	clientStatus *sessionv1alpha1.ClientStatus,
	isClientConnected bool,
	allocationMap map[string]allocationValue,
	templatePodToReutilizeMap map[string]map[string]corev1.Pod,
) {
	client := podClient{Id: clientId, Connected: isClientConnected}

	for podName := range clientStatus.PodStatus {
		podTemplateName := strings.Split(podName, "-")[2]
		pods := allocationMap[podTemplateName].Pods
		delete(templatePodToReutilizeMap[podTemplateName], podName)

		found := false
		for i := 0; i < len(pods); i++ {
			pod := pods[i]

			if !found && pod.Name == podName {
				found = true
				pods[i].Clients = append(pods[i].Clients, client)
			}
		}

		if !found {
			pods = append(pods, pod{Name: podName, Clients: []podClient{client}})
		}

		allocValue := allocationMap[podTemplateName]
		allocValue.Pods = pods
		allocationMap[podTemplateName] = allocValue
	}
}

func sortAllocationMap(allocationMap map[string]allocationValue) {
	for _, allocValue := range allocationMap {
		sort.Slice(allocValue.Pods, func(i, j int) bool {
			return !comparePods(&allocValue.Pods[i], &allocValue.Pods[j])
		})
	}
}

func comparePods(podA *pod, podB *pod) bool {
	if len(podA.Clients) < len(podB.Clients) {
		return true
	}

	podIdA := podA.Name[strings.LastIndex(podA.Name, "-"):]
	podIdB := podB.Name[strings.LastIndex(podB.Name, "-"):]

	if len(podA.Clients) == len(podB.Clients) && podIdA < podIdB {
		return true
	}

	return false
}

func allocateClients(
	allocationMap map[string]allocationValue,
	newClients []string,
	session *sessionv1alpha1.Session,
	templatePodToReutilizeMap map[string]map[string]corev1.Pod,
) {
	for _, clientId := range newClients {
		pods := map[string]sessionv1alpha1.PodStatus{}
		client := podClient{Id: clientId, Connected: true}

		for podTemplateName, allocValue := range allocationMap {
			podIndex := findFirstPodAvailable(allocValue.PodTemplate.MaxClients, allocValue.Pods)
			var podName string

			if podIndex == -1 {
				podName = reutilizePod(templatePodToReutilizeMap[podTemplateName])

				if podName == "" {
					podName = session.Name + "-" + podTemplateName + "-" + uuid.New().String()[:4]
				} else {
					delete(templatePodToReutilizeMap[podTemplateName], podName)
				}

				allocValue.Pods = append(allocValue.Pods, pod{Name: podName, Clients: []podClient{client}})
			} else {
				// allocate the client to an existing instance
				podName = allocValue.Pods[podIndex].Name
				allocValue.Pods[podIndex].Clients = append(allocValue.Pods[podIndex].Clients, client)
			}

			allocationMap[podTemplateName] = allocValue
			pods[podName] = buildPodStatus(podName, allocValue.PodTemplate.Template.Spec)
		}

		session.Status.Clients[clientId] = sessionv1alpha1.ClientStatus{
			LastSeenAt: defaultTime,
			Ready:      false,
			PodStatus:  pods,
		}
	}
}

func findFirstPodAvailable(maxClients int, pods []pod) int {
	result := -1

	for i := 0; i < len(pods); i++ {
		if len(pods[i].Clients) < maxClients {
			return i
		}
	}

	return result
}

func reutilizePod(pods map[string]corev1.Pod) string {
	for k := range pods {
		return k
	}

	return ""
}

func buildPodStatus(podName string, podSpec corev1.PodSpec) sessionv1alpha1.PodStatus {
	paths := []string{}

	for _, container := range podSpec.Containers {
		for _, port := range container.Ports {
			paths = append(paths, "/"+podName+"/"+port.Name)
		}
	}

	return sessionv1alpha1.PodStatus{Ready: false, Paths: paths}
}

func manageGCRegistrations(
	ctx context.Context,
	rClient client.Client,
	session *sessionv1alpha1.Session,
	allocationMap map[string]allocationValue,
	templatePodToReutilizeMap map[string]map[string]corev1.Pod,
	gcRegistrations []gcv1alpha1.GCRegistration,
) error {
	logger := log.FromContext(ctx)

	gcRegistrationsMap := make(map[string]*gcv1alpha1.GCRegistration, len(gcRegistrations))
	for _, gcRegistration := range gcRegistrations {
		gcRegistrationsMap[gcRegistration.Name] = &gcRegistration
	}

	for _, podTemplateName := range templatePodToReutilizeMap {
		for _, pod := range podTemplateName {
			if _, ok := gcRegistrationsMap[pod.Name]; !ok {
				if err := createGCRegistration(ctx, rClient, pod.Name, session.Name, session.Namespace); err != nil {
					return err
				}
			}
		}
	}

	for _, allocValue := range allocationMap {
		for _, pod := range allocValue.Pods {
			isEmpty := podIsEmpty(&pod)

			if gcRegistration, ok := gcRegistrationsMap[pod.Name]; !ok && isEmpty {
				if err := createGCRegistration(ctx, rClient, pod.Name, session.Name, session.Namespace); err != nil {
					return err
				}

			} else if ok && !isEmpty {
				if err := rClient.Delete(ctx, gcRegistration); err != nil && !errors.IsNotFound(err) {
					logger.Error(err, "unable to delete gc registration", "session", session.Name)
					return err
				}
			}
		}
	}

	return nil
}

func createGCRegistration(
	ctx context.Context,
	rClient client.Client,
	podName string,
	sessionName string,
	sessionNamespace string,
) error {
	logger := log.FromContext(ctx)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: gcNamespace,
		},
	}

	if err := rClient.Create(ctx, namespace); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "unable to create gc namespace", "session", sessionName)
		return err
	}

	gcRegistration := &gcv1alpha1.GCRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: gcNamespace,
		},
		Spec: gcv1alpha1.GCRegistrationSpec{
			Session: corev1.ObjectReference{Name: sessionName, Namespace: sessionNamespace},
			Type:    gcv1alpha1.Timeout,
		},
	}

	if err := rClient.Create(ctx, gcRegistration); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "unable to create gc registration", "session", sessionName)
		return err
	}
	return nil
}

func podIsEmpty(pod *pod) bool {
	empty := true

	for _, client := range pod.Clients {
		if client.Connected {
			empty = false
			break
		}
	}

	return empty
}

func reconcilePods(
	allocationMap map[string]allocationValue,
	statusClients map[string]sessionv1alpha1.ClientStatus,
	templatePodMap map[string]map[string]corev1.Pod,
) []corev1.Pod {
	podsToSpawn := []corev1.Pod{}

	for _, allocValue := range allocationMap {
		for _, pod := range allocValue.Pods {

			if value, ok := templatePodMap[allocValue.PodTemplate.Name][pod.Name]; !ok {
				// instance was not found, we have to spawn it
				setClientStatusReadiness(false, pod.Name, pod.Clients, statusClients)

				podsToSpawn = append(podsToSpawn, corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:   pod.Name,
						Labels: map[string]string{"type": "client"},
					},
					Spec: allocValue.PodTemplate.Template.Spec,
				})
			} else {
				// instance was found, we still have to check its status and report it
				readyStatus := utils.ExtractReadyConditionStatusFromPod(&value)

				if readyStatus == corev1.ConditionTrue {
					setClientStatusReadiness(true, pod.Name, pod.Clients, statusClients)
				} else {
					setClientStatusReadiness(false, pod.Name, pod.Clients, statusClients)
				}
			}
		}
	}

	return podsToSpawn
}

func setClientStatusReadiness(
	ready bool,
	podName string,
	podClients []podClient,
	statusClients map[string]sessionv1alpha1.ClientStatus,
) {
	for _, client := range podClients {
		if value, ok := statusClients[client.Id]; ok {
			topReadiness := true

			if podStatus, ok := value.PodStatus[podName]; ok {
				podStatus.Ready = ready
				value.PodStatus[podName] = podStatus

				if !ready {
					topReadiness = false

				} else {
					for _, v := range value.PodStatus {
						if !v.Ready {
							topReadiness = false
							break
						}
					}
				}
			}
			value.Ready = topReadiness
			statusClients[client.Id] = value
		}
	}
}
