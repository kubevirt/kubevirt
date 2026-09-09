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

package apiserver

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/util/compatibility"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachine"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver/storage/virtualmachineinstance"
)

type connectRequestBodyDescriber interface {
	ConnectRequestBody(method string) interface{}
}

// BuildSubresourceOpenAPISpec returns the OpenAPI v2 spec of the
// subresources.kubevirt.io API group as virt-api actually serves it.
//
// It installs the same storage maps and the same ClusterLevelRoutes into an
// offline GenericAPIServer and returns the spec its PrepareRun produces, so
// tools/openapispec documents the routes of the real API server instead of a
// second route table maintained by hand. Handlers are never invoked which is
// why the storages can be built with nil clients.
func BuildSubresourceOpenAPISpec() (*spec.Swagger, error) {
	scheme := NewScheme()
	factory := serializer.NewCodecFactory(scheme)

	config := genericapiserver.NewRecommendedConfig(factory)
	config.EffectiveVersion = compatibility.DefaultBuildEffectiveVersion()
	config.OpenAPIConfig = NewOpenAPIConfig(scheme)
	config.OpenAPIV3Config = NewOpenAPIV3Config(scheme)
	config.ExternalAddress = "127.0.0.1:8443"
	config.LoopbackClientConfig = &restclient.Config{Host: "https://127.0.0.1:8443"}
	config.Authorization.Authorizer = authorizerfactory.NewAlwaysAllowAuthorizer()

	apiGroups := APIGroups{}
	for _, gv := range v1.SubresourceGroupVersions {
		storage := virtualmachine.NewStorageMap(nil, nil, 0, nil, nil)
		for resource, store := range virtualmachineinstance.NewStorageMap(nil, nil, 0, nil, nil) {
			storage[resource] = store
		}
		apiGroups[gv] = storage
	}

	server, err := config.Complete().New("virt-api-openapispec", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, fmt.Errorf("create offline GenericAPIServer: %w", err)
	}
	for _, groupInfo := range buildAPIGroupInfos(apiGroups, scheme, factory) {
		gi := groupInfo
		if err := server.InstallAPIGroup(&gi); err != nil {
			return nil, fmt.Errorf("install API group %q: %w", gi.PrioritizedVersions[0].Group, err)
		}
	}

	if err := installClusterLevelOpenAPIRoutes(server.Handler.GoRestfulContainer, v1.SubresourceGroupVersions); err != nil {
		return nil, err
	}
	installRootLevelOpenAPIRoutes(server.Handler.GoRestfulContainer, v1.SubresourceGroupVersions)

	prepared := server.PrepareRun()
	if prepared.StaticOpenAPISpec == nil {
		return nil, fmt.Errorf("PrepareRun produced no OpenAPI spec")
	}

	if err := documentConnectRequestBodies(prepared.StaticOpenAPISpec, apiGroups); err != nil {
		return nil, err
	}
	documentSubresourceOperations(prepared.StaticOpenAPISpec)
	return prepared.StaticOpenAPISpec, nil
}

// documentConnectRequestBodies restores the request body schemas that the
// aggregated server drops for rest.Connecter subresources. For every storage
// that implements connectRequestBodyDescriber it adds a body parameter to the
// matching operation and pulls the referenced definition into the spec.
func documentConnectRequestBodies(swagger *spec.Swagger, apiGroups APIGroups) error {
	if swagger.Paths == nil {
		return nil
	}

	// getDefinitions keys schemas by their Go import path; reuse the same ref
	// naming the rest of the spec uses so injected $refs line up.
	namer := func(path string) spec.Ref {
		name, _ := getDefinitionName(path)
		return spec.MustCreateRef("#/definitions/" + name)
	}
	allDefs := getDefinitions(namer)
	byDefinitionName := map[string]string{}
	for goPath := range allDefs {
		name, _ := getDefinitionName(goPath)
		byDefinitionName[name] = goPath
	}

	for gv, storages := range apiGroups {
		for resource, store := range storages {
			describer, ok := store.(connectRequestBodyDescriber)
			if !ok {
				continue
			}
			connecter, ok := store.(rest.Connecter)
			if !ok {
				continue
			}
			for _, method := range connecter.ConnectMethods() {
				body := describer.ConnectRequestBody(method)
				if body == nil {
					continue
				}
				goPath := goDefinitionPath(body)
				defName, _ := getDefinitionName(goPath)
				if err := addDefinition(swagger, allDefs, byDefinitionName, defName); err != nil {
					return err
				}
				for _, path := range connectPaths(gv, resource, swagger) {
					injectBodyParameter(swagger.Paths.Paths[path], method, defName)
				}
			}
		}
	}
	return nil
}

