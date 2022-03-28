package common

import (
	"context"

	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/reconciler/shared/hash"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

// TODO:- Reduce the params
func (sh SharedInstaller) UpdateInstallerSet(ctx context.Context, obj v1alpha1.TektonComponent, installedTIS *v1alpha1.TektonInstallerSet, existingInstallerSet, installerSetTargetNamespace, installerSetReleaseVersion string) error {
	logger := logging.FromContext(ctx)
	// Check if TargetNamespace of existing TektonInstallerSet is same as expected
	// Check if Release Version in TektonInstallerSet is same as expected
	// If any of the thing above is not same then delete the existing TektonInstallerSet
	// and create a new with expected properties

	if installerSetTargetNamespace != obj.GetSpec().GetTargetNamespace() || installerSetReleaseVersion != sh.OperatorVersion {
		// Delete the existing TektonInstallerSet
		err := sh.OperatorClientSet.OperatorV1alpha1().TektonInstallerSets().
			Delete(ctx, existingInstallerSet, metav1.DeleteOptions{})
		if err != nil {
			logger.Error("failed to delete InstallerSet: %s", err)
			return err
		}

		// Make sure the TektonInstallerSet is deleted
		_, err = sh.OperatorClientSet.OperatorV1alpha1().TektonInstallerSets().
			Get(ctx, existingInstallerSet, metav1.GetOptions{})
		if err == nil {
			obj.GetStatus().MarkNotReady("Waiting for previous installer set to get deleted")
			return v1alpha1.REQUEUE_EVENT_AFTER
		}
		if !apierrors.IsNotFound(err) {
			logger.Error("failed to get InstallerSet: %s", err)
			return err
		}
		return nil
	} else {
		// If target namespace and version are not changed then check if spec
		// of TektonPipeline is changed by checking hash stored as annotation on
		// TektonInstallerSet with computing new hash of TektonPipeline Spec

		// Hash of TektonPipeline Spec
		expectedSpecHash, err := hash.Compute(obj.GetSpec())
		if err != nil {
			return err
		}

		// spec hash stored on installerSet
		lastAppliedHash := installedTIS.GetAnnotations()[v1alpha1.LastAppliedHashKey]

		if lastAppliedHash != expectedSpecHash {
			manifest := sh.Manifest
			// TODO:- Fix the transformation

			// if err := r.transform(ctx, &manifest, obj); err != nil {
			// 	// logger.Error("manifest transformation failed:  ", err)
			// 	return err
			// }

			// Update the spec hash
			current := installedTIS.GetAnnotations()
			current[v1alpha1.LastAppliedHashKey] = expectedSpecHash
			installedTIS.SetAnnotations(current)

			// Update the manifests
			installedTIS.Spec.Manifests = manifest.Resources()

			if _, err = sh.OperatorClientSet.OperatorV1alpha1().TektonInstallerSets().
				Update(ctx, installedTIS, metav1.UpdateOptions{}); err != nil {
				return err
			}

			// after updating installer set enqueue after a duration
			// to allow changes to get deployed
			return v1alpha1.REQUEUE_EVENT_AFTER
		}
	}
	return nil
}
