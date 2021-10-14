package tektonhub

import (
	"context"
	"fmt"

	k8s_ctrl "github.com/tektoncd/operator/pkg/reconciler/kubernetes/tektonhub"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	fmt.Println("Openshift Controller entry")
	return k8s_ctrl.NewExtendedController(OpenShiftExtension)(ctx, cmw)
}
