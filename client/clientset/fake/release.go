/*
Copyright 2017 AppsCode Inc. All rights reserved.
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

package fake

import (
	aci "k8s.io/helm/api"
	"k8s.io/kubernetes/pkg/api"
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeRelease struct {
	Fake *testing.Fake
	ns   string
}

var resource = schema.GroupVersionResource{Group: "helm.sh", Version: "v1alpha1", Resource: "releases"}

// Get returns the Release by name.
func (mock *FakeRelease) Get(name string) (*aci.Release, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(resource, mock.ns, name), &aci.Release{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Release), err
}

// List returns the a of Releases.
func (mock *FakeRelease) List(opts api.ListOptions) (*aci.ReleaseList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(resource, mock.ns, opts), &aci.Release{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &aci.ReleaseList{}
	for _, item := range obj.(*aci.ReleaseList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Release.
func (mock *FakeRelease) Create(r *aci.Release) (*aci.Release, error) {
	if r != nil {
		r.Namespace = mock.ns
	}
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(resource, mock.ns, r), &aci.Release{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Release), err
}

// Update updates a Release.
func (mock *FakeRelease) Update(r *aci.Release) (*aci.Release, error) {
	if r != nil {
		r.Namespace = mock.ns
	}
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(resource, mock.ns, r), &aci.Release{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Release), err
}

// Delete deletes a Release by name.
func (mock *FakeRelease) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(resource, mock.ns, name), &aci.Release{})

	return err
}

func (mock *FakeRelease) UpdateStatus(r *aci.Release) (*aci.Release, error) {
	if r != nil {
		r.Namespace = mock.ns
	}
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(resource, "status", mock.ns, r), &aci.Release{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Release), err
}

func (mock *FakeRelease) Watch(opts api.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(resource, mock.ns, opts))
}
