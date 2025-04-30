/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// SessionSpec defines the desired state of Session.

type ClientPodTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of pod templates
	Items []ClientPodTemplate `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type ClientPodTemplate struct {
	MaxClients         int `json:"maxClients"`
	corev1.PodTemplate `json:",inline"`
}

type SessionSpec struct {
	SessionPodTemplates     corev1.PodTemplateList `json:"sessionPodTemplates"`
	ClientPodTemplates      ClientPodTemplateList  `json:"clientPodTemplates"`
	TimeoutSeconds          int                    `json:"timeoutSeconds"`
	ReutilizeTimeoutSeconds int                    `json:"reutilizeTimeoutSeconds"`
	Clients                 map[string]bool        `json:"clients"`
}

type PodStatus struct {
	Paths []string `json:"paths"`
	Ready bool     `json:"ready"`
}

type ClientStatus struct {
	LastSeenAt metav1.Time          `json:"lastSeenAt"`
	Ready      bool                 `json:"ready"`
	PodStatus  map[string]PodStatus `json:"podStatus"`
}

type SessionPodsStatus struct {
	Conditions []metav1.Condition   `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	PodsStatus map[string]PodStatus `json:"podsStatus,omitempty"`
}

// SessionStatus defines the observed state of Session.
type SessionStatus struct {
	SessionPods SessionPodsStatus       `json:"sessionPods,omitempty"`
	Clients     map[string]ClientStatus `json:"clients,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Session is the Schema for the sessions API.
type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionSpec   `json:"spec,omitempty"`
	Status SessionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SessionList contains a list of Session.
type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Session `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Session{}, &SessionList{})
}
