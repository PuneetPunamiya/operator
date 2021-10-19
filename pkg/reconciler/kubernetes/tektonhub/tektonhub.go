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

package tektonhub

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	clientset "github.com/tektoncd/operator/pkg/client/clientset/versioned"
	tektonhubconciler "github.com/tektoncd/operator/pkg/client/injection/reconciler/operator/v1alpha1/tektonhub"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler for TektonHub resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// operatorClientSet allows us to configure operator objects
	operatorClientSet clientset.Interface
	// manifest is empty, but with a valid client and logger. all
	// manifests are immutable, and any created during reconcile are
	// expected to be appended to this one, obviating the passing of
	// client & logger
	manifest mf.Manifest
	// Platform-specific behavior to affect the transform
	extension common.Extension
}

var (
	apiConfigMapName string = "api"
	keyMissing       error  = fmt.Errorf("secret doesn't contains all the keys")
	// Check that our Reconciler implements controller.Reconciler
	_ tektonhubconciler.Interface = (*Reconciler)(nil)
	_ tektonhubconciler.Finalizer = (*Reconciler)(nil)
)

const (
	dbInstallerSet          = "DbInstallerSet"
	dbMigrationInstallerSet = "DbMigrationInstallerSet"
	apiInstallerSet         = "ApiInstallerSet"
	uiInstallerSet          = "UiInstallerSet"

	releaseVersionKey  = "operator.tekton.dev/release-version"
	createdByKey       = "operator.tekton.dev/created-by"
	targetNamespaceKey = "operator.tekton.dev/target-namespace"
	createdByValue     = "TektonHub"
)

// FinalizeKind removes all resources after deletion of a TektonHub.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *v1alpha1.TektonHub) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	installerSets := original.Status.HubInstallerSet
	if len(installerSets) == 0 {
		return nil
	}

	for _, value := range installerSets {
		err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().Delete(ctx, value, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	// List all TektonHub to determine if cluster-scoped resources should be deleted.
	tps, err := r.operatorClientSet.OperatorV1alpha1().TektonHubs().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all TektonHubs: %w", err)
	}

	for _, tp := range tps.Items {
		if tp.GetDeletionTimestamp().IsZero() {
			// Not deleting all TektonHubs. Nothing to do here.
			return nil
		}
	}

	if err := r.extension.Finalize(ctx, original); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}
	logger.Info("Deleting cluster-scoped resources")
	manifest, err := r.installed(ctx, original)
	if err != nil {
		logger.Error("Unable to fetch installed manifest; no cluster-scoped resources will be finalized", err)
		return nil
	}
	if err := common.Uninstall(ctx, manifest, nil); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}
	return nil
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, th *v1alpha1.TektonHub) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	th.Status.InitializeConditions()
	th.Status.ObservedGeneration = th.Generation
	koDataDir := os.Getenv(common.KoEnvKey)

	version := common.TargetVersion(th)

	hubDir := filepath.Join(koDataDir, "hub", version)

	if th.GetName() != common.HubResourceName {
		msg := fmt.Sprintf("Resource ignored, Expected Name: %s, Got Name: %s",
			common.HubResourceName,
			th.GetName(),
		)
		logger.Error(msg)
		th.Status.MarkNotReady(msg)
		return nil
	}

	// DB install flow
	// check if the db secrets are created
	if err := r.validateDBSecretsAreCreated(ctx, th); err != nil {
		th.Status.MarkDbDependencyMissing("db secrets are either invalid or not present")
		return err
	}
	th.Status.MarkDbDependenciesInstalled()

	exist, err := checkIfInstallerSetExist(ctx, r.operatorClientSet, version, th, dbInstallerSet)
	if err != nil {
		return err
	}

	if !exist {
		th.Status.MarkDbInstallerSetNotAvailable("DB installer set not available")
		dbLocation := filepath.Join(hubDir, "db")
		err := r.applyManifest(ctx, dbLocation, th, dbInstallerSet, version, "hub-db")
		if err != nil {
			return err
		}
	}

	err = r.checkComponentStatus(ctx, th, dbInstallerSet)
	if err != nil {
		// th.Status.MarkNotReady(err.Error())
		return err
	}

	th.Status.MarkDbInstallerSetAvailable()

	// db-migration
	exist, err = checkIfInstallerSetExist(ctx, r.operatorClientSet, version, th, dbMigrationInstallerSet)
	if err != nil {
		return err
	}

	if !exist {
		dbMigrationLocation := filepath.Join(hubDir, "db-migration")
		err := r.applyManifest(ctx, dbMigrationLocation, th, dbMigrationInstallerSet, version, "hub-db-migration")
		if err != nil {
			return err
		}
	}

	// create API

	if err := r.validateApiSecrets(ctx, th); err != nil {
		th.Status.MarkApiDependencyMissing("api secrets not present")
		return err
	}

	th.Status.MarkApiDependenciesInstalled()

	exist, err = checkIfInstallerSetExist(ctx, r.operatorClientSet, version, th, apiInstallerSet)
	if err != nil {
		return err
	}

	if !exist {
		th.Status.MarkApiInstallerSetNotAvailable("API installer set not available")
		apiLocation := filepath.Join(hubDir, "api")
		err := r.applyManifest(ctx, apiLocation, th, apiInstallerSet, version, "hub-api")
		if err != nil {
			return err
		}
	}

	th.Status.MarkApiInstallerSetAvailable()

	if err := r.extension.PostReconcile(ctx, th); err != nil {
		return err
	}

	th.Status.MarkPostReconcilerComplete()

	return nil
}

