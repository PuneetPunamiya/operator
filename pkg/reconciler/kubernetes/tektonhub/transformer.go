package tektonhub

import (
	"fmt"

	mf "github.com/manifestival/manifestival"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

func changeNamespaceName(targetNamespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetName() != targetNamespace {
			u.SetName(targetNamespace)
			return nil
		}
		return nil
	}
}

func changeNamespace(targetNamespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetNamespace() != targetNamespace {
			u.SetNamespace(targetNamespace)
			fmt.Println(u.GetNamespace())
			return nil
		}
		return nil
	}
}
