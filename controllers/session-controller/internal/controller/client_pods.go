package controller

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	telepresencev1 "mr.telepresence/controller/api/v1"
	"mr.telepresence/controller/internal/controller/utils"
)

var defaultTime = metav1.Time{Time: time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)}

type allocationValue struct {
	Pod       telepresencev1.ClientPod
	Instances []podInstance
}

type podInstance struct {
	Name    string
	Clients []string
}

func (r *SessionReconciler) ReconcileClientPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	logger := log.FromContext(ctx)
	now := metav1.NewTime(time.Now())

	// build map of clients present in status and clear the ones that expired due to lost connection
	clientStatusMap := make(map[string]*telepresencev1.ClientStatus, len(session.Status.Clients))
	for i := 0; i < len(session.Status.Clients); i++ {
		if !clientHasExpired(now, session.Status.Clients[i].LastSeen, session.Spec.TimeoutSeconds) {
			clientStatusMap[session.Status.Clients[i].Client] = &session.Status.Clients[i]
		} else {
			// since the client expired we can remove it from the status
			session.Status.Clients = append(session.Status.Clients[:i], session.Status.Clients[i+1:]...)
			i--
		}
	}

	newClients := []string{}
	clientSpecSet := make(map[string]struct{})

	// compare spec clients with status clients and check for lost connections, new clients and clients that reconnected
	for _, client := range session.Spec.Clients {
		value, ok := clientStatusMap[client.Id]

		clientSpecSet[client.Id] = struct{}{}

		if ok && !client.Connected && value.LastSeen.Time.Unix() == defaultTime.Time.Unix() {
			// client lost connection
			value.LastSeen = now
		} else if !ok && client.Connected {
			// new client found
			newClients = append(newClients, client.Id)
		} else if client.Connected {
			// client may have reconected
			value.LastSeen = defaultTime
		}
	}

	// Find pods in the cluster
	var clientPods corev1.PodList
	fieldSelector := client.MatchingFields{podOwnerField: session.Name, podTypeField: "client"}

	if err := r.List(ctx, &clientPods, client.InNamespace(namespace), fieldSelector); err != nil {
		logger.Error(err, "unable to get client pods", "session", session.Name)
		return err
	}

	/**
	 * Here we build the allocation Map. Each map key corresponds to a clientService declared in the spec, and the value
	 * holds the service (pod) spec and a list with the respective pod instances serving a group of clients.
	 */
	allocationMap := make(map[string]allocationValue, len(session.Spec.ClientServices))
	podInstancesMap := make(map[string]map[string]corev1.Pod, len(clientPods.Items))
	scaffoldAllocationMap(allocationMap, session.Spec.ClientServices)
	scaffoldPodInstancesMap(clientPods.Items, podInstancesMap)
	podInstancesToReutilizeMap := copyPodInstancesMap(podInstancesMap)

	for i := 0; i < len(session.Status.Clients); i++ {
		client := session.Status.Clients[i]

		if _, ok := clientSpecSet[client.Client]; !ok {
			// client left the session (exists in status but not in spec)
			delete(clientStatusMap, client.Client)
			session.Status.Clients = append(session.Status.Clients[:i], session.Status.Clients[i+1:]...)
			i--
		} else {
			// client is connected
			buildAllocationMap(allocationMap, client, podInstancesToReutilizeMap)
		}
	}

	// allocate the new clients
	if len(newClients) != 0 {
		tPodInstancesToReutilizeMap := transformPodInstancesToReutilizeMap(podInstancesToReutilizeMap)
		allocateClients(allocationMap, newClients, session, tPodInstancesToReutilizeMap)
	}

	// reconcile workload
	return reconcilePods(ctx, r.Client, r.Scheme, session, allocationMap, clientStatusMap, podInstancesMap)
}

func clientHasExpired(now metav1.Time, lastSeen metav1.Time, timeoutSeconds int) bool {
	if lastSeen.Time.Unix() == defaultTime.Time.Unix() {
		return false
	}

	return lastSeen.Time.Add(time.Second*time.Duration(timeoutSeconds)).Before(now.Time) ||
		lastSeen.Time.Add(time.Second*time.Duration(timeoutSeconds)).Equal(now.Time)
}

