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

package api

import (
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/runtime"
	versionedwatch "k8s.io/kubernetes/pkg/watch/versioned"
)

// SchemeGroupVersion is group version used to register these objects
var V1alpha1SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

var (
	V1alpha1SchemeBuilder = runtime.NewSchemeBuilder(v1addKnownTypes)
	V1betaAddToScheme     = V1alpha1SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func v1addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(V1alpha1SchemeGroupVersion,
		&Release{},
		&ReleaseList{},

		&v1.ListOptions{},
		&v1.DeleteOptions{},
	)
	versionedwatch.AddToGroupVersion(scheme, V1alpha1SchemeGroupVersion)
	return nil
}
