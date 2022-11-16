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

package tektondashboard

import (
	"context"
	"fmt"

	"github.com/tektoncd/operator/pkg/reconciler/kubernetes/tektoninstallerset/client"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	clientset "github.com/tektoncd/operator/pkg/client/clientset/versioned"
	pipelineinformer "github.com/tektoncd/operator/pkg/client/informers/externalversions/operator/v1alpha1"
	tektondashboardreconciler "github.com/tektoncd/operator/pkg/client/injection/reconciler/operator/v1alpha1/tektondashboard"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"github.com/tektoncd/operator/pkg/reconciler/shared/hash"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler for TektonDashboard resources.
type Reconciler struct {
	// installer Set client to do CRUD operations for components
	installerSetClient *client.InstallerSetClient
	// operatorClientSet allows us to configure operator objects
	operatorClientSet clientset.Interface
	// readOnlyManifest has the source manifest of Tekton Dashboard for
	// a particular version with readonly value as true
	readonlyManifest mf.Manifest
	// fullaccessManifest has the source manifest of Tekton Dashboard for
	// a particular version with readonly value as false
	fullaccessManifest mf.Manifest
	// Platform-specific behavior to affect the transform
	extension        common.Extension
	pipelineInformer pipelineinformer.TektonPipelineInformer
	operatorVersion  string
	dashboardVersion string
}

// Check that our Reconciler implements controller.Reconciler
var _ tektondashboardreconciler.Interface = (*Reconciler)(nil)
var _ tektondashboardreconciler.Finalizer = (*Reconciler)(nil)

var (
	ls = metav1.LabelSelector{
		MatchLabels: map[string]string{
			v1alpha1.CreatedByKey:     createdByValue,
			v1alpha1.InstallerSetType: v1alpha1.DashboardResourceName,
		},
	}
)

const createdByValue = "TektonDashboard"

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, td *v1alpha1.TektonDashboard) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	td.Status.InitializeConditions()
	td.Status.ObservedGeneration = td.Generation
	td.Status.SetVersion(r.dashboardVersion)

	logger.Infow("Reconciling TektonDashboards", "status", td.Status)

	if td.GetName() != v1alpha1.DashboardResourceName {
		msg := fmt.Sprintf("Resource ignored, Expected Name: %s, Got Name: %s",
			v1alpha1.DashboardResourceName,
			td.GetName(),
		)
		logger.Error(msg)
		td.Status.MarkNotReady(msg)
		return nil
	}

	// find the valid tekton-pipeline installation
	if _, err := common.PipelineReady(r.pipelineInformer); err != nil {
		if err.Error() == common.PipelineNotReady || err == v1alpha1.DEPENDENCY_UPGRADE_PENDING_ERR {
			td.Status.MarkDependencyInstalling("tekton-pipelines is still installing")
			// wait for pipeline status to change
			return v1alpha1.REQUEUE_EVENT_AFTER

		}
		// (tektonpipeline.opeator.tekton.dev instance not available yet)
		td.Status.MarkDependencyMissing("tekton-pipelines does not exist")
		return err
	}
	td.Status.MarkDependenciesInstalled()

	if err := r.extension.PreReconcile(ctx, td); err != nil {
		td.Status.MarkPreReconcilerFailed(fmt.Sprintf("PreReconciliation failed: %s", err.Error()))
		return err
	}

	// Mark PreReconcile Complete
	td.Status.MarkPreReconcilerComplete()

	var manifest mf.Manifest
	if td.Spec.Readonly {
		manifest = r.readonlyManifest
	} else {
		manifest = r.fullaccessManifest
	}
	if err := r.installerSetClient.MainSet(ctx, td, &manifest, filterAndTransform(r.extension)); err != nil {
		msg := fmt.Sprintf("Main Reconcilation failed: %s", err.Error())
		logger.Error(msg)
		if err == v1alpha1.REQUEUE_EVENT_AFTER {
			return err
		}
		td.Status.MarkInstallerSetNotReady(msg)
		return nil
	}

	if err := r.extension.PostReconcile(ctx, td); err != nil {
		msg := fmt.Sprintf("PostReconciliation failed: %s", err.Error())
		logger.Error(msg)
		if err == v1alpha1.REQUEUE_EVENT_AFTER {
			return err
		}
		td.Status.MarkPostReconcilerFailed(msg)
		return nil
	}

	td.Status.MarkPostReconcilerComplete()
	return nil
}

