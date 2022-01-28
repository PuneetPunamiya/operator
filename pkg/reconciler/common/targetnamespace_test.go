package common

import (
	"context"
	"fmt"
	"path"
	"testing"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
)

type fakeClient struct {
	err            error
	getErr         error
	createErr      error
	resourcesExist bool
	creates        []unstructured.Unstructured
	deletes        []unstructured.Unstructured
}

func (f *fakeClient) Get(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	var resource *unstructured.Unstructured
	if f.resourcesExist {
		resource = &unstructured.Unstructured{}
	}
	return resource, f.getErr
}

func (f *fakeClient) Create(obj *unstructured.Unstructured, options ...mf.ApplyOption) error {
	obj.SetAnnotations(nil) // Deleting the extra annotation. Irrelevant for the test.
	f.creates = append(f.creates, *obj)
	return f.createErr
}

func (f *fakeClient) Delete(obj *unstructured.Unstructured, options ...mf.DeleteOption) error {
	f.deletes = append(f.deletes, *obj)
	return f.err
}

func (f *fakeClient) Update(obj *unstructured.Unstructured, options ...mf.ApplyOption) error {
	return f.err
}

func TestCreateTargetNamespace(t *testing.T) {
	targetNamespace := "test-ns"
	component := &v1alpha1.TektonPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
		},
		Spec: v1alpha1.TektonPipelineSpec{
			CommonSpec: v1alpha1.CommonSpec{
				TargetNamespace: targetNamespace,
			},
		},
	}
	fakeClientset := fake.NewSimpleClientset()

	err := CreateTargetNamespace(nil, component, fakeClientset)
	assert.Equal(t, err, nil)

	ns, err := fakeClientset.CoreV1().Namespaces().Get(context.TODO(), targetNamespace, metav1.GetOptions{})
	assert.Equal(t, err, nil)
	assert.Equal(t, ns.ObjectMeta.Name, targetNamespace)
}

func TestTargetNamespaceNotFound(t *testing.T) {
	component := &v1alpha1.TektonPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
		},
		Spec: v1alpha1.TektonPipelineSpec{
			CommonSpec: v1alpha1.CommonSpec{},
		},
	}
	fakeClientset := fake.NewSimpleClientset()

	err := CreateTargetNamespace(nil, component, fakeClientset)

	_, err = fakeClientset.CoreV1().Namespaces().Get(context.TODO(), "foo", metav1.GetOptions{})
	assert.Equal(t, err.Error(), `namespaces "foo" not found`)
}

func TestOperatorVersionCreateConfigMap(t *testing.T) {
	testData := path.Join("testdata", "test-create-operator-version-config-map.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))

	client := &fakeClient{}
	mani, err := mf.ManifestFrom(mf.Slice(manifest.Resources()), mf.UseClient(client))

	targetNamespace := "test-ns"
	component := &v1alpha1.TektonPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
		},
		Spec: v1alpha1.TektonPipelineSpec{
			CommonSpec: v1alpha1.CommonSpec{
				TargetNamespace: targetNamespace,
			},
		},
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: component.GetSpec().GetTargetNamespace(),
			Labels: map[string]string{
				"operator.tekton.dev/targetNamespace": "true",
			},
		},
	}

	fakeClientset := fake.NewSimpleClientset()

	_, err = fakeClientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	assert.Equal(t, err, nil)

	err = CreateOperatorVersionConfigMap(mani, component)

	ns, err := fakeClientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	fmt.Println(ns.Items[0].Name)

	_, err = fakeClientset.CoreV1().ConfigMaps(targetNamespace).Get(context.TODO(), "operators-info", metav1.GetOptions{})
	assert.Equal(t, err, nil)

	// assert.Equal(t, cm.ObjectMeta.Name, "operators-info")
}
