package controller

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	telepresencev1 "mr.telepresence/controller/api/v1"
)

var defaultTime = metav1.Time{time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)}

type allocationValue struct {
	PodSpec telepresencev1.ClientPod
	Pods    []podInstance
}

type podInstance struct {
	Name    string
	Clients int
}

func (r *SessionReconciler) ReconcileClientPods(
	ctx context.Context,
	namespace string,
	session *telepresencev1.Session) error {

	now := metav1.NewTime(time.Now())

	// build map of clients present in status and clear the ones that expired due to lost connection
	clientStatusMap := make(map[string]*telepresencev1.ClientStatus, len(session.Status.Clients))
	for i := 0; i < len(session.Status.Clients); i++ {
		if !clientHasExpired(now, session.Status.Clients[i].LastSeen, session.Spec.TimeoutSeconds) {
			clientStatusMap[session.Status.Clients[i].Client] = &session.Status.Clients[i]
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

	/**
	* Here we build the allocation Map. Each map key corresponds to a clientService declared in the spec, and the value
	* holds the service (pod) spec and a list with the respective pod instances serving a group of clients.
	 */
	allocationMap := make(map[string]allocationValue, len(session.Spec.ClientServices))
	scaffoldAllocationMap(allocationMap, session.Spec.ClientServices)
	for k, v := range clientStatusMap {
		if _, ok := clientSpecSet[k]; !ok {
			// client left the session (exists in status but not in spec)
			delete(clientStatusMap, k)

		} else {
			// client is connected
			buildAllocationMap(allocationMap, *v)
		}
	}

	// allocate the new clients
	allocateClients(allocationMap, newClients, session)

	// reconcile workload

	return nil
}

func clientHasExpired(now metav1.Time, lastSeen metav1.Time, timeoutSeconds int) bool {
	if lastSeen.Time.Unix() == defaultTime.Time.Unix() {
		return false
	}

	return lastSeen.Add(time.Second*time.Duration(timeoutSeconds)).Before(now.Time) ||
		lastSeen.Add(time.Second*time.Duration(timeoutSeconds)).Equal(now.Time)
}

func scaffoldAllocationMap(allocationMap map[string]allocationValue, clientPods []telepresencev1.ClientPod) {
	for _, pod := range clientPods {
		allocationMap[pod.Name] = allocationValue{PodSpec: pod, Pods: []podInstance{}}
	}
}

func buildAllocationMap(allocationMap map[string]allocationValue, client telepresencev1.ClientStatus) {
	for _, endpoint := range client.Endpoints {
		podName := strings.Split(endpoint.Pod, "-")[2]
		podInstances := allocationMap[podName].Pods

		found := false
		for i := 0; i < len(podInstances); i++ {
			instance := podInstances[i]

			if !found && instance.Name == endpoint.Pod {
				found = true
				podInstances[i].Clients += 1
			}

			if found && i < len(podInstances)-1 && instanceIsLessThen(instance, podInstances[i+1]) {
				podInstances[i] = podInstances[i+1]
				podInstances[i+1] = instance
			} else if found {
				break
			}
		}

		if !found {
			podInstances = append(podInstances, podInstance{Name: endpoint.Pod, Clients: 1})

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

		value := allocationMap[podName]
		value.Pods = podInstances
		allocationMap[podName] = value
	}
}

func instanceIsLessThen(instanceA podInstance, instanceB podInstance) bool {
	if instanceA.Clients < instanceB.Clients {
		return true
	}

	podIdA := instanceA.Name[strings.LastIndex(instanceA.Name, "-"):]
	podIdB := instanceB.Name[strings.LastIndex(instanceB.Name, "-"):]

	if instanceA.Clients == instanceB.Clients && podIdA < podIdB {
		return true
	}

	return false
}

func allocateClients(
	allocationMap map[string]allocationValue,
	newClients []string,
	session *telepresencev1.Session,
) {
	for _, client := range newClients {
		endpoints := []telepresencev1.ClientEndpointStatus{}

		for key, value := range allocationMap {
			instanceIndex := findFirstPodInstanceAvailable(value.PodSpec.MaxClients, value.Pods)
			var podInstanceName string

			if instanceIndex == -1 {
				// a new instance must be created TODO: OR REUTILIZE AN EXISTING ONE
				podInstanceName = session.Name + "-" + key + "-" + uuid.New().String()[:4]
				value.Pods = append(allocationMap[key].Pods, podInstance{Name: podInstanceName, Clients: 1})
			} else {
				// allocate the client to an existing instance
				podInstanceName = value.Pods[instanceIndex].Name
				value.Pods[instanceIndex].Clients += 1
			}

			allocationMap[key] = value
			endpoints = append(endpoints, buildEndpoint(podInstanceName, value.PodSpec.Spec))
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
		if instances[i].Clients < maxClients {
			return i
		}
	}

	return result
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
