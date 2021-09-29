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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	clientset "github.com/tektoncd/operator/pkg/client/clientset/versioned"
	tektonhubconciler "github.com/tektoncd/operator/pkg/client/injection/reconciler/operator/v1alpha1/tektonhub"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"k8s.io/client-go/kubernetes"
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

// Check that our Reconciler implements controller.Reconciler
var _ tektonhubconciler.Interface = (*Reconciler)(nil)
var _ tektonhubconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a TektonHub.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *v1alpha1.TektonHub) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

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
	hubDir := filepath.Join(koDataDir, "hub", common.TargetVersion(th))

	if th.GetName() != common.HubResourceName {
		msg := fmt.Sprintf("Resource ignored, Expected Name: %s, Got Name: %s",
			common.HubResourceName,
			th.GetName(),
		)
		logger.Error(msg)
		th.GetStatus().MarkInstallFailed(msg)
		return nil
	}
	manifest := r.manifest.Append()

	namespaceLocation := filepath.Join(hubDir, "namespace")

	if err := common.AppendManifest(&manifest, namespaceLocation); err != nil {
		return err
	}

	ownerRef := *metav1.NewControllerRef(th, th.GroupVersionKind())

	transformedManifest, err := manifest.Transform(
		injectOwner([]metav1.OwnerReference{ownerRef}),
		changeNamespaceName(th.Spec.TargetNamespace),
	)
	if err != nil {
		logger.Error("failed to transform manifest")
		return err
	}

	// apply the namespace
	if err := transformedManifest.Apply(); err != nil {
		return err
	}

	// check if the secrets are created
	if err := r.validateSecretsAreCreated(ctx, th); err != nil {
		return err
	}
	th.Status.MarkDependenciesInstalled()

	manifest = r.manifest.Append()

	dbLocation := filepath.Join(hubDir, "db")

	if err := common.AppendManifest(&manifest, dbLocation); err != nil {
		return err
	}

	transformedManifest, err = manifest.Transform(
		injectOwner([]metav1.OwnerReference{ownerRef}),
		changeNamespace(th.Spec.TargetNamespace),
	)
	if err != nil {
		logger.Error("failed to transform manifest")
		return err
	}

	// create the DB
	if err := common.Install(ctx, &transformedManifest, th); err != nil {
		return err
	}

	if err := common.CheckDeployments(ctx, &transformedManifest, th); err != nil {
		return err
	}

	manifest = r.manifest.Append()

	// create DB migration
	dbMigrationLocation := filepath.Join(hubDir, "db-migration")

	if err := common.AppendManifest(&manifest, dbMigrationLocation); err != nil {
		return err
	}

	transformedManifest, err = manifest.Transform(
		injectOwner([]metav1.OwnerReference{ownerRef}),
		changeNamespace(th.Spec.TargetNamespace),
	)
	if err != nil {
		logger.Error("failed to transform manifest")
		return err
	}

	if err := common.Install(ctx, &transformedManifest, th); err != nil {
		return err
	}

	if err := common.CheckJobs(ctx, &transformedManifest, th); err != nil {
		return err
	}

	return nil
}

// TektonHubs expects secrets to be created before installing
func (r *Reconciler) validateSecretsAreCreated(ctx context.Context, th *v1alpha1.TektonHub) error {
	logger := logging.FromContext(ctx)

	secret, err := r.kubeClientSet.CoreV1().Secrets(th.Spec.TargetNamespace).Get(ctx, th.Spec.SecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			dbSecret := createSecret(th.Spec.SecretName, th.Spec.TargetNamespace)
			_, err = r.kubeClientSet.CoreV1().Secrets(th.Spec.TargetNamespace).Create(ctx, dbSecret, metav1.CreateOptions{})
			if err != nil {
				logger.Error(err)
				th.Status.MarkDependencyMissing(fmt.Sprintf("%s secret is missing", th.Spec.SecretName))
				return err
			}
			return nil
		}
		logger.Error(err)
		return err
	}

	allKeys := true
	dbKeys := []string{"POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_PORT"}
	for _, dbKey := range dbKeys {
		if _, ok := secret.Data[dbKey]; !ok {
			allKeys = false
			th.Status.MarkDependencyMissing(fmt.Sprintf("missing value %s from %s secret", dbKey, th.Spec.SecretName))
			break
		}
	}

	if !allKeys {
		_, err = r.kubeClientSet.CoreV1().Secrets(th.Spec.TargetNamespace).Update(ctx, createSecret(th.Spec.SecretName, th.Spec.TargetNamespace), metav1.UpdateOptions{})
		if err != nil {
			logger.Error(err)
			th.Status.MarkDependencyMissing(fmt.Sprintf("%s secret is missing", th.Spec.SecretName))
			return err
		}
	}

	return nil
}

func createSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"apps": "db",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"POSTGRES_DB":       "hub",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_PORT":     "5432",
		},
	}

}

// transform mutates the passed manifest to one with common, component
// and platform transformations applied
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.TektonComponent) error {
	instance := comp.(*v1alpha1.TektonHub)
	targetNs := comp.GetSpec().GetTargetNamespace()
	extra := []mf.Transformer{
		common.ReplaceNamespaceInDeploymentArgs(targetNs),
		common.ReplaceNamespaceInDeploymentEnv(targetNs),
	}
	extra = append(extra, r.extension.Transformers(instance)...)
	return common.Transform(ctx, manifest, instance, extra...)
}

func (r *Reconciler) installed(ctx context.Context, instance v1alpha1.TektonComponent) (*mf.Manifest, error) {
	// Create new, empty manifest with valid client and logger
	installed := r.manifest.Append()
	stages := common.Stages{common.AppendInstalled}
	err := stages.Execute(ctx, &installed, instance)
	return &installed, err
}