func (r *Reconciler) validateApiSecrets(ctx context.Context, th *v1alpha1.TektonHub) error {
	logger := logging.FromContext(ctx)

	th.Status.MarkApiDependencyInstalling("checking for api secrets in the namespace and creating the ConfigMap")

	apiSecretKeys := []string{"GH_CLIENT_ID", "GH_CLIENT_SECRET", "JWT_SIGNING_KEY", "ACCESS_JWT_EXPIRES_IN", "REFRESH_JWT_EXPIRES_IN", "GHE_URL"}
	apiConfigMapKeys := []string{"CONFIG_FILE_URL"}

	_, err := r.getSecretForHub(ctx, th.Spec.Api.ApiSecretName, th.Spec.TargetNamespace, apiSecretKeys)
	if err != nil {
		if apierrors.IsNotFound(err) {
			th.Status.MarkApiDependencyMissing(fmt.Sprintf("%s secret is missing", th.Spec.Api.ApiSecretName))
			return err
		}
		if err == keyMissing {
			th.Status.MarkApiDependencyMissing(fmt.Sprintf("%s secret is missing the keys", th.Spec.Api.ApiSecretName))
			return err
		} else {
			logger.Error(err)
			return err
		}
	}

	_, err = r.getConfigMapForHub(ctx, apiConfigMapName, th.Spec.TargetNamespace, apiConfigMapKeys)
	if err != nil {
		if apierrors.IsNotFound(err) {
			configMap := createConfigMap(apiConfigMapName, th)
			_, err = r.kubeClientSet.CoreV1().ConfigMaps(th.Spec.TargetNamespace).Create(ctx, configMap, metav1.CreateOptions{})
			if err != nil {
				logger.Error(err)
				th.Status.MarkApiDependencyMissing(fmt.Sprintf("%s configMap is missing", apiConfigMapName))
				return err
			}
			return nil
		}
		if err == keyMissing {
			th.Status.MarkApiDependencyMissing(fmt.Sprintf("%s configMap is missing the keys", apiConfigMapName))
			return err
		} else {
			logger.Error(err)
			return err
		}
	}

	return nil
}

