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

type Arch int

const (
	AMD64 Arch = iota
	ARM64
)

func (f Arch) String() string {
	names := [...]string{"amd64", "arm64"}
	if f < AMD64 || f > ARM64 {
		return "Unknown"
	}
	return names[f]
}

type CloudProvider int

const (
	AWS CloudProvider = iota
	GCP
)

func (f CloudProvider) String() string {
	names := [...]string{"aws", "gcp"}
	if f < AWS || f > GCP {
		return "Unknown"
	}
	return names[f]
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildkitSpec defines the desired state of Buildkit
type BuildkitSpec struct {
	// CloudProvider
	CloudProvider CloudProvider `json:"cloud,omitempty"`

	Arch []Arch `json:"arch,omitempty"`

	Image string `json:"image,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	MaxReplica int64 `json:"max_replica,omitempty"`

	PublicCertsSecretName string `json:"public_certs,omitempty"`

	DaemonCertsSecretName string `json:"daemon_certs,omitempty"`

	Rootless bool `json:"rootless,omitempty"`
}

// BuildkitStatus defines the observed state of Buildkit
type BuildkitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status bool `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Buildkit is the Schema for the buildkits API
type Buildkit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildkitSpec   `json:"spec,omitempty"`
	Status BuildkitStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuildkitList contains a list of Buildkit
type BuildkitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Buildkit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Buildkit{}, &BuildkitList{})
}
