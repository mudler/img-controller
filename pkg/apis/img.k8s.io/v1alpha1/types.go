/*
Copyright 2017 The Kubernetes Authors.

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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PackageBuild is a specification for a PackageBuild resource
type ImageBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec"`
	Status BuildStatus `json:"status"`
}

type RegistryCredentials struct {
	Enabled    bool   `json:"enabled"`
	Registry   string `json:"registry"` // e.g. quay.io
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromSecret string `json:"fromSecret"`
}

type Repository struct {
	Url      string `json:"url"`
	Checkout string `json:"checkout"`
}

// BuildSpec is the spec for a PackageBuild resource
type BuildSpec struct {
	ImageName           string                      `json:"imageName"`
	Context             string                      `json:"context"`
	Dockerfile          string                      `json:"dockerfile`
	Repository          Repository                  `json:"repository"`
	Privileged          bool                        `json:"privileged"`
	NodeSelector        map[string]string           `json:"nodeSelector"`
	Annotations         map[string]string           `json:"annotations"`
	Labels              map[string]string           `json:"labels"`
	RegistryCredentials RegistryCredentials         `json:"registry"`
	Resources           corev1.ResourceRequirements `json:"resources"`
}

type BuildStatus struct {
	State string `json:"state"`
}
