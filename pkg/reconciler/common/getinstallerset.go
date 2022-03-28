package common

import (
	"context"

	clientset "github.com/tektoncd/operator/pkg/client/clientset/versioned"
	"github.com/tektoncd/operator/pkg/reconciler/kubernetes/tektoninstallerset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetInstallerSet(ctx context.Context, ls metav1.LabelSelector, op clientset.Interface) (string, error) {
	// Check if an tekton installer set already exists, if not then create
	labelSelector, err := LabelSelector(ls)
	if err != nil {
		return "", err
	}
	existingInstallerSet, err := tektoninstallerset.CurrentInstallerSetName(ctx, op, labelSelector)
	if err != nil {
		return "", err
	}
	return existingInstallerSet, nil

}
