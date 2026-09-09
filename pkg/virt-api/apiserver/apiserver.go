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
 * Copyright The KubeVirt Authors.
 *
 */

// Package apiserver hosts the aggregated API server scaffolding used by virt-api
// to serve KubeVirt's subresources.kubevirt.io API group on top of
// k8s.io/apiserver. The structure mirrors the kubevirt/virt-template project so
// the same "skeleton once, fill storages one by one" pattern can be reused.
package apiserver

import (
	"context"
	"flag"
	"net/http"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/compatibility"
	"k8s.io/klog/v2"
	openapicommon "k8s.io/kube-openapi/pkg/common"
)

type (
	APIGroups = map[schema.GroupVersion]map[string]rest.Storage

	// a plain mux handler that is served from the secured handler chain for the requests
	// its Matches predicate selects. It is meant for endpoints that cannot be
	// expressed with a rest.Storage interface, e.g. expand-vm-spec which does a
	// PUT to the collection path without a name.
	ConditionalAPIHandler struct {
		Matches func(*request.RequestInfo) bool
		Handler http.Handler
	}

	// MuxHandler registered http.Handler on the GenericAPIServer's
	// NonGoRestfulMux under a fixed path. It is meant for endpoints that are
	// not part of the aggregated API surface and are not rest.Storage.
	MuxHandler struct {
		Path    string
		Handler http.Handler
	}

	APIServer struct {
		secureServingOpts *options.SecureServingOptionsWithLoopback
		authnOpts         *options.DelegatingAuthenticationOptions
		authzOpts         *options.DelegatingAuthorizationOptions

		apiHandlers      []ConditionalAPIHandler
		muxHandlers      []MuxHandler
		alwaysAllowPaths []string

		longRunningSubresources []string
	}
)

func New() *APIServer {
	return &APIServer{
		secureServingOpts: options.NewSecureServingOptions().WithLoopback(),
		authnOpts:         options.NewDelegatingAuthenticationOptions(),
		authzOpts:         options.NewDelegatingAuthorizationOptions(),
	}
}

func (a *APIServer) AddFlags(fs *pflag.FlagSet) {
	a.secureServingOpts.AddFlags(fs)
	a.authnOpts.AddFlags(fs)
	a.authzOpts.AddFlags(fs)

	goFs := flag.NewFlagSet("", flag.ExitOnError)
	klog.InitFlags(goFs)

	// to prevent flags from being overridden by the binary
	goFs.VisitAll(func(gf *flag.Flag) {
		if fs.Lookup(gf.Name) == nil {
			fs.AddGoFlag(gf)
		}
	})
}

func (a *APIServer) WithSecureServingPort(port int) *APIServer {
	a.secureServingOpts.BindPort = port
	return a
}

// BindPort returns the configured secure serving port. It lets the legacy
// virt-api server reuse the same --secure-port flag as the GenericAPIServer.
func (a *APIServer) BindPort() int {
	return a.secureServingOpts.BindPort
}

// CertFile returns the configured serving certificate file path.
func (a *APIServer) CertFile() string {
	return a.secureServingOpts.ServerCert.CertKey.CertFile
}

// KeyFile returns the configured serving private key file path.
func (a *APIServer) KeyFile() string {
	return a.secureServingOpts.ServerCert.CertKey.KeyFile
}

func (a *APIServer) WithSecureServingCertDirectory(dir string) *APIServer {
	a.secureServingOpts.ServerCert.CertDirectory = dir
	return a
}

func (a *APIServer) WithAPIHandlers(handlers ...ConditionalAPIHandler) *APIServer {
	a.apiHandlers = append(a.apiHandlers, handlers...)
	return a
}

// WithMuxHandlers registered plain http.Handlers on the GenericAPIServer's NonGoRestfulMux
func (a *APIServer) WithMuxHandlers(handlers ...MuxHandler) *APIServer {
	a.muxHandlers = append(a.muxHandlers, handlers...)
	return a
}

func (a *APIServer) WithAlwaysAllowPaths(paths ...string) *APIServer {
	a.alwaysAllowPaths = append(a.alwaysAllowPaths, paths...)
	return a
}

// marks the given subresources as long-running so the GenericAPIServer does not
// enforce its default RequestTimeout on them.
func (a *APIServer) WithLongRunningSubresources(subresources ...string) *APIServer {
	a.longRunningSubresources = append(a.longRunningSubresources, subresources...)
	return a
}

func (a *APIServer) WithSecureServingCert(certFile, keyFile string) *APIServer {
	a.secureServingOpts.ServerCert.CertKey.CertFile = certFile
	a.secureServingOpts.ServerCert.CertKey.KeyFile = keyFile
	return a
}

