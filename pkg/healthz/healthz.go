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

	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/util/json"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/client-go/kubecli"
)

func KubeConnectionHealthzFuncFactory(clusterConfig *virtconfig.ClusterConfig) func(_ *restful.Request, response *restful.Response) {
	return func(_ *restful.Request, response *restful.Response) {
		res := map[string]interface{}{}
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
		var version interface{}
		err = json.Unmarshal(body, &version)
		if err != nil {
			unhealthy(err, clusterConfig, response)
			return
		}
		res["apiserver"] = map[string]interface{}{"connectivity": "ok", "version": version}
		res["config-resource-version"] = clusterConfig.GetResourceVersion()
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
