/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 * Copyright 2018 The Kubernetes Authors.
 *
 */

package kubecli

import (
	"flag"
	"os"
	"sync"

	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	clonev1alpha1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/clone/v1alpha1"

	secv1 "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	"github.com/spf13/pflag"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	cdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	k8ssnapshotclient "kubevirt.io/client-go/generated/external-snapshotter/clientset/versioned"
	generatedclient "kubevirt.io/client-go/generated/kubevirt/clientset/versioned"
	migrationsv1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/migrations/v1alpha1"
	networkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned"
	promclient "kubevirt.io/client-go/generated/prometheus-operator/clientset/versioned"
)

var (
	kubeconfig string
	master     string
)

var (
	SchemeBuilder  runtime.SchemeBuilder
	Scheme         *runtime.Scheme
	Codecs         serializer.CodecFactory
	ParameterCodec runtime.ParameterCodec
)

func init() {
	// This allows consumers of the KubeVirt client go package to
	// customize what version the client uses. Without specifying a
	// version, all versions are registered. While this techincally
	// file to register all versions, so k8s ecosystem libraries
	// do not work well with this. By explicitly setting the env var,
	// consumers of our client go can avoid these scenarios by only
	// registering a single version
	registerVersion := os.Getenv(v1.KubeVirtClientGoSchemeRegistrationVersionEnvVar)
	if registerVersion != "" {
		SchemeBuilder = runtime.NewSchemeBuilder(v1.AddKnownTypesGenerator([]schema.GroupVersion{schema.GroupVersion{Group: core.GroupName, Version: registerVersion}}))
	} else {
		SchemeBuilder = runtime.NewSchemeBuilder(v1.AddKnownTypesGenerator(v1.GroupVersions))
	}
	Scheme = runtime.NewScheme()
	AddToScheme := SchemeBuilder.AddToScheme
	Codecs = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
	AddToScheme(Scheme)
	AddToScheme(scheme.Scheme)
}

type RestConfigHookFunc func(*rest.Config)

var restConfigHooks []RestConfigHookFunc
var restConfigHooksLock sync.Mutex

var virtclient KubevirtClient
var once sync.Once

// Init adds the default `kubeconfig` and `master` flags. It is not added by default to allow integration into
// the different controller generators which normally add these flags too.
func Init() {
	if flag.CommandLine.Lookup("kubeconfig") == nil {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	if flag.CommandLine.Lookup("master") == nil {
		flag.StringVar(&master, "master", "", "master url")
	}
}

func RegisterRestConfigHook(fn RestConfigHookFunc) {
	restConfigHooksLock.Lock()
	defer restConfigHooksLock.Unlock()

	restConfigHooks = append(restConfigHooks, fn)
}

func executeRestConfigHooks(config *rest.Config) {
	restConfigHooksLock.Lock()
	defer restConfigHooksLock.Unlock()

	for _, hookFn := range restConfigHooks {
		hookFn(config)
	}
}

func FlagSet() *flag.FlagSet {
	set := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	set.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	set.StringVar(&master, "master", "", "master url")
	return set
}

func GetKubevirtSubresourceClientFromFlags(master string, kubeconfig string) (KubevirtClient, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &v1.SubresourceStorageGroupVersion
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	coreClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	generatedKubeVirtClient, err := generatedclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cdiClient, err := cdiclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	networkClient, err := networkclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	extensionsClient, err := extclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	secClient, err := secv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	routeClient, err := routev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	prometheusClient, err := promclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	snapshotClient, err := k8ssnapshotclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	migrationsClient, err := migrationsv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cloneClient, err := clonev1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &kubevirt{
		master,
		kubeconfig,
		restClient,
		config,
		generatedKubeVirtClient,
		cdiClient,
		networkClient,
		extensionsClient,
		secClient,
		routeClient,
		discoveryClient,
		prometheusClient,
		snapshotClient,
		dynamicClient,
		migrationsClient,
		cloneClient,
		coreClient,
	}, nil
}

// DefaultClientConfig creates a clientcmd.ClientConfig with the following hierarchy:
//
//  1. Use the kubeconfig builder.  The number of merges and overrides here gets a little crazy.  Stay with me.
//
//  1. Merge the kubeconfig itself.  This is done with the following hierarchy rules:
//
//  1. CommandLineLocation - this parsed from the command line, so it must be late bound.  If you specify this,
//     then no other kubeconfig files are merged.  This file must exist.
//
//  2. If $KUBECONFIG is set, then it is treated as a list of files that should be merged.
//
//  3. HomeDirectoryLocation
//     Empty filenames are ignored.  Files with non-deserializable content produced errors.
//     The first file to set a particular value or map key wins and the value or map key is never changed.
//     This means that the first file to set CurrentContext will have its context preserved.  It also means
//     that if two files specify a "red-user", only values from the first file's red-user are used.  Even
//     non-conflicting entries from the second file's "red-user" are discarded.
//
//  2. Determine the context to use based on the first hit in this chain
//
//  1. command line argument - again, parsed from the command line, so it must be late bound
//
//  2. CurrentContext from the merged kubeconfig file
//
//  3. Empty is allowed at this stage
//
//  3. Determine the cluster info and auth info to use.  At this point, we may or may not have a context.  They
//     are built based on the first hit in this chain.  (run it twice, once for auth, once for cluster)
//
//  1. command line argument
//
//  2. If context is present, then use the context value
//
//  3. Empty is allowed
//
//  4. Determine the actual cluster info to use.  At this point, we may or may not have a cluster info.  Build
//     each piece of the cluster info based on the chain:
//
//  1. command line argument
//
//  2. If cluster info is present and a value for the attribute is present, use it.
//
//  3. If you don't have a server location, bail.
//
//  5. Auth info is build using the same rules as cluster info, EXCEPT that you can only have one authentication
//     technique per auth info.  The following conditions result in an error:
//
//  1. If there are two conflicting techniques specified from the command line, fail.
//
//  2. If the command line does not specify one, and the auth info has conflicting techniques, fail.
//
//  3. If the command line specifies one and the auth info specifies another, honor the command line technique.
//
//  2. Use default values and potentially prompt for auth information
//
//     However, if it appears that we're running in a kubernetes cluster
//     container environment, then run with the auth info kubernetes mounted for
//     us. Specifically:
//     The env vars KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT are
//     set, and the file /var/run/secrets/kubernetes.io/serviceaccount/token
//     exists and is not a directory.
//
// Initially copied from https://github.com/kubernetes/kubernetes/blob/09f321c80bfc9bca63a5530b56d7a1a3ba80ba9b/pkg/kubectl/cmd/util/factory_client_access.go#L174
func DefaultClientConfig(flags *pflag.FlagSet) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// use the standard defaults for this client command
	// DEPRECATED: remove and replace with something more accurate
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	flags.StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	flagNames := clientcmd.RecommendedConfigOverrideFlags("")
	// short flagnames are disabled by default.  These are here for compatibility with existing scripts
	flagNames.ClusterOverrideFlags.APIServer.ShortName = "s"

	clientcmd.BindOverrideFlags(overrides, flags, flagNames)
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, overrides, os.Stdin)

	return clientConfig
}

