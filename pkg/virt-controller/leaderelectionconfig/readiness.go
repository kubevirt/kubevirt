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

package leaderelectionconfig

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"errors"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

func ControllerLeadElectionReadinessFunc(_ *restful.Request, response *restful.Response) {
	res := map[string]interface{}{}
	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		unhealthy(err, response)
		return
	}

	endpoints, err := cli.CoreV1().Endpoints(DefaultNamespace).Get(DefaultEndpointName, metav1.GetOptions{})
	if err != nil {
		unhealthy(err, response)
		return
	}
	if endpoints.Annotations == nil {
		endpoints.Annotations = make(map[string]string)
	}

	var record resourcelock.LeaderElectionRecord
	if recordBytes, found := endpoints.Annotations[resourcelock.LeaderElectionRecordAnnotationKey]; found {
		if err := json.Unmarshal([]byte(recordBytes), &record); err != nil {
			unhealthy(err, response)
			return
		}
	}

	id, err := os.Hostname()
	if err != nil {
		unhealthy(err, response)
		return
	}

	if id != record.HolderIdentity {
		unhealthy(errors.New("current pod is not leader"), response)
		return
	}

	res["apiserver"] = map[string]interface{}{"connectivity": "ok", "leader": record.HolderIdentity}
	response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
	return
}

func unhealthy(err error, response *restful.Response) {
	res := map[string]interface{}{}
	res["apiserver"] = map[string]interface{}{"connectivity": "failed", "error": fmt.Sprintf("%v", err)}
	response.WriteHeaderAndJson(http.StatusInternalServerError, res, restful.MIME_JSON)
}
