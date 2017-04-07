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
	"k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5/fake"
	"k8s.io/kubernetes/pkg/runtime"
)

type ClientSets struct {
	*fake.Clientset
	ExtensionClient *FakeExtensionClient
}

func NewFakeClient(objects ...runtime.Object) *ClientSets {
	return &ClientSets{
		Clientset:       fake.NewSimpleClientset(objects...),
		ExtensionClient: NewFakeExtensionClient(objects...),
	}
}