func scaffoldAllocationMap(allocationMap map[string]allocationValue, clientPodTypes []telepresencev1.ClientPod) {
	for _, pod := range clientPodTypes {
		allocationMap[pod.Name] = allocationValue{Pod: pod, Instances: []podInstance{}}
	}
}

func scaffoldPodInstancesMap(clientPods []corev1.Pod, podInstancesMap map[string]map[string]corev1.Pod,
) {
	for _, pod := range clientPods {
		podType := strings.Split(pod.Name, "-")[2]

		if _, ok := podInstancesMap[podType]; !ok {
			podInstancesMap[podType] = map[string]corev1.Pod{pod.Name: pod}
		} else {
			podInstancesMap[podType][pod.Name] = pod
		}
	}
}

func copyPodInstancesMap(podInstancesMap map[string]map[string]corev1.Pod) map[string]map[string]corev1.Pod {
	copy := make(map[string]map[string]corev1.Pod, len(podInstancesMap))

	for k, v := range podInstancesMap {
		copy[k] = make(map[string]corev1.Pod, len(v))

		for innerK, innerV := range copy[k] {
			copy[k][innerK] = innerV
		}
	}

	return copy
}

func buildAllocationMap(
	allocationMap map[string]allocationValue,
	client telepresencev1.ClientStatus,
	podInstancesToReutilizeMap map[string]map[string]corev1.Pod,
) {
	for _, endpoint := range client.Endpoints {
		clientPodType := strings.Split(endpoint.Pod, "-")[2]
		podInstances := allocationMap[clientPodType].Instances
		delete(podInstancesToReutilizeMap[clientPodType], endpoint.Pod)

		found := false
		for i := 0; i < len(podInstances); i++ {
			instance := podInstances[i]

			if !found && instance.Name == endpoint.Pod {
				found = true
				podInstances[i].Clients = append(podInstances[i].Clients, client.Client)
			}

			if found && i < len(podInstances)-1 && instanceIsLessThen(instance, podInstances[i+1]) {
				podInstances[i] = podInstances[i+1]
				podInstances[i+1] = instance
			} else if found {
				break
			}
		}

		if !found {
			podInstances = append(podInstances, podInstance{Name: endpoint.Pod, Clients: []string{client.Client}})

			for i := len(podInstances) - 1; i >= 0; i-- {
				instance := podInstances[i]

				if i > 0 && !instanceIsLessThen(instance, podInstances[i-1]) {
					podInstances[i] = podInstances[i-1]
					podInstances[i-1] = instance
				} else {
					break
				}
			}
		}

		value := allocationMap[clientPodType]
		value.Instances = podInstances
		allocationMap[clientPodType] = value
	}
}

func instanceIsLessThen(instanceA podInstance, instanceB podInstance) bool {
	if len(instanceA.Clients) < len(instanceB.Clients) {
		return true
	}

	podIdA := instanceA.Name[strings.LastIndex(instanceA.Name, "-"):]
	podIdB := instanceB.Name[strings.LastIndex(instanceB.Name, "-"):]

	if len(instanceA.Clients) == len(instanceB.Clients) && podIdA < podIdB {
		return true
	}

	return false
}

func transformPodInstancesToReutilizeMap(
	podInstancesToReutilizeMap map[string]map[string]corev1.Pod,
) map[string][]string {

	mapping := make(map[string][]string, len(podInstancesToReutilizeMap))

	for k, v := range podInstancesToReutilizeMap {
		mapping[k] = []string{}

		for instance := range v {
			mapping[k] = append(mapping[k], instance)
		}

		sort.Strings(mapping[k])
	}

	return mapping
}

