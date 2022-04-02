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

type ShareResources interface {
	Transform() error
}

type PipelineInstaller struct {
	Ctx               context.Context
	Manifest          mf.Manifest
	Extension         Extension
	Component         v1alpha1.TektonComponent
	OperatorVersion   string
	CreatedByValue    string
	InstallerSetType  string
	OperatorClientSet clientset.Interface
	Resources         ShareResources
}

func CreateInstallerSet(p PipelineInstaller) (*v1alpha1.TektonInstallerSet, error) {
	err := p.Resources.Transform()
	if err != nil {
		return nil, err
	}

	// compute the hash of tektonpipeline spec and store as an annotation
	// in further reconciliation we compute hash of tp spec and check with
	// annotation, if they are same then we skip updating the object
	// otherwise we update the manifest
	specHash, err := hash.Compute(p.Component.GetSpec())
	if err != nil {
		return nil, err
	}

	// create installer set
	tis := p.makeInstallerSet(specHash)
	createdIs, err := p.OperatorClientSet.OperatorV1alpha1().TektonInstallerSets().
		Create(p.Ctx, tis, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return createdIs, nil
}

func (sh PipelineInstaller) makeInstallerSet(specHash string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(sh.Component, sh.Component.GroupVersionKind())
	return &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", sh.InstallerSetType),
			Labels: map[string]string{
				v1alpha1.CreatedByKey:      sh.Component.GroupVersionKind().Kind,
				v1alpha1.InstallerSetType:  sh.InstallerSetType,
				v1alpha1.ReleaseVersionKey: sh.OperatorVersion,
			},
			Annotations: map[string]string{
				v1alpha1.TargetNamespaceKey: sh.Component.GetSpec().GetTargetNamespace(),
				v1alpha1.LastAppliedHashKey: specHash,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: v1alpha1.TektonInstallerSetSpec{
			Manifests: sh.Manifest.Resources(),
		},
	}
}