// connectPaths returns the OpenAPI paths that carry a subresource's connect
// operation: the resource path itself and, for connecters that take a trailing
// {path} its wildcard child.
func connectPaths(gv schema.GroupVersion, resource string, swagger *spec.Swagger) []string {
	parts := strings.SplitN(resource, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	base := fmt.Sprintf("%s/namespaces/{namespace}/%s/{name}/%s", GroupVersionRoot(gv), parts[0], parts[1])

	var paths []string
	for _, candidate := range []string{base, base + "/{path}"} {
		if _, ok := swagger.Paths.Paths[candidate]; ok {
			paths = append(paths, candidate)
		}
	}
	return paths
}

func injectBodyParameter(item spec.PathItem, method, defName string) {
	var op *spec.Operation
	switch method {
	case http.MethodGet:
		op = item.Get
	case http.MethodPut:
		op = item.Put
	case http.MethodPost:
		op = item.Post
	}
	if op == nil {
		return
	}
	for _, param := range op.Parameters {
		if param.In == "body" {
			return
		}
	}
	op.Parameters = append(op.Parameters, spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:     "body",
			In:       "body",
			Required: true,
			Schema:   &spec.Schema{SchemaProps: spec.SchemaProps{Ref: spec.MustCreateRef("#/definitions/" + defName)}},
		},
	})
}

// addDefinition copies a definition and everything it references from the full
// KubeVirt definition set into the spec so the injected body $refs resolve.
func addDefinition(swagger *spec.Swagger, allDefs map[string]common.OpenAPIDefinition, byDefinitionName map[string]string, defName string) error {
	if swagger.Definitions == nil {
		swagger.Definitions = spec.Definitions{}
	}
	if _, ok := swagger.Definitions[defName]; ok {
		return nil
	}
	goPath, ok := byDefinitionName[defName]
	if !ok {
		return fmt.Errorf("no OpenAPI definition found for %q", defName)
	}
	definition := allDefs[goPath]
	swagger.Definitions[defName] = definition.Schema
	for _, referenced := range referencedDefinitionNames(definition.Schema) {
		if err := addDefinition(swagger, allDefs, byDefinitionName, referenced); err != nil {
			return err
		}
	}
	return nil
}

// referencedDefinitionNames walks a schema and returns the definition names it
// points at via "#/definitions/<name>" references.
func referencedDefinitionNames(s spec.Schema) []string {
	var names []string
	var walk func(spec.Schema)
	walk = func(sc spec.Schema) {
		if ref := sc.Ref.String(); strings.HasPrefix(ref, "#/definitions/") {
			names = append(names, strings.TrimPrefix(ref, "#/definitions/"))
		}
		for _, child := range sc.Properties {
			walk(child)
		}
		for _, child := range sc.AllOf {
			walk(child)
		}
		if sc.Items != nil {
			if sc.Items.Schema != nil {
				walk(*sc.Items.Schema)
			}
			for _, child := range sc.Items.Schemas {
				walk(child)
			}
		}
		if sc.AdditionalProperties != nil && sc.AdditionalProperties.Schema != nil {
			walk(*sc.AdditionalProperties.Schema)
		}
	}
	walk(s)
	return names
}

