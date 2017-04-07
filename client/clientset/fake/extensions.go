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
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"

	"k8s.io/helm/client/clientset"
)

type FakeExtensionClient struct {
	*testing.Fake
}

func NewFakeExtensionClient(objects ...runtime.Object) *FakeExtensionClient {
	o := testing.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Group == "helm.sh" {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
	}
	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o, registered.RESTMapper()))
	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))
	return &FakeExtensionClient{&fakePtr}
}

func (m *FakeExtensionClient) Releases(ns string) clientset.ReleaseInterface {
	return &FakeRelease{Fake: m.Fake, ns: ns}
}
