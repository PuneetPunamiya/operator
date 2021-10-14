/*
Copyright 2021 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    httr://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

var (
	_          TektonComponentStatus = (*TektonHubStatus)(nil)
	hubCondSet                       = apis.NewLivingConditionSet(
		DependenciesInstalled,
		DeploymentsAvailable,
		InstallSucceeded,
	)
)

// GroupVersionKind returns SchemeGroupVersion of a TektonHub
func (th *TektonHub) GroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(KindTektonHub)
}

// GetCondition returns the current condition of a given condition type
func (ths *TektonHubStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return hubCondSet.Manage(ths).GetCondition(t)
}

// InitializeConditions initializes conditions of an TektonHubStatus
func (ths *TektonHubStatus) InitializeConditions() {
	hubCondSet.Manage(ths).InitializeConditions()
}

// IsReady looks at the conditions returns true if they are all true.
func (ths *TektonHubStatus) IsReady() bool {
	return hubCondSet.Manage(ths).IsHappy()
}

// MarkInstallSucceeded marks the InstallationSucceeded status as true.
func (ths *TektonHubStatus) MarkInstallSucceeded() {
	hubCondSet.Manage(ths).MarkTrue(InstallSucceeded)
	if ths.GetCondition(DependenciesInstalled).IsUnknown() {
		// Assume deps are installed if we're not sure
		ths.MarkDependenciesInstalled()
	}
}

// MarkInstallFailed marks the InstallationSucceeded status as false with the given
// message.
func (ths *TektonHubStatus) MarkInstallFailed(msg string) {
	hubCondSet.Manage(ths).MarkFalse(
		InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

// MarkDeploymentsAvailable marks the DeploymentsAvailable status as true.
func (ths *TektonHubStatus) MarkDeploymentsAvailable() {
	hubCondSet.Manage(ths).MarkTrue(DeploymentsAvailable)
}

// MarkDeploymentsNotReady marks the DeploymentsAvailable status as false and calls out
// it's waiting for deployments.
func (ths *TektonHubStatus) MarkDeploymentsNotReady() {
	hubCondSet.Manage(ths).MarkFalse(
		DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments")
}

// MarkDependenciesInstalled marks the DependenciesInstalled status as true.
func (ths *TektonHubStatus) MarkDependenciesInstalled() {
	hubCondSet.Manage(ths).MarkTrue(DependenciesInstalled)
}

// MarkDependencyInstalling marks the DependenciesInstalled status as false with the
// given message.
func (ths *TektonHubStatus) MarkDependencyInstalling(msg string) {
	hubCondSet.Manage(ths).MarkFalse(
		DependenciesInstalled,
		"Installing",
		"Dependency installing: %s", msg)
}

// MarkDependencyMissing marks the DependenciesInstalled status as false with the
// given message.
func (ths *TektonHubStatus) MarkDependencyMissing(msg string) {
	hubCondSet.Manage(ths).MarkFalse(
		DependenciesInstalled,
		"Error",
		"Dependency missing: %s", msg)
}

// GetVersion gets the currently installed version of the component.
func (ths *TektonHubStatus) GetVersion() string {
	return ths.Version
}

// SetVersion sets the currently installed version of the component.
func (ths *TektonHubStatus) SetVersion(version string) {
	ths.Version = version
}

// GetManifests gets the url links of the manifests.
func (ths *TektonHubStatus) GetManifests() []string {
	return ths.Manifests
}

// SetManifests sets the url links of the manifests.
func (ths *TektonHubStatus) SetManifests(manifests []string) {
	ths.Manifests = manifests
}

// GetManifests gets the url links of the manifests.
func (ths *TektonHubStatus) GetApiRoute() string {
	return ths.ApiRouteUrl
}

// SetManifests sets the url links of the manifests.
func (ths *TektonHubStatus) SetApiRoute(routeUrl string) {
	ths.ApiRouteUrl = routeUrl
}