// goDefinitionPath returns the Go import path plus type name used to key an
// object in the generated OpenAPI definition set, e.g.
// kubevirt.io/api/core/v1.StartOptions.
func goDefinitionPath(obj interface{}) string {
	t := reflect.TypeOf(obj)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// installClusterLevelOpenAPIRoutes documents the ClusterLevelRoutes on the
// go-restful container. At runtime these are served by ConditionalAPIHandlers
// from the handler chain, which the OpenAPI builder cannot see, so they are
// replayed here as routes with no-op handlers.
func installClusterLevelOpenAPIRoutes(container *restful.Container, groupVersions []schema.GroupVersion) error {
	noop := func(*restful.Request, *restful.Response) {}

	// Reuse the WebServices InstallAPIGroup created: go-restful silently drops a
	// second WebService registered on an already served root path.
	byRoot := map[string]*restful.WebService{}
	for _, ws := range container.RegisteredWebServices() {
		byRoot[strings.TrimSuffix(ws.RootPath(), "/")] = ws
	}

	for _, gv := range groupVersions {
		root := GroupVersionRoot(gv)
		ws := byRoot[root]
		if ws == nil {
			ws = new(restful.WebService)
			ws.Path(root)
			container.Add(ws)
			byRoot[root] = ws
		}

		for _, route := range ClusterLevelRoutes() {
			operation, err := clusterLevelOperationID(route)
			if err != nil {
				return err
			}

			builder := ws.Method(route.Method).Path(route.SubPath()).
				To(noop).
				Operation(operation).
				Doc(route.Doc).
				Produces(restful.MIME_JSON).
				Consumes(restful.MIME_JSON)
			if route.Namespaced {
				builder = builder.Param(ws.PathParameter("namespace", "Object name and auth scope, such as for teams and projects").
					Required(true).DataType("string"))
			}
			ws.Route(builder)
		}
	}
	return nil
}

// installRootLevelOpenAPIRoutes documents the endpoints virt-api serves from the
// NonGoRestfulMux. The mux is invisible to the OpenAPI builder, so they are
// replayed as go-restful routes derived from the paths virt-api registers.
//
// Besides KubeVirt's own healthz and profiler endpoints this covers the index
// and the OpenAPI endpoints that GenericAPIServer installs during PrepareRun.
func installRootLevelOpenAPIRoutes(container *restful.Container, groupVersions []schema.GroupVersion) {
	noop := func(*restful.Request, *restful.Response) {}

	paths := []string{HealthzPath}
	paths = append(paths, ComponentProfilerPaths()...)
	paths = append(paths, "/openapi/v2", "/openapi/v3")
	for _, gv := range groupVersions {
		paths = append(paths, "/openapi/v3"+GroupVersionRoot(gv))
	}

	ws := new(restful.WebService)
	ws.Path("/")
	for _, path := range paths {
		ws.Route(ws.GET(path).
			To(noop).
			Operation("read" + rootLevelOperationName(path)).
			Doc("Endpoint served by virt-api outside of the aggregated API surface.").
			Produces(restful.MIME_JSON))
	}
	container.Add(ws)
}

// rootLevelOperationName turns a mux path into the unique part of its operation
// ID. Operation IDs cannot carry slashes or dashes, and "/" needs a name of its
// own because trimming leaves nothing behind.
func rootLevelOperationName(path string) string {
	if path == "/" {
		return "Index"
	}
	var name strings.Builder
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		name.WriteString(camelCase(strings.ReplaceAll(segment, ".", "-")))
	}
	return name.String()
}

// camelCase turns a dash separated path segment into an upper camel case name
// usable inside an OpenAPI operation ID.
func camelCase(s string) string {
	var b strings.Builder
	for _, word := range strings.Split(s, "-") {
		if word == "" {
			continue
		}
		b.WriteString(strings.ToUpper(word[:1]))
		b.WriteString(word[1:])
	}
	return b.String()
}

// clusterLevelOperationID builds an operation name for a ClusterLevelRoute.
// GetOperationIDAndTags rejects names that do not start with a known verb, and
// expands the rest with the group and version from the path, so this only needs
// to be unique within a group version.
func clusterLevelOperationID(route ClusterLevelRoute) (string, error) {
	var verb string
	switch route.Method {
	case http.MethodGet:
		verb = "read"
	case http.MethodPut:
		verb = "replace"
	default:
		return "", fmt.Errorf("cluster level route %q uses unmapped method %q", route.Resource, route.Method)
	}
	return verb + strings.ReplaceAll(route.Resource, "-", ""), nil
}
