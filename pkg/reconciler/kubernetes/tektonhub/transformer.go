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
		fmt.Println("TektonHub----------->", u.GetOwnerReferences())
		return nil
	}
}
