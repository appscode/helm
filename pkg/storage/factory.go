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

package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/graymeta/stow"
	"github.com/graymeta/stow/azure"
	gcs "github.com/graymeta/stow/google"
	"github.com/graymeta/stow/s3"
	"github.com/graymeta/stow/swift"
	"k8s.io/kubernetes/pkg/api"
	kberrs "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"

	rapi "k8s.io/helm/api"
	rcs "k8s.io/helm/client/clientset"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/tiller/environment"
)

type StoreType string

const (
	StorageMemory         StoreType = "memory"
	StorageConfigMap      StoreType = "configmap"
	StorageInlineTPR      StoreType = "inline-tpr"
	StorageObjectStoreTPR StoreType = "object-store-tpr"
)

type StoreOptions struct {
	StoreType StoreType

	ObjectStoreProvider      string
	S3ConfigAccessKeyID      string
	S3ConfigEndpoint         string
	S3ConfigRegion           string
	S3ConfigSecretKey        string
	GCSConfigJSONKeyPath     string
	GCSConfigProjectId       string
	AzureConfigAccount       string
	AzureConfigKey           string
	SwiftConfigKey           string
	SwiftConfigTenantAuthURL string
	SwiftConfigTenantName    string
	SwiftConfigUsername      string

	Container     string
	StoragePrefix string
}

func NewStorage(client *kube.Client, opts StoreOptions) (*Storage, error) {
	clientcfg, err := client.ClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot initialize Kubernetes connection: %s\n", err)
		os.Exit(1)
	}
	clientset, err := client.ClientSet()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot initialize Kubernetes connection: %s\n", err)
		os.Exit(1)
	}

	switch opts.StoreType {
	case StorageMemory:
		return Init(driver.NewMemory()), nil
	case StorageConfigMap:
		return Init(driver.NewConfigMaps(clientset.Core().ConfigMaps(namespace()))), nil
	case StorageInlineTPR:
		ensureResource(clientset)
		cs := rcs.NewExtensionsForConfigOrDie(clientcfg)
		return Init(driver.NewReleases(cs.Release(namespace()))), nil
	case StorageObjectStoreTPR:
		ensureResource(clientset)
		stowCfg := stow.ConfigMap{}
		switch opts.ObjectStoreProvider {
		case s3.Kind:
			if opts.S3ConfigAccessKeyID != "" {
				stowCfg[s3.ConfigAccessKeyID] = opts.S3ConfigAccessKeyID
			}
			if opts.S3ConfigEndpoint != "" {
				stowCfg[s3.ConfigEndpoint] = opts.S3ConfigEndpoint
			}
			if opts.S3ConfigRegion != "" {
				stowCfg[s3.ConfigRegion] = opts.S3ConfigRegion
			}
			if opts.S3ConfigSecretKey != "" {
				stowCfg[s3.ConfigSecretKey] = opts.S3ConfigSecretKey
			}
		case gcs.Kind:
			if opts.GCSConfigJSONKeyPath != "" {
				jsonKey, err := ioutil.ReadFile(opts.GCSConfigJSONKeyPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Cannot read json key file: %v\n", err)
					os.Exit(1)
				}
				stowCfg[gcs.ConfigJSON] = string(jsonKey)
			}
			if opts.GCSConfigProjectId != "" {
				stowCfg[gcs.ConfigProjectId] = opts.GCSConfigProjectId
			}
		case azure.Kind:
			if opts.AzureConfigAccount != "" {
				stowCfg[azure.ConfigAccount] = opts.AzureConfigAccount
			}
			if opts.AzureConfigKey != "" {
				stowCfg[azure.ConfigKey] = opts.AzureConfigKey
			}
		case swift.Kind:
			if opts.SwiftConfigKey != "" {
				stowCfg[swift.ConfigKey] = opts.SwiftConfigKey
			}
			if opts.SwiftConfigTenantAuthURL != "" {
				stowCfg[swift.ConfigTenantAuthURL] = opts.SwiftConfigTenantAuthURL
			}
			if opts.SwiftConfigTenantName != "" {
				stowCfg[swift.ConfigTenantName] = opts.SwiftConfigTenantName
			}
			if opts.SwiftConfigUsername != "" {
				stowCfg[swift.ConfigUsername] = opts.SwiftConfigUsername
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown provider: %v\n", opts.ObjectStoreProvider)
			os.Exit(1)
		}
		loc, err := stow.Dial(opts.ObjectStoreProvider, stowCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot connect to object store: %v\n", err)
			os.Exit(1)
		}
		c, err := loc.Container(opts.Container)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot find container: %v\n", err)
			os.Exit(1)
		}
		cs := rcs.NewExtensionsForConfigOrDie(clientcfg)
		return Init(driver.NewObjectStoreReleases(cs.Release(namespace()), c, opts.StoragePrefix)), nil
	}
	return nil, fmt.Errorf("Unknow store type %v", opts.StoreType)
}

// namespace returns the namespace of tiller
func namespace() string {
	if ns := os.Getenv("TILLER_NAMESPACE"); ns != "" {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return environment.DefaultTillerNamespace
}

func ensureResource(clientset *internalclientset.Clientset) {
	_, err := clientset.Extensions().ThirdPartyResources().Get("release." + rapi.V1alpha1SchemeGroupVersion.Group)
	if kberrs.IsNotFound(err) {
		tpr := &extensions.ThirdPartyResource{
			TypeMeta: unversioned.TypeMeta{
				APIVersion: "extensions/v1alpha1",
				Kind:       "ThirdPartyResource",
			},
			ObjectMeta: api.ObjectMeta{
				Name: "release." + rapi.V1alpha1SchemeGroupVersion.Group,
			},
			Versions: []extensions.APIVersion{
				{
					Name: rapi.V1alpha1SchemeGroupVersion.Version,
				},
			},
		}
		_, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create third party resource: %s\n", err)
			os.Exit(1)
		}
	}
}
