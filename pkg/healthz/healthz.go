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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package healthz

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	restful "github.com/emicklei/go-restful/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	virtApiComponentName        = "virtApi"
	virtHandlerComponentName    = "virtHandler"
	virtControllerComponentName = "virtController"
)

type KubeApiHealthzVersion struct {
	version interface{}
	sync.RWMutex
}

func (h *KubeApiHealthzVersion) Update(body interface{}) {
	h.Lock()
	defer h.Unlock()
	h.version = body
}

func (h *KubeApiHealthzVersion) Clear() {
	h.Lock()
	defer h.Unlock()
	h.version = nil
}

func (h *KubeApiHealthzVersion) GetVersion() (v interface{}) {
	h.RLock()
	defer h.RUnlock()
	v = h.version
	return
}

/*
   This check is primarily to determine whether a controller can reach the Kubernetes API.
   We can reflect this based on other connections we depend on (informers and their error handling),
   rather than testing the kubernetes API every time the healthcheck endpoint is called. This
   should avoid a lot of unnecessary calls to the API while informers are healthy.

   Note that It is possible for the contents of a KubeApiHealthzVersion to be out of date if the
   Kubernetes API version changes without an informer disconnect, or if informer doesn't call
   KubeApiHealthzVersion.Clear() when it encounters an error.
*/

func KubeConnectionHealthzFuncFactory(clusterConfig *virtconfig.ClusterConfig, hVersion *KubeApiHealthzVersion, component string) func(_ *restful.Request, response *restful.Response) {
	return func(_ *restful.Request, response *restful.Response) {
		res := map[string]interface{}{}
		var version = hVersion.GetVersion()

		if version == nil {
			cli, err := kubecli.GetKubevirtClient()
			if err != nil {
				unhealthy(err, clusterConfig, response)
				return
			}

			body, err := cli.CoreV1().RESTClient().Get().AbsPath("/version").Do(context.Background()).Raw()
			if err != nil {
				unhealthy(err, clusterConfig, response)
				return
			}

			err = json.Unmarshal(body, &version)
			if err != nil {
				unhealthy(err, clusterConfig, response)
				return
			}

			hVersion.Update(version)
		}

		res["apiserver"] = map[string]interface{}{"connectivity": "ok", "version": version}
		resourceVersion := clusterConfig.GetResourceVersion()
		res["config-resource-version"] = resourceVersion
		reportComponentVersion(clusterConfig, component, resourceVersion)
		response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
		return
	}
}

func unhealthy(err error, clusterConfig *virtconfig.ClusterConfig, response *restful.Response) {
	res := map[string]interface{}{}
	res["apiserver"] = map[string]interface{}{"connectivity": "failed", "error": fmt.Sprintf("%v", err)}
	res["config-resource-version"] = clusterConfig.GetResourceVersion()
	response.WriteHeaderAndJson(http.StatusInternalServerError, res, restful.MIME_JSON)
}

func reportComponentVersion(clusterConfig *virtconfig.ClusterConfig, componentName string, resourceVersion string) {
	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to get KubeVirt client")
		return
	}

	kv := clusterConfig.GetConfigFromKubeVirtCR()
	if kv == nil {
		return
	}

	podSuffix := getPodSuffixFromHostname()

	shouldPatch, patchPath := prepareComponentVersionsPatch(kv, componentName, podSuffix, resourceVersion)
	if !shouldPatch {
		return
	}

	patchPayload, err := patch.New(patch.WithReplace(patchPath, resourceVersion)).GeneratePayload()

	_, err = cli.KubeVirt(kv.Namespace).PatchStatus(context.Background(), kv.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to patch KubeVirt CR status")
		return
	}
}

func prepareComponentVersionsPatch(kv *v1.KubeVirt, componentName, podSuffix, resourceVersion string) (bool, string) {
	componentVersions := kv.Status.ComponentVersions

	var componentMap map[string]string

	switch componentName {
	case virtControllerComponentName:
		componentMap = componentVersions.VirtController
	case virtHandlerComponentName:
		componentMap = componentVersions.VirtHandler
	case virtApiComponentName:
		componentMap = componentVersions.VirtApi
	default:
		return false, ""
	}

	currentVersion, exists := componentMap[podSuffix]
	if !exists || currentVersion < resourceVersion {
		patchPath := fmt.Sprintf("/status/componentVersions/%s/%s", componentName, podSuffix)
		return true, patchPath
	}

	// No patch needed if the version is already up-to-date
	return false, ""
}

func getPodSuffixFromHostname() string {
	hostname := os.Getenv("HOSTNAME")
	podParts := strings.Split(hostname, "-")
	return podParts[len(podParts)-1]
}