func (a *APIServer) Run(
	ctx context.Context,
	name string,
	scheme *runtime.Scheme,
	openAPIConfig *openapicommon.Config,
	openapiV3Config *openapicommon.OpenAPIV3Config,
	apiGroups APIGroups,
) error {
	factory := serializer.NewCodecFactory(scheme)
	config := genericapiserver.NewRecommendedConfig(factory)
	config.EffectiveVersion = compatibility.DefaultBuildEffectiveVersion()
	config.OpenAPIConfig = openAPIConfig
	config.OpenAPIV3Config = openapiV3Config
	// Disable discovery to not confuse kubectl and other client with dummy resources
	config.EnableDiscovery = false

	config.LongRunningFunc = genericfilters.BasicLongRunningRequestCheck(
		sets.NewString("watch"),
		sets.NewString(a.longRunningSubresources...),
	)

	a.authzOpts.AlwaysAllowPaths = append(a.authzOpts.AlwaysAllowPaths, a.collectAlwaysAllowPaths(apiGroups)...)
	if len(a.apiHandlers) > 0 {
		apiHandlers := a.apiHandlers
		base := config.BuildHandlerChainFunc
		if base == nil {
			base = genericapiserver.DefaultBuildHandlerChain
		}
		config.BuildHandlerChainFunc = func(apiHandler http.Handler, c *genericapiserver.Config) http.Handler {
			dispatcher := apiHandler
			if len(apiHandlers) > 0 {
				next := apiHandler
				dispatcher = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if info, ok := request.RequestInfoFrom(r.Context()); ok {
						for _, ah := range apiHandlers {
							if ah.Matches(info) {
								ah.Handler.ServeHTTP(w, r)
								return
							}
						}
					}
					next.ServeHTTP(w, r)
				})
			}

			secured := base(dispatcher, c)
			return secured
		}
	}

	innerChain := config.BuildHandlerChainFunc
	if innerChain == nil {
		innerChain = genericapiserver.DefaultBuildHandlerChain
	}
	config.BuildHandlerChainFunc = func(apiHandler http.Handler, c *genericapiserver.Config) http.Handler {
		next := innerChain(apiHandler, c)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	}

	if err := a.secureServingOpts.ApplyTo(&config.SecureServing, &config.LoopbackClientConfig); err != nil {
		klog.Errorf("Failed to apply secure serving options: %v", err)
		return err
	}
	if err := a.authnOpts.ApplyTo(&config.Authentication, config.SecureServing, config.OpenAPIConfig); err != nil {
		klog.Errorf("Failed to apply authentication options: %v", err)
		return err
	}
	if err := a.authzOpts.ApplyTo(&config.Authorization); err != nil {
		klog.Errorf("Failed to apply authorization options: %v", err)
		return err
	}
	// AlwaysAllowPaths cannot exempt the cluster level routes, because they are
	// resource requests. Union the exemption in front of the delegated authorizer.
	config.Authorization.Authorizer = union.New(
		clusterLevelAlwaysAllowAuthorizer(apiGroups),
		config.Authorization.Authorizer,
	)

	server, err := config.Complete().New(name, genericapiserver.NewEmptyDelegate())
	if err != nil {
		klog.Errorf("Failed to create server: %v", err)
		return err
	}

	// A single API group may expose multiple versions
	// InstallAPIGroup must be called once per group with all
	// its versions merged, so build merged APIGroupInfos first.
	for _, groupInfo := range buildAPIGroupInfos(apiGroups, scheme, factory) {
		gi := groupInfo
		if err := server.InstallAPIGroup(&gi); err != nil {
			klog.Errorf("Failed to install APIGroup: %v", err)
			return err
		}
	}
	for gv, resourcesStorage := range apiGroups {
		resourcesToHide := getParentResourceNames(resourcesStorage)
		if len(resourcesToHide) > 0 {
			klog.Infof("Hiding parent resources from APIResourceList: %v", resourcesToHide)
			if err := installFilteredAPIVersionHandler(gv, resourcesToHide, server.Handler.GoRestfulContainer, factory); err != nil {
				return err
			}
		}
	}

	klog.Info("Starting aggregated API server...")
	preparedServer := server.PrepareRun()
	for _, mh := range a.muxHandlers {
		server.Handler.NonGoRestfulMux.Unregister(mh.Path)
		server.Handler.NonGoRestfulMux.Handle(mh.Path, mh.Handler)
	}

	if err := preparedServer.RunWithContext(ctx); err != nil {
		klog.Errorf("Failed to run server: %v", err)
		return err
	}

	return nil
}

// buildAPIGroupInfos merges all versions of the same API group into a single
// APIGroupInfo, so each group is installed exactly once with every version it
// exposes.
func buildAPIGroupInfos(
	apiGroups APIGroups, scheme *runtime.Scheme, factory serializer.CodecFactory,
) map[string]genericapiserver.APIGroupInfo {
	result := map[string]genericapiserver.APIGroupInfo{}
	for gv, storage := range apiGroups {
		gi, ok := result[gv.Group]
		if !ok {
			gi = genericapiserver.NewDefaultAPIGroupInfo(
				gv.Group, scheme, runtime.NewParameterCodec(scheme), factory,
			)
		}
		gi.VersionedResourcesStorageMap[gv.Version] = storage
		result[gv.Group] = gi
	}
	return result
}

func (a *APIServer) collectAlwaysAllowPaths(apiGroups APIGroups) []string {
	paths := []string{
		"/", genericapiserver.APIGroupPrefix, "/openapi/v2", "/openapi/v3", "/openapi/v3/*",
	}
	paths = append(paths, getAdditionalAlwaysAllowPaths(apiGroups)...)
	for _, mh := range a.muxHandlers {
		paths = append(paths, mh.Path)
	}
	paths = append(paths, a.alwaysAllowPaths...)
	return paths
}

func getAdditionalAlwaysAllowPaths(apiGroups APIGroups) []string {
	var additionalAlwaysAllowPaths []string
	for gv := range apiGroups {
		additionalAlwaysAllowPaths = append(additionalAlwaysAllowPaths,
			genericapiserver.APIGroupPrefix+"/"+gv.Group,
			genericapiserver.APIGroupPrefix+"/"+gv.String(),
		)
	}
	return additionalAlwaysAllowPaths
}
