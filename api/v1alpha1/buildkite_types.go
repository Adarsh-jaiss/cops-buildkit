/*
Copyright 2024.

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

// BuildkiteSpec defines the desired state of Buildkite
type BuildkiteSpec struct {
	Image string `json:"image,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// BuildkiteStatus defines the observed state of Buildkite
type BuildkiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Buildkite is the Schema for the buildkites API
type Buildkite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildkiteSpec   `json:"spec,omitempty"`
	Status BuildkiteStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuildkiteList contains a list of Buildkite
type BuildkiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Buildkite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Buildkite{}, &BuildkiteList{})
}
