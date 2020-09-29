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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTektonDashboards implements TektonDashboardInterface
type FakeTektonDashboards struct {
	Fake *FakeOperatorV1alpha1
}

var tektondashboardsResource = schema.GroupVersionResource{Group: "operator.tekton.dev", Version: "v1alpha1", Resource: "tektondashboards"}

var tektondashboardsKind = schema.GroupVersionKind{Group: "operator.tekton.dev", Version: "v1alpha1", Kind: "TektonDashboard"}

// Get takes name of the tektonDashboard, and returns the corresponding tektonDashboard object, and an error if there is any.
func (c *FakeTektonDashboards) Get(name string, options v1.GetOptions) (result *v1alpha1.TektonDashboard, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(tektondashboardsResource, name), &v1alpha1.TektonDashboard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TektonDashboard), err
}

// List takes label and field selectors, and returns the list of TektonDashboards that match those selectors.
func (c *FakeTektonDashboards) List(opts v1.ListOptions) (result *v1alpha1.TektonDashboardList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(tektondashboardsResource, tektondashboardsKind, opts), &v1alpha1.TektonDashboardList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.TektonDashboardList{ListMeta: obj.(*v1alpha1.TektonDashboardList).ListMeta}
	for _, item := range obj.(*v1alpha1.TektonDashboardList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tektonDashboards.
func (c *FakeTektonDashboards) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(tektondashboardsResource, opts))
}

// Create takes the representation of a tektonDashboard and creates it.  Returns the server's representation of the tektonDashboard, and an error, if there is any.
func (c *FakeTektonDashboards) Create(tektonDashboard *v1alpha1.TektonDashboard) (result *v1alpha1.TektonDashboard, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(tektondashboardsResource, tektonDashboard), &v1alpha1.TektonDashboard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TektonDashboard), err
}

// Update takes the representation of a tektonDashboard and updates it. Returns the server's representation of the tektonDashboard, and an error, if there is any.
func (c *FakeTektonDashboards) Update(tektonDashboard *v1alpha1.TektonDashboard) (result *v1alpha1.TektonDashboard, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(tektondashboardsResource, tektonDashboard), &v1alpha1.TektonDashboard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TektonDashboard), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeTektonDashboards) UpdateStatus(tektonDashboard *v1alpha1.TektonDashboard) (*v1alpha1.TektonDashboard, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(tektondashboardsResource, "status", tektonDashboard), &v1alpha1.TektonDashboard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TektonDashboard), err
}

// Delete takes name of the tektonDashboard and deletes it. Returns an error if one occurs.
func (c *FakeTektonDashboards) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(tektondashboardsResource, name), &v1alpha1.TektonDashboard{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTektonDashboards) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(tektondashboardsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.TektonDashboardList{})
	return err
}

// Patch applies the patch and returns the patched tektonDashboard.
func (c *FakeTektonDashboards) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.TektonDashboard, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(tektondashboardsResource, name, pt, data, subresources...), &v1alpha1.TektonDashboard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.TektonDashboard), err
}
