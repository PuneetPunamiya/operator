package tektonhub

import (
	"context"
	"fmt"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

func (r *Reconciler) managePVC(ctx context.Context, th *v1alpha1.TektonHub, componentInstallerSetType string) error {
	// Check if PVC installerset already exists
	existingPVCInstallerSet, err := r.CheckIfInstallerSetExists(ctx, componentInstallerSetType)
	if err != nil {
		return err
	}

	// If not exists create the installerset
	if existingPVCInstallerSet == "" {
		th.Status.MarkDbPVCInstallerSetNotAvailable(fmt.Sprintf("%s Installerset not available", componentInstallerSetType))
		manifest, err := r.GetManifest(ctx, th, "db")
		if err != nil {
			return err
		}
		_, err = r.CreatePVCInstallerSet(ctx, th, manifest)
		if err != nil {
			return err
		}
		return v1alpha1.RECONCILE_AGAIN_ERR
	}

	logger := logging.FromContext(ctx)
	// If exists, then fetch the Tekton Hub db pvc InstallerSet
	installedPVCTIS, err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
		Get(ctx, existingPVCInstallerSet, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			manifest, err := r.GetManifest(ctx, th, "db")
			if err != nil {
				return err
			}
			_, err = r.CreatePVCInstallerSet(ctx, th, manifest)
			if err != nil {
				return err
			}
			return v1alpha1.RECONCILE_AGAIN_ERR
		}
		logger.Error("failed to get InstallerSet: %s", err)
		return err
	}

	// Get the namespace for pvc installerset and compare it with the Tekton Hub spec's namespace
	// if it is not same then delete the installerset
	pvcInstallerSetTargetNamespace := installedPVCTIS.Annotations[v1alpha1.TargetNamespaceKey]

	if pvcInstallerSetTargetNamespace != th.Spec.TargetNamespace {
		err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
			Delete(ctx, existingPVCInstallerSet, metav1.DeleteOptions{})
		if err != nil {
			logger.Error("failed to delete TektonHubDbPVC InstallerSet: %s", err)
			return err
		}

		// Make sure the Tekton hub db pvc InstallerSet is deleted
		_, err = r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
			Get(ctx, existingPVCInstallerSet, metav1.GetOptions{})
		if err == nil {
			th.Status.MarkNotReady("Waiting for previous installer set to get deleted")
			return v1alpha1.REQUEUE_EVENT_AFTER
		}
		if !apierrors.IsNotFound(err) {
			logger.Error("failed to get InstallerSet: %s", err)
			return err
		}
		return nil
	}

	err = r.checkComponentStatus(ctx, th, componentInstallerSetType)
	if err != nil {
		th.Status.MarkDbPVCInstallerSetNotAvailable(err.Error())
		return v1alpha1.RECONCILE_AGAIN_ERR
	}

	return nil
}
