package tektonhub

import (
	"context"
	"github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"path/filepath"
)

func (r *Reconciler) GetManifest(ctx context.Context, th *v1alpha1.TektonHub, componentType string) (*manifestival.Manifest, error) {
	version := common.TargetVersion(th)
	hubDir := filepath.Join(common.ComponentDir(th), version)

	location := filepath.Join(hubDir, componentType)
	manifest, err := r.getManifest(ctx, th, location)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}