// this function is defined as a closure so iut could be overwritten by unit tests
var GetKubevirtClientFromClientConfig = func(cmdConfig clientcmd.ClientConfig) (KubevirtClient, error) {
	config, err := cmdConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return GetKubevirtClientFromRESTConfig(config)

}

func GetKubevirtClientFromRESTConfig(config *rest.Config) (KubevirtClient, error) {
	shallowCopy := *config
	shallowCopy.GroupVersion = &v1.StorageGroupVersion
	shallowCopy.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: Codecs}
	shallowCopy.APIPath = "/apis"
	shallowCopy.ContentType = runtime.ContentTypeJSON
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}

	executeRestConfigHooks(&shallowCopy)

	restClient, err := rest.RESTClientFor(&shallowCopy)
	if err != nil {
		return nil, err
	}

	coreClient, err := kubernetes.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	generatedKubeVirtClient, err := generatedclient.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	cdiClient, err := cdiclient.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	networkClient, err := networkclient.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	extensionsClient, err := extclient.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	secClient, err := secv1.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	routeClient, err := routev1.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	prometheusClient, err := promclient.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	snapshotClient, err := k8ssnapshotclient.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	migrationsClient, err := migrationsv1.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	cloneClient, err := clonev1alpha1.NewForConfig(&shallowCopy)
	if err != nil {
		return nil, err
	}

	return &kubevirt{
		master,
		kubeconfig,
		restClient,
		&shallowCopy,
		generatedKubeVirtClient,
		cdiClient,
		networkClient,
		extensionsClient,
		secClient,
		routeClient,
		discoveryClient,
		prometheusClient,
		snapshotClient,
		dynamicClient,
		migrationsClient,
		cloneClient,
		coreClient,
	}, nil
}

func GetKubevirtClientFromFlags(master string, kubeconfig string) (KubevirtClient, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}
	return GetKubevirtClientFromRESTConfig(config)
}

func GetKubevirtClient() (KubevirtClient, error) {
	var err error
	once.Do(func() {
		virtclient, err = GetKubevirtClientFromFlags(master, kubeconfig)
	})
	return virtclient, err
}

func GetKubevirtSubresourceClient() (KubevirtClient, error) {
	return GetKubevirtSubresourceClientFromFlags(master, kubeconfig)
}

// Deprecated: Use GetKubevirtClientConfig instead
func GetConfig() (*restclient.Config, error) {
	return clientcmd.BuildConfigFromFlags(master, kubeconfig)
}

func GetKubevirtClientConfig() (*rest.Config, error) {
	return GetConfig()
}
