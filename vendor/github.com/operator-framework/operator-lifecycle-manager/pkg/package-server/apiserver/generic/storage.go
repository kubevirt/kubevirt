// Copyright 2018 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generic

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/registry/rest"
	generic "k8s.io/apiserver/pkg/server"

	apps "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/apps/install"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/apps/v1alpha1"
	operators "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/install"
	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/provider"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/storage"
)

var (
	// Scheme contains the types needed by the resource metrics API.
	Scheme = runtime.NewScheme()
	// Codecs is a codec factory for serving the resource metrics API.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	operators.Install(Scheme)
	apps.Install(Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ProviderConfig holds the providers for packagemanifests.
type ProviderConfig struct {
	Provider provider.PackageManifestProvider
}

// BuildStorage constructs APIGroupInfo for the packages.apps.redhat.com and packages.operators.coreos.com API groups.
func BuildStorage(providers *ProviderConfig) []generic.APIGroupInfo {

	// Build storage for packages.operators.coreos.com
	operatorInfo := generic.NewDefaultAPIGroupInfo(v1.Group, Scheme, metav1.ParameterCodec, Codecs)
	operatorStorage := storage.NewStorage(v1.Resource("packagemanifests"), providers.Provider, Scheme)
	operatorResources := map[string]rest.Storage{
		"packagemanifests": operatorStorage,
	}
	operatorInfo.VersionedResourcesStorageMap[v1.Version] = operatorResources

	// Build storage for packages.apps.redhat.com
	appInfo := generic.NewDefaultAPIGroupInfo(v1alpha1.Group, Scheme, metav1.ParameterCodec, Codecs)

	// Use storage for package.operators.coreos.com since types are identical
	appResources := map[string]rest.Storage{
		"packagemanifests": operatorStorage,
	}
	appInfo.VersionedResourcesStorageMap[v1alpha1.Version] = appResources

	return []generic.APIGroupInfo{
		operatorInfo,
		appInfo,
	}
}

// InstallStorage builds the storage for the packages.apps.redhat.com and packages.operators.coreos.com API groups and then installs them into the given API server.
func InstallStorage(providers *ProviderConfig, server *generic.GenericAPIServer) error {
	errs := []error{}
	groups := BuildStorage(providers)
	for i := 0; i < len(groups); i++ {
		info := groups[i]
		errs = append(errs, server.InstallAPIGroup(&info))
	}

	return utilerrors.NewAggregate(errs)
}