// TektonHubs expects secrets to be created before installing
func (r *Reconciler) validateDBSecretsAreCreated(ctx context.Context, th *v1alpha1.TektonHub) error {
	logger := logging.FromContext(ctx)

	th.Status.MarkDbDependencyInstalling("db secrets are being added into the namespace")

	dbKeys := []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_PORT"}

	dbSecret, err := r.getSecretForHub(ctx, th.Spec.Db.DbSecretName, th.Spec.TargetNamespace, dbKeys)
	if err != nil {
		fmt.Println("DBSecret------->", dbSecret)
		newDbSecret := createSecret(th.Spec.Db.DbSecretName, th.Spec.TargetNamespace, dbSecret)
		if apierrors.IsNotFound(err) {
			_, err = r.kubeClientSet.CoreV1().Secrets(th.Spec.TargetNamespace).Create(ctx, newDbSecret, metav1.CreateOptions{})
			if err != nil {
				logger.Error(err)
				th.Status.MarkDbDependencyMissing(fmt.Sprintf("%s secret is missing", th.Spec.Db.DbSecretName))
				return err
			}
			return nil
		}
		if err == keyMissing {
			_, err = r.kubeClientSet.CoreV1().Secrets(th.Spec.TargetNamespace).Update(ctx, newDbSecret, metav1.UpdateOptions{})
			if err != nil {
				logger.Error(err)
				th.Status.MarkDbDependencyMissing(fmt.Sprintf("%s secret is missing", th.Spec.Db.DbSecretName))
				return err
			}
		} else {
			logger.Error(err)
			return err
		}
	}

	return nil
}

func (r *Reconciler) getSecretForHub(ctx context.Context, name, targetNs string, keys []string) (*corev1.Secret, error) {
	secret, err := r.kubeClientSet.CoreV1().Secrets(targetNs).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	allKeys := true
	for _, key := range keys {
		if _, ok := secret.Data[key]; !ok {
			allKeys = false
			break
		}
	}

	if !allKeys {
		return nil, keyMissing
	}

	return secret, nil
}

func (r *Reconciler) getConfigMapForHub(ctx context.Context, name, targetNs string, keys []string) (*corev1.ConfigMap, error) {
	configMap, err := r.kubeClientSet.CoreV1().ConfigMaps(targetNs).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	allKeys := true
	for _, key := range keys {
		if _, ok := configMap.Data[key]; !ok {
			allKeys = false
			break
		}
	}

	if !allKeys {
		return nil, keyMissing
	}

	return configMap, nil
}

func createConfigMap(name string, th *v1alpha1.TektonHub) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: th.Spec.TargetNamespace,
			Labels: map[string]string{
				"app": "api",
			},
		},
		Data: map[string]string{
			"CONFIG_FILE_URL": th.Spec.Api.HubConfigUrl,
		},
	}
}

func createSecret(name, namespace string, existingSecret *corev1.Secret) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "db",
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	if existingSecret != nil && existingSecret.Data != nil {
		s.Data = existingSecret.Data
	}

	s.StringData = make(map[string]string, 0)

	if s.Data["POSTGRES_DB"] == nil || len(s.Data["POSTGRES_DB"]) == 0 {
		s.StringData["POSTGRES_DB"] = "hub"
	}

	if s.Data["POSTGRES_USER"] == nil || len(s.Data["POSTGRES_USER"]) == 0 {
		s.StringData["POSTGRES_USER"] = "postgres"
	}

	if s.Data["POSTGRES_PASSWORD"] == nil || len(s.Data["POSTGRES_PASSWORD"]) == 0 {
		s.StringData["POSTGRES_PASSWORD"] = "postgres"
	}

	if s.Data["POSTGRES_PORT"] == nil || len(s.Data["POSTGRES_PORT"]) == 0 {
		s.StringData["POSTGRES_PORT"] = "5432"
	}

	return s
}

func (r *Reconciler) applyManifest(ctx context.Context, manifestLocation string, th *v1alpha1.TektonHub, installerSetName, version, prefixName string) error {
	manifest := r.manifest.Append()
	ownerRef := *metav1.NewControllerRef(th, th.GroupVersionKind())
	logger := logging.FromContext(ctx)

	if err := common.AppendManifest(&manifest, manifestLocation); err != nil {
		return err
	}
	manifest, err := manifest.Transform(
		injectOwner([]metav1.OwnerReference{ownerRef}),
		changeNamespace(th.Spec.TargetNamespace),
	)
	if err != nil {
		logger.Error("failed to transform manifest")
		return err
	}

	if err := createInstallerSet(ctx, r.operatorClientSet, th, manifest,
		version, installerSetName, prefixName); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) installed(ctx context.Context, instance v1alpha1.TektonComponent) (*mf.Manifest, error) {
	// Create new, empty manifest with valid client and logger
	installed := r.manifest.Append()
	stages := common.Stages{common.AppendInstalled}
	err := stages.Execute(ctx, &installed, instance)
	return &installed, err
}

