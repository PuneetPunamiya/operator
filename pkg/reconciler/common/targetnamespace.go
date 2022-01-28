package common

import (
	"context"
	"fmt"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateTargetNamespace(labels map[string]string, obj v1alpha1.TektonComponent, kubeClientSet kubernetes.Interface) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetSpec().GetTargetNamespace(),
			Labels: map[string]string{
				"operator.tekton.dev/targetNamespace": "true",
			},
		},
	}
	for key, value := range labels {
		namespace.Labels[key] = value
	}

	if _, err := kubeClientSet.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateOperatorVersionConfigMap(manifest mf.Manifest, obj v1alpha1.TektonComponent) error {
	// koDataDir := os.Getenv(KoEnvKey)
	// operatorDir := filepath.Join(koDataDir, "info")

	// if err := AppendManifest(&manifest, operatorDir); err != nil {
	// 	return err
	// }

	manifest, err := manifest.Transform(
		mf.InjectNamespace(obj.GetSpec().GetTargetNamespace()),
		mf.InjectOwner(obj),
	)
	if err != nil {
		return err
	}

	if err = manifest.Apply(); err != nil {
		fmt.Println("error logged here")
		return err
	}

	return nil
}
