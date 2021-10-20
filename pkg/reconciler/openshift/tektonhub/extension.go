/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tektonhub

import (
	"context"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/go-logr/zapr"
	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/openshift/client-go/route/clientset/versioned/scheme"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/client/clientset/versioned"
	operatorclient "github.com/tektoncd/operator/pkg/client/injection/client"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"go.uber.org/zap"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

func OpenShiftExtension(ctx context.Context) common.Extension {
	logger := logging.FromContext(ctx)
	mfclient, err := mfc.NewClient(injection.GetConfig(ctx))
	if err != nil {
		logger.Fatalw("error creating client from injected config", zap.Error(err))
	}
	mflogger := zapr.NewLogger(logger.Named("manifestival").Desugar())
	manifest, err := mf.ManifestFrom(mf.Slice{}, mf.UseClient(mfclient), mf.UseLogger(mflogger))
	if err != nil {
		logger.Fatalw("error creating initial manifest", zap.Error(err))
	}

	ext := openshiftExtension{
		operatorClientSet: operatorclient.Get(ctx),
		manifest:          manifest,
	}
	return ext
}

type openshiftExtension struct {
	operatorClientSet versioned.Interface
	manifest          mf.Manifest
}

func (oe openshiftExtension) Transformers(comp v1alpha1.TektonComponent) []mf.Transformer {
	return nil
}
func (oe openshiftExtension) PreReconcile(ctx context.Context, tc v1alpha1.TektonComponent) error {
	return nil
}
func (oe openshiftExtension) PostReconcile(ctx context.Context, tc v1alpha1.TektonComponent) error {

	th := tc.(*v1alpha1.TektonHub)
	logger := logging.FromContext(ctx)

	koDataDir := os.Getenv(common.KoEnvKey)
	hubDir := filepath.Join(koDataDir, "hub", common.TargetVersion(th), "api")

	manifest := oe.manifest.Append()

	ownerRef := *metav1.NewControllerRef(th, th.GroupVersionKind())
	if err := common.AppendManifest(&manifest, hubDir); err != nil {
		return err
	}
	manifest, err := manifest.Transform(
		injectOwner([]metav1.OwnerReference{ownerRef}),
		changeNamespace("openshift-pipelines"),
	)
	if err != nil {
		logger.Error("failed to transform manifest")
		return err
	}

	if err := manifest.Filter(mf.ByKind("Route")).Apply(); err != nil {
		return err
	}

	route, err := getRouteHost(&manifest)
	if err != nil {
		return err
	}

	th.Status.SetApiRoute(route)

	return nil
}
func (oe openshiftExtension) Finalize(context.Context, v1alpha1.TektonComponent) error {
	return nil
}

func injectOwner(owner []v1.OwnerReference) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := u.GetKind()
		if kind == "CustomResourceDefinition" {
			return nil
		}
		u.SetOwnerReferences(owner)
		return nil
	}
}

func changeNamespace(targetNamespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetNamespace() != targetNamespace {
			u.SetNamespace(targetNamespace)
			return nil
		}
		return nil
	}
}

func getRouteHost(manifest *mf.Manifest) (string, error) {
	var hostUrl string
	for _, r := range manifest.Filter(mf.ByKind("Route")).Resources() {
		u, err := manifest.Client.Get(&r)
		if err != nil {
			return "", err
		}
		if u.GetName() == "api" {
			route := &routev1.Route{}
			if err := scheme.Scheme.Convert(u, route, nil); err != nil {
				return "", err
			}
			hostUrl = route.Spec.Host
		}
	}
	return hostUrl, nil
}
