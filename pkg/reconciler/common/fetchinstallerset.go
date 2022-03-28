package common

import (
	"context"

	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	clientset "github.com/tektoncd/operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FetchInstallerSet(ctx context.Context, existingInstallerSet string, op clientset.Interface) (*v1alpha1.TektonInstallerSet, error) {
	installedTIS, err := op.OperatorV1alpha1().TektonInstallerSets().
		Get(ctx, existingInstallerSet, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return installedTIS, nil
}
