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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/klog/v2"
)

const getTimeout = 30 * time.Second

func getParentResourceNames(resourcesStorage map[string]rest.Storage) []string {
	parents := map[string]struct{}{}
	for resource := range resourcesStorage {
		if strings.Contains(resource, "/") {
			const subresourceParts = 2
			parts := strings.SplitN(resource, "/", subresourceParts)
			parents[parts[0]] = struct{}{}
		}
	}
	return slices.Collect(maps.Keys(parents))
}

// installFilteredAPIVersionHandler replaces the APIVersionHandler for a GroupVersion
// to filter out specified parent resources from the returned APIResourceList.
func installFilteredAPIVersionHandler(
	gv schema.GroupVersion,
	resourcesToHide []string,
	container *restful.Container,
	factory runtime.NegotiatedSerializer,
) error {
	wsPath := path.Join(genericapiserver.APIGroupPrefix, gv.Group, gv.Version)
	var ws *restful.WebService
	for _, registeredWs := range container.RegisteredWebServices() {
		if registeredWs.RootPath() == wsPath {
			ws = registeredWs
			break
		}
	}
	if ws == nil {
		return fmt.Errorf("could not find the APIResource WebService for %s", gv.String())
	}

	routePath := wsPath + "/"
	var originalHandler restful.RouteFunction
	for _, route := range ws.Routes() {
		if route.Method == http.MethodGet && route.Path == routePath {
			originalHandler = route.Function
			break
		}
	}
	if originalHandler == nil {
		return fmt.Errorf("could not find the APIVersionHandler for %s", gv.String())
	}

	filteringLister := &filteringAPIResourceLister{
		groupVersion:    gv.String(),
		originalHandler: originalHandler,
		resourcesToHide: resourcesToHide,
	}

	ws.SetDynamicRoutes(true)
	if err := ws.RemoveRoute(routePath, http.MethodGet); err != nil {
		return err
	}

	handler := discovery.NewAPIVersionHandler(factory, gv, filteringLister)
	handler.AddToWebService(ws)

	return nil
}

type filteringAPIResourceLister struct {
	groupVersion    string
	originalHandler restful.RouteFunction
	resourcesToHide []string
	cached          bool
	cachedList      []metav1.APIResource
}

// ListAPIResources calls the original handler once to get the full list, then filters and caches it
func (f *filteringAPIResourceLister) ListAPIResources() []metav1.APIResource {
	if f.cached {
		return f.cachedList
	}

	resources, err := f.get()
	if err != nil {
		klog.Errorf("Failed to get unfiltered APIResourceList, returning empty list for %s: %v", f.groupVersion, err)
		f.cached = true
		return f.cachedList
	}

	return f.filter(resources)
}

func (f *filteringAPIResourceLister) get() ([]metav1.APIResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
	defer cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	req.Header.Set("Accept", "application/json")
	restReq := restful.NewRequest(req)

	recorder := httptest.NewRecorder()
	restResp := restful.NewResponse(recorder)

	f.originalHandler(restReq, restResp)

	var apiResourceList metav1.APIResourceList
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &apiResourceList); err != nil {
		return nil, err
	}

	return apiResourceList.APIResources, nil
}

func (f *filteringAPIResourceLister) filter(unfiltered []metav1.APIResource) []metav1.APIResource {
	var filtered []metav1.APIResource
	for _, resource := range unfiltered {
		if !slices.Contains(f.resourcesToHide, resource.Name) {
			filtered = append(filtered, resource)
		}
	}
	f.cachedList = filtered
	f.cached = true
	return filtered
}
