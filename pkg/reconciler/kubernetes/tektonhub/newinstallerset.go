package tektonhub

import (
	"context"
	"fmt"
	"github.com/manifestival/manifestival"
	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"github.com/tektoncd/operator/pkg/reconciler/kubernetes/tektoninstallerset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Reconciler) CheckIfInstallerSetExists(ctx context.Context, componentInstallerSetType string) (string, error) {
	labels := r.getLabels(componentInstallerSetType)
	labelSelector, err := common.LabelSelector(labels)
	if err != nil {
		return "", err
	}

	compInstallerSet, err := tektoninstallerset.CurrentInstallerSetName(ctx, r.operatorClientSet, labelSelector)
	if err != nil {
		return "", err
	}

	return compInstallerSet, nil
}
func (r *Reconciler) CreatePVCInstallerSet(ctx context.Context, th *v1alpha1.TektonHub, manifest *manifestival.Manifest) (*v1alpha1.TektonInstallerSet, error) {

	// filter only secret for this installerset as this needs
	// to be restored over upgrade
	pvcManifest := manifest.Filter(mf.ByKind("PersistentVolumeClaim"))
	if _, err := r.transform(ctx, pvcManifest, th); err != nil {
		th.Status.MarkNotReady("transformation failed: " + err.Error())
		return nil, err
	}

	tis := MakeInstallerSet(th, pvcManifest, dbPVC)

	// create installer set
	createdIs, err := r.operatorClientSet.OperatorV1alpha1().TektonInstallerSets().
		Create(ctx, tis, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return createdIs, nil
}

func MakeInstallerSet(th *v1alpha1.TektonHub, manifest mf.Manifest, componentInstallerSetType string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(th, th.GetGroupVersionKind())

	is := &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", componentInstallerSetType),
			Labels: map[string]string{
				v1alpha1.CreatedByKey:      createdByValue,
				v1alpha1.InstallerSetType:  componentInstallerSetType,
				v1alpha1.ReleaseVersionKey: common.TargetVersion(th),
			},
			Annotations: map[string]string{
				v1alpha1.TargetNamespaceKey: namespace,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: v1alpha1.TektonInstallerSetSpec{
			Manifests: manifest.Resources(),
		},
	}

	return is
}
