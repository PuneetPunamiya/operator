/*
Copyright 2020 The Tekton Authors

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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

const (
	// DB
	DbDependenciesInstalled apis.ConditionType = "DbDependenciesInstalled"
	DbInstallerSetAvailable apis.ConditionType = "DbInstallSetAvailable"
	DbInstallerSetReady     apis.ConditionType = "DbInstallSetReady"
	// DB-migration
	DbMigrationInstallerSetAvailable apis.ConditionType = "DbMigrationInstallSetAvailable"
	DbMigrationInstallerSetReady     apis.ConditionType = "DbMigrationInstallSetReady"
	DbMigrationFailed                apis.ConditionType = "DbMigrationFailed"
	// API
	ApiDependenciesInstalled apis.ConditionType = "ApiDependenciesInstalled"
	ApiInstallerSetAvailable apis.ConditionType = "ApiInstallSetAvailable"
	ApiInstallerSetReady     apis.ConditionType = "ApiInstallSetReady"
)

var (
	// TODO: Add this back after refactoring all components
	// and updating TektonComponentStatus to have updated
	// conditions
	// _ TektonComponentStatus = (*TektonHubStatus)(nil)

	hubCondSet = apis.NewLivingConditionSet(
		DbDependenciesInstalled,
		DbInstallerSetAvailable,
		DbMigrationInstallerSetAvailable,
		ApiDependenciesInstalled,
		ApiInstallerSetAvailable,
		PostReconciler,
	)
)

// GroupVersionKind returns SchemeGroupVersion of a TektonHub
func (th *TektonHub) GroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(KindTektonHub)
}

// required by new type of FilterController
// might have to keep this and remove previous or vice-versa
func (th *TektonHub) GetGroupVersionKind() schema.GroupVersionKind {
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

func (ths *TektonHubStatus) MarkNotReady(msg string) {
	hubCondSet.Manage(ths).MarkFalse(
		apis.ConditionReady,
		"Error",
		"Ready: %s", msg)
}

func (ths *TektonHubStatus) MarkPostReconcilerComplete() {
	hubCondSet.Manage(ths).MarkTrue(PostReconciler)
}

func (ths *TektonHubStatus) MarkPostReconcilerFailed(msg string) {
	ths.MarkNotReady("PostReconciliation failed")
	hubCondSet.Manage(ths).MarkFalse(
		PostReconciler,
		"Error",
		"PostReconciliation failed with message: %s", msg)
}

func (ths *TektonHubStatus) MarkApiDependencyInstalling(msg string) {
	ths.MarkNotReady("Dependencies installing for API")
	hubCondSet.Manage(ths).MarkFalse(
		ApiDependenciesInstalled,
		"Error",
		"Dependencies are installing for API: %s", msg)
}

func (ths *TektonHubStatus) MarkApiDependencyMissing(msg string) {
	ths.MarkNotReady("Missing Dependencies for API")
	hubCondSet.Manage(ths).MarkFalse(
		ApiDependenciesInstalled,
		"Error",
		"Dependencies are missing for API: %s", msg)
}

func (ths *TektonHubStatus) MarkApiDependenciesInstalled() {
	hubCondSet.Manage(ths).MarkTrue(ApiDependenciesInstalled)
}

func (ths *TektonHubStatus) MarkDbDependencyInstalling(msg string) {
	ths.MarkNotReady("Dependencies installing for DB")
	hubCondSet.Manage(ths).MarkFalse(
		DbDependenciesInstalled,
		"Error",
		"Dependencies are installing for DB: %s", msg)
}

// GetManifests gets the url links of the manifests.
func (ths *TektonHubStatus) GetManifests() []string {
	return ths.Manifests
}

// SetManifests sets the url links of the manifests.
func (ths *TektonHubStatus) SetManifests(manifests []string) {
	ths.Manifests = manifests
}

func (ths *TektonHubStatus) MarkDbDependencyMissing(msg string) {
	ths.MarkNotReady("Missing Dependencies for DB")
	hubCondSet.Manage(ths).MarkFalse(
		DbDependenciesInstalled,
		"Error",
		"Dependencies are missing for DB: %s", msg)
}

func (ths *TektonHubStatus) MarkDbDependenciesInstalled() {
	hubCondSet.Manage(ths).MarkTrue(DbDependenciesInstalled)
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
func (ths *TektonHubStatus) GetApiRoute() string {
	return ths.ApiRouteUrl
}

// SetManifests sets the url links of the manifests.
func (ths *TektonHubStatus) SetApiRoute(routeUrl string) {
	ths.ApiRouteUrl = routeUrl
}

// DB
func (ths *TektonHubStatus) MarkDbInstallerSetNotReady(msg string) {
	ths.MarkNotReady("TektonInstallerSet not ready for DB")
	hubCondSet.Manage(ths).MarkFalse(
		DbInstallerSetReady,
		"Error",
		"Installer set not ready: %s", msg)
}

func (ths *TektonHubStatus) MarkDbInstallerSetReady() {
	hubCondSet.Manage(ths).MarkTrue(DbInstallerSetReady)
}

func (ths *TektonHubStatus) MarkDbInstallerSetNotAvailable(msg string) {
	ths.MarkNotReady("TektonInstallerSet not ready for DB")
	hubCondSet.Manage(ths).MarkFalse(
		DbInstallerSetAvailable,
		"Error",
		"Installer set not ready: %s", msg)
}

func (ths *TektonHubStatus) MarkDbInstallerSetAvailable() {
	hubCondSet.Manage(ths).MarkTrue(DbInstallerSetAvailable)
}

// DB-Migration
func (ths *TektonHubStatus) MarkDbMigrationInstallerSetNotReady(msg string) {
	ths.MarkNotReady("TektonInstallerSet not ready for DB")
	hubCondSet.Manage(ths).MarkFalse(
		DbMigrationInstallerSetReady,
		"Error",
		"Installer set not ready: %s", msg)
}

func (ths *TektonHubStatus) MarkDbMigrationInstallerSetReady() {
	hubCondSet.Manage(ths).MarkTrue(DbMigrationInstallerSetReady)
}

func (ths *TektonHubStatus) MarkDbMigrationInstallerSetNotAvailable(msg string) {
	ths.MarkNotReady("TektonInstallerSet not ready for DB")
	hubCondSet.Manage(ths).MarkFalse(
		DbMigrationInstallerSetAvailable,
		"Error",
		"Installer set not ready: %s", msg)
}

func (ths *TektonHubStatus) MarkDbMigrationInstallerSetAvailable() {
	hubCondSet.Manage(ths).MarkTrue(DbMigrationInstallerSetAvailable)
}

func (ths *TektonHubStatus) MarkDbMigrationFailed() {
	ths.MarkNotReady("Tekton Hub DB migration failed")
	hubCondSet.Manage(ths).MarkFalse(
		DbMigrationFailed,
		"Error",
		"DB migration failed: %s")
}

// for API
func (ths *TektonHubStatus) MarkApiInstallerSetNotReady(msg string) {
	ths.MarkNotReady("TektonInstallerSet not ready for API")
	hubCondSet.Manage(ths).MarkFalse(
		ApiInstallerSetReady,
		"Error",
		"Installer set not ready for API: %s", msg)
}

func (ths *TektonHubStatus) MarkApiInstallerSetReady() {
	hubCondSet.Manage(ths).MarkTrue(ApiInstallerSetReady)
}

func (ths *TektonHubStatus) MarkApiInstallerSetNotAvailable(msg string) {
	ths.MarkNotReady("TektonInstallerSet not ready for API")
	hubCondSet.Manage(ths).MarkFalse(
		ApiInstallerSetAvailable,
		"Error",
		"Installer set not ready for API: %s", msg)
}

func (ths *TektonHubStatus) MarkApiInstallerSetAvailable() {
	hubCondSet.Manage(ths).MarkTrue(ApiInstallerSetAvailable)
}

// TODO: below methods are not required for TektonAddon
// but as extension implements TektonComponent we need to define them
// this will be removed

func (tas *TektonHubStatus) MarkInstallSucceeded() {
	panic("MarkInstallSucceeded implement me")
}

func (ths *TektonHubStatus) MarkInstallFailed(msg string) {
	panic("MarkInstallFailed implement me")
}

func (ths *TektonHubStatus) MarkDeploymentsAvailable() {
	panic("MarkDeploymentsAvailable implement me")
}

func (ths *TektonHubStatus) MarkDeploymentsNotReady() {
	panic("MarkDeploymentsNotReady implement me")
}

func (ths *TektonHubStatus) MarkDependenciesInstalled() {
	panic("MarkDependenciesInstalled implement me")
}

func (ths *TektonHubStatus) MarkDependencyInstalling(msg string) {
	panic("MarkDependencyInstalling implement me")
}

func (ths *TektonHubStatus) MarkDependencyMissing(msg string) {
	panic("MarkDependencyMissing implement me")
}
