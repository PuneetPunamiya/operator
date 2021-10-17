/*
Copyright 2021 The Tekton Authors
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var (
	_ TektonComponent     = (*TektonHub)(nil)
	_ TektonComponentSpec = (*TektonHubSpec)(nil)
)

// TektonHub is the Schema for the tektonhub API
// +genclient
// +genreconciler:krshapedlogic=false
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced
type TektonHub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TektonHubSpec   `json:"spec,omitempty"`
	Status TektonHubStatus `json:"status,omitempty"`
}

// GetSpec implements TektonComponent
func (tp *TektonHub) GetSpec() TektonComponentSpec {
	return &tp.Spec
}

// GetStatus implements TektonComponent
func (tp *TektonHub) GetStatus() TektonComponentStatus {
	return &tp.Status
}

type DbSpec struct {
	DbSecretName string `json:"secret,omitempty"`
}

type ApiSpec struct {
	HubConfigUrl  string `json:"hubConfigUrl,omitempty"`
	ApiSecretName string `json:"secret,omitempty"`
}

// TektonResultSpec defines the desired state of TektonResult
type TektonHubSpec struct {
	CommonSpec `json:",inline"`
	Db         DbSpec  `json:"db,omitempty"`
	Api        ApiSpec `json:"api,omitempty"`
}

// TektonResultStatus defines the observed state of TektonResult
type TektonHubStatus struct {
	duckv1.Status `json:",inline"`

	// The version of the installed release
	// +optional
	Version string `json:"version,omitempty"`

	// The url links of the manifests, separated by comma
	// +optional
	Manifests []string `json:"manifests,omitempty"`

	ApiRouteUrl string `json:"apiUrl,omitempty"`

	// The current installer set name
	// +optional
	TektonInstallerSet map[string]string `json:"tektonInstallerSets,omitempty"`
}

// TektonResultsList contains a list of TektonResult
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TektonHubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TektonHub `json:"items"`
}
