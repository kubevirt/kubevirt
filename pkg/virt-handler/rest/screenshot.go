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
package rest

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubevirt.io/client-go/log"
)

func (lh *LifecycleHandler) ScreenshotRequestHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}
	defer client.Close()

	log.Log.Object(vmi).Infof("Requesting screenshot")
	screenshotResponse, err := client.GetScreenshot(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get Screenshot")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.AddHeader("Content-Type", screenshotResponse.Mime)
	if nbytes, err := response.Write(screenshotResponse.Data); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to write response")
		response.WriteError(http.StatusInternalServerError, err)
	} else if nbytes != len(screenshotResponse.Data) {
		err = fmt.Errorf("Failed to write full response: %d of %d written", nbytes, len(screenshotResponse.Data))
		log.Log.Object(vmi).Reason(err).Error("Incomplete message written")
		response.WriteError(http.StatusInternalServerError, err)
	}
}