func allocateClients(
	allocationMap map[string]allocationValue,
	newClients []string,
	session *telepresencev1.Session,
	podInstancesToReutilize map[string][]string,
) {
	for _, client := range newClients {
		endpoints := []telepresencev1.ClientEndpointStatus{}

		for key, value := range allocationMap {
			instanceIndex := findFirstPodInstanceAvailable(value.Pod.MaxClients, value.Instances)
			var podInstanceName string

			if instanceIndex == -1 {
				podInstanceName = reutilizePodInstance(podInstancesToReutilize[key])

				if podInstanceName == "" {
					podInstanceName = session.Name + "-" + key + "-" + uuid.New().String()[:4]
				} else {
					podInstancesToReutilize[key] = podInstancesToReutilize[key][1:]
				}

				value.Instances = append(allocationMap[key].Instances,
					podInstance{Name: podInstanceName, Clients: []string{client}})
			} else {
				// allocate the client to an existing instance
				podInstanceName = value.Instances[instanceIndex].Name
				value.Instances[instanceIndex].Clients = append(value.Instances[instanceIndex].Clients, client)
			}

			allocationMap[key] = value
			endpoints = append(endpoints, buildEndpoint(podInstanceName, value.Pod.Spec))
		}

		session.Status.Clients = append(session.Status.Clients, telepresencev1.ClientStatus{
			Client:    client,
			LastSeen:  defaultTime,
			Ready:     false,
			Endpoints: endpoints,
		})
	}
}

func findFirstPodInstanceAvailable(maxClients int, instances []podInstance) int {
	result := -1

	for i := 0; i < len(instances); i++ {
		if len(instances[i].Clients) < maxClients {
			return i
		}
	}

	return result
}

func reutilizePodInstance(instances []string) string {
	if len(instances) == 0 {
		return ""
	}

	return instances[0]
}

func buildEndpoint(podInstanceName string, podSpec corev1.PodSpec) telepresencev1.ClientEndpointStatus {
	paths := []string{}

	for _, container := range podSpec.Containers {
		for _, port := range container.Ports {
			paths = append(paths, "/"+podInstanceName+"/"+port.Name)
		}
	}

	return telepresencev1.ClientEndpointStatus{
		Pod:   podInstanceName,
		Ready: false,
		Paths: paths,
	}
}

func reconcilePods(
	ctx context.Context,
	rClient client.Client,
	scheme *runtime.Scheme,
	session *telepresencev1.Session,
	allocationMap map[string]allocationValue,
	clientStatusMap map[string]*telepresencev1.ClientStatus,
	podInstancesMap map[string]map[string]corev1.Pod,
) error {
	for _, v := range allocationMap {
		for _, instance := range v.Instances {

			if value, ok := podInstancesMap[v.Pod.Name][instance.Name]; !ok {
				// instance was not found, we have to spawn it
				setClientStatusReadiness(false, instance.Name, instance.Clients, clientStatusMap)
				pod := telepresencev1.Pod{
					Name:   instance.Name,
					Labels: map[string]string{"type": "client"},
					Spec:   v.Pod.Spec,
				}

				// TODO: SPAWN ONLY AFTER THE OBJECT GETS UPDATED
				if err := utils.SpawnPod(ctx, rClient, scheme, session, pod); err != nil {
					return err
				}
			} else {
				// instance was found, we still have to check its status and report it
				readyStatus := utils.ExtractReadyConditionStatusFromPod(&value)

				if readyStatus == corev1.ConditionTrue {
					setClientStatusReadiness(true, instance.Name, instance.Clients, clientStatusMap)
				} else {
					setClientStatusReadiness(false, instance.Name, instance.Clients, clientStatusMap)
				}
			}
		}
	}

	return nil
}

func setClientStatusReadiness(
	ready bool,
	instanceName string,
	instanceClients []string,
	clientStatusMap map[string]*telepresencev1.ClientStatus,
) {
	for _, client := range instanceClients {
		if value, ok := clientStatusMap[client]; ok {
			topReadiness := true

			for i := 0; i < len(value.Endpoints); i++ {
				if value.Endpoints[i].Pod == instanceName && !ready {
					value.Endpoints[i].Ready = false
					topReadiness = false
					break

				} else if value.Endpoints[i].Pod == instanceName {
					value.Endpoints[i].Ready = ready

				} else if !value.Endpoints[i].Ready {
					topReadiness = false
				}
			}

			value.Ready = topReadiness
		}
	}
}