func (r *Reconciler) updateTektonDashboardStatus(ctx context.Context, td *v1alpha1.TektonDashboard, createdIs *v1alpha1.TektonInstallerSet) error {
	// update the td with TektonInstallerSet and releaseVersion
	td.Status.SetTektonInstallerSet(createdIs.Name)
	td.Status.SetVersion(r.dashboardVersion)
	return nil
}

// transform mutates the passed manifest to one with common, component
// and platform transformations applied
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.TektonComponent) error {
	instance := comp.(*v1alpha1.TektonDashboard)
	extra := []mf.Transformer{
		common.InjectOperandNameLabelOverwriteExisting(v1alpha1.OperandTektoncdDashboard),
		common.ApplyProxySettings,
		common.AddConfiguration(instance.Spec.Config),
		common.AddDeploymentRestrictedPSA(),
	}
	extra = append(extra, r.extension.Transformers(instance)...)
	return common.Transform(ctx, manifest, instance, extra...)
}

func (r *Reconciler) createInstallerSet(ctx context.Context, td *v1alpha1.TektonDashboard) (*v1alpha1.TektonInstallerSet, error) {

	var manifest mf.Manifest
	if td.Spec.Readonly {
		manifest = r.readonlyManifest
	} else {
		manifest = r.fullaccessManifest
	}

	if err := r.transform(ctx, &manifest, td); err != nil {
		td.Status.MarkNotReady("transformation failed: " + err.Error())
		return nil, err
	}

	// compute the hash of tektondashboard spec and store as an annotation
	// in further reconciliation we compute hash of td spec and check with
	// annotation, if they are same then we skip updating the object
	// otherwise we update the manifest
	specHash, err := hash.Compute(td.Spec)
	if err != nil {
		return nil, err
	}

	// create installer set
	tis := r.makeInstallerSet(td, manifest, specHash)
	createdIs, err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
		Create(ctx, tis, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return createdIs, nil
}

func (r *Reconciler) makeInstallerSet(td *v1alpha1.TektonDashboard, manifest mf.Manifest, tdSpecHash string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(td, td.GetGroupVersionKind())
	return &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", v1alpha1.DashboardResourceName),
			Labels: map[string]string{
				v1alpha1.CreatedByKey:      createdByValue,
				v1alpha1.InstallerSetType:  v1alpha1.DashboardResourceName,
				v1alpha1.ReleaseVersionKey: r.operatorVersion,
			},
			Annotations: map[string]string{
				v1alpha1.TargetNamespaceKey: td.Spec.TargetNamespace,
				v1alpha1.LastAppliedHashKey: tdSpecHash,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: v1alpha1.TektonInstallerSetSpec{
			Manifests: manifest.Resources(),
		},
	}
}

func (r *Reconciler) markUpgrade(ctx context.Context, td *v1alpha1.TektonDashboard) error {
	labels := td.GetLabels()
	ver, ok := labels[v1alpha1.ReleaseVersionKey]
	if ok && ver == r.operatorVersion {
		return nil
	}
	if ok && ver != r.operatorVersion {
		td.Status.MarkInstallerSetNotReady(v1alpha1.UpgradePending)
		td.Status.MarkPreReconcilerFailed(v1alpha1.UpgradePending)
		td.Status.MarkPostReconcilerFailed(v1alpha1.UpgradePending)
		td.Status.MarkNotReady(v1alpha1.UpgradePending)
	}
	if labels == nil {
		labels = map[string]string{}
	}
	labels[v1alpha1.ReleaseVersionKey] = r.operatorVersion
	td.SetLabels(labels)

	if _, err := r.operatorClientSet.OperatorV1alpha1().TektonDashboards().Update(ctx,
		td, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return v1alpha1.RECONCILE_AGAIN_ERR
}
