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

package install

import (
	aci "k8s.io/helm/api"
	"k8s.io/kubernetes/pkg/apimachinery/announced"
	"k8s.io/kubernetes/pkg/util/sets"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  aci.GroupName,
			VersionPreferenceOrder:     []string{aci.V1alpha1SchemeGroupVersion.Version},
			ImportPrefix:               "k8s.io/helm/api",
			RootScopedKinds:            sets.NewString("ThirdPartyResource"),
			AddInternalObjectsToScheme: aci.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			aci.V1alpha1SchemeGroupVersion.Version: aci.V1betaAddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
}
