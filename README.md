# A Kubernetes framework for XR telepresence workloads

Kubernetes was originally designed for cloud-centric stateless architectures with load-based scaling. This model conflicts with the demands of XR telepresence systems, which require session-aware scaling for user-shared sessions and 
stable network identities for persistent, stateful workloads. This framework encapsulates the complexities of compute scheduling, service discovery, and networking across sliced edge environments into a simple interface designed to be 
integrated directly into an application’s runtime, enabling it to communicate with the infrastructure and schedule compute resources according to session demands.

Unlike REST services that scale based on generic metrics like CPU or request load, XR telepresence must scale in response to application-level session events, such as users joining or leaving. Standard primitives like the Deployment and 
StatefulSet resources are ill-suited for this task. This limitation is evident during both scale-up and scale-down operations. When scaling up, these controllers simply increment a replica count; they create a generic Pod and have no 
mechanism to associate it with the specific client that triggered the scaling event. This forces the client into a complex and inefficient discovery process to find its allocated resource. When scaling down, the problem persists. A Deployment 
treats its Pods as fungible replicas and terminates them non-deterministically, which could remove a Pod serving an active client. A StatefulSet imposes a strict ordinal scaling policy, which prevents the targeted removal of a specific idle 
Pod from the middle of the set.

## Architecture

TODO

## Features

**Session Lifecycle Management**: The framework provides an interface to manage the lifecycle of multiple, concurrent telepresence sessions. This includes creating sessions and tracking client activity (e.g., joining or leaving a session), 
and translates these events into corresponding workload scheduling actions within Kubernetes.

**Multi-Cluster Support and Location Affinity**: The framework supports workload deployment across distinct, geographically distributed clusters. It also provides a mechanism for clients to measure network latency to these locations and select the
most performant one for hosting their workloads.

**Configurable Session Topologies**: The framework supports different types of sessions, each with a unique, configurable workload topology. This allows for diverse deployment models; for example, one session type could require a dedicated Pod per-
forming object detection for each client, while another type could require a single, shared Pod performing rendering for every five clients who join the session.

**Garbage Collection and Reuse**: The framework provides mechanisms for efficient resource utilization. This includes automatically terminating idle Pods after a configurable grace period. Furthermore, it supports an optional Pod reuse policy by
preserving idle Pods for a second configurable timeout, making them available to new clients joining the same session.

## Usage

### Prerequisites and Installation

Before installing the framework, the following dependencies should be installed and configured into the clusters. You should setup KubEdge with one cluster being the primary (where the framework controllers and session manager will live) and the 
amount of cluster workers you desire.

- [KubeEdge](https://github.com/kubeedge/kubeedge)
- [STUNner](https://github.com/l7mp/stunner) (only required for WebRTC workloads)

Installing the framework: controllers and session manager

TODO

### Deploying workloads

TODO
