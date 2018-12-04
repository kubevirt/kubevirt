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

package filter

import (
	"strings"

	restful "github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/log"
)

func RequestLoggingFilter() restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		var username = "-"
		if req.Request.URL.User != nil {
			if name := req.Request.URL.User.Username(); name != "" {
				username = name
			}
		}
		chain.ProcessFilter(req, resp)
		log.Log.Level(log.INFO).
			With("remoteAddress", strings.Split(req.Request.RemoteAddr, ":")[0]).
			With("username", username).
			With("method", req.Request.Method).
			With("url", req.Request.URL.RequestURI()).
			With("proto", req.Request.Proto).
			With("statusCode", resp.StatusCode()).
			Log("contentLength", resp.ContentLength())
	}
}
