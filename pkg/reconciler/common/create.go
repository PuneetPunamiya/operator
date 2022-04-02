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

type Service interface {
	Transform() error
}

type Shared struct {
	Obj               v1alpha1.TektonComponent
	Manifest          mf.Manifest
	CreatedByValue    string
	OperatorVersion   string
	OperatorClientSet clientset.Interface
}

func (sh *Shared) Create(t Service) (*v1alpha1.TektonInstallerSet, error) {
	err := t.Transform()
	if err != nil {
		return nil, err
	}

	// compute the hash of tektonpipeline spec and store as an annotation
	// in further reconciliation we compute hash of tp spec and check with
	// annotation, if they are same then we skip updating the object
	// otherwise we update the manifest
	specHash, err := hash.Compute(sh.Obj.GetSpec())
	if err != nil {
		return nil, err
	}

	tis := sh.makeInstallerSet(specHash)
	createdIs, err := sh.OperatorClientSet.OperatorV1alpha1().TektonInstallerSets().
		Create(context.Background(), tis, metav1.CreateOptions{})
	if err != nil {
		return tis, err
	}
	fmt.Println("Basic implementation done ")
	return createdIs, nil

}

func (sh *Shared) makeInstallerSet(tpSpecHash string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(sh.Obj, sh.Obj.GroupVersionKind())
	return &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", v1alpha1.PipelineResourceName),
			Labels: map[string]string{
				v1alpha1.CreatedByKey:      sh.CreatedByValue,
				v1alpha1.InstallerSetType:  v1alpha1.PipelineResourceName, // TODO: improve
				v1alpha1.ReleaseVersionKey: sh.OperatorVersion,
			},
			Annotations: map[string]string{
				v1alpha1.TargetNamespaceKey: sh.Obj.GetSpec().GetTargetNamespace(),
				v1alpha1.LastAppliedHashKey: tpSpecHash,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Spec: v1alpha1.TektonInstallerSetSpec{
			Manifests: sh.Manifest.Resources(),
		},
	}
}
