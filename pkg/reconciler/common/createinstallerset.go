package common

import (
	"context"
	"fmt"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	clientset "github.com/tektoncd/operator/pkg/client/clientset/versioned"
	"github.com/tektoncd/operator/pkg/reconciler/shared/hash"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SharedInstaller struct {
	OperatorClientSet clientset.Interface
	Manifest          mf.Manifest
	OperatorVersion   string
	InstallerSetType  string
}

func (sh SharedInstaller) CreateInstallerSet(ctx context.Context, obj v1alpha1.TektonComponent) (*v1alpha1.TektonInstallerSet, error) {
	specHash, err := hash.Compute(obj.GetSpec())
	if err != nil {
		return nil, err
	}

	// create installer set
	tis := sh.makeInstallerSet(obj, sh.Manifest, specHash)
	createdIs, err := sh.OperatorClientSet.OperatorV1alpha1().TektonInstallerSets().
		Create(ctx, tis, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return createdIs, nil
}

func (sh SharedInstaller) makeInstallerSet(obj v1alpha1.TektonComponent, manifest mf.Manifest, specHash string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(obj, obj.GroupVersionKind())
	return &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", sh.InstallerSetType),
			Labels: map[string]string{
				v1alpha1.CreatedByKey:      obj.GroupVersionKind().Kind,
				v1alpha1.InstallerSetType:  sh.InstallerSetType,
				v1alpha1.ReleaseVersionKey: sh.OperatorVersion,
			},
			Annotations: map[string]string{
				v1alpha1.TargetNamespaceKey: obj.GetSpec().GetTargetNamespace(),
				v1alpha1.LastAppliedHashKey: specHash,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: v1alpha1.TektonInstallerSetSpec{
			Manifests: manifest.Resources(),
		},
	}
}