// checkIfInstallerSetExist checks if installer set exists for a component and return true/false based on it
// and if installer set which already exist is of older version then it deletes and return false to create a new
// installer set
func checkIfInstallerSetExist(ctx context.Context, oc clientset.Interface, relVersion string,
	th *v1alpha1.TektonHub, component string) (bool, error) {

	// Check if installer set is already created
	compInstallerSet, ok := th.Status.HubInstallerSet[component]
	if !ok {
		return false, nil
	}

	if compInstallerSet != "" {
		// if already created then check which version it is
		ctIs, err := oc.OperatorV1alpha1().TektonInstallerSets().
			Get(ctx, compInstallerSet, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		version, ok := ctIs.Annotations[releaseVersionKey]
		if ok && version == relVersion {
			// if installer set already exist and release version is same
			// then ignore and move on
			return true, nil
		}

		// release version doesn't exist or is different from expected
		// deleted existing InstallerSet and create a new one

		err = oc.OperatorV1alpha1().TektonInstallerSets().
			Delete(ctx, compInstallerSet, metav1.DeleteOptions{})
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

func createInstallerSet(ctx context.Context, oc clientset.Interface, th *v1alpha1.TektonHub,
	manifest mf.Manifest, releaseVersion, component, installerSetPrefix string) error {

	is := makeInstallerSet(th, manifest, installerSetPrefix, releaseVersion)

	createdIs, err := oc.OperatorV1alpha1().TektonInstallerSets().
		Create(ctx, is, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if len(th.Status.HubInstallerSet) == 0 {
		th.Status.HubInstallerSet = map[string]string{}
	}

	// Update the status of addon with created installerSet name
	th.Status.HubInstallerSet[component] = createdIs.Name
	th.Status.SetVersion(releaseVersion)

	_, err = oc.OperatorV1alpha1().TektonHubs().
		UpdateStatus(ctx, th, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func makeInstallerSet(th *v1alpha1.TektonHub, manifest mf.Manifest, prefix, releaseVersion string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(th, th.GetGroupVersionKind())
	return &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", prefix),
			Labels: map[string]string{
				createdByKey: createdByValue,
			},
			Annotations: map[string]string{
				releaseVersionKey:  releaseVersion,
				targetNamespaceKey: th.Spec.TargetNamespace,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: v1alpha1.TektonInstallerSetSpec{
			Manifests: manifest.Resources(),
		},
	}
}

func (r *Reconciler) deleteInstallerSet(ctx context.Context, th *v1alpha1.TektonHub, component string) error {

	compInstallerSet, ok := th.Status.HubInstallerSet[component]
	if !ok {
		return nil
	}

	if compInstallerSet != "" {
		// delete the installer set
		err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
			Delete(ctx, th.Status.HubInstallerSet[component], metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		// clear the name of installer set from TektonAddon status
		delete(th.Status.HubInstallerSet, component)
		_, err = r.operatorClientSet.OperatorV1alpha1().TektonHubs().
			UpdateStatus(ctx, th, metav1.UpdateOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (r *Reconciler) checkComponentStatus(ctx context.Context, th *v1alpha1.TektonHub, component string) error {

	// Check if installer set is already created
	compInstallerSet, ok := th.Status.HubInstallerSet[component]
	if !ok {
		return nil
	}

	if compInstallerSet != "" {

		ctIs, err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
			Get(ctx, compInstallerSet, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		ready := ctIs.Status.GetCondition(apis.ConditionReady)
		if ready == nil || ready.Status == corev1.ConditionUnknown {
			return fmt.Errorf("InstallerSet %s: waiting for installation", ctIs.Name)
		} else if ready.Status == corev1.ConditionFalse {
			return fmt.Errorf("InstallerSet %s: ", ready.Message)
		}
	}

	return nil
}
