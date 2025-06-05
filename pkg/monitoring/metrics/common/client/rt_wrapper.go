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
 * Copyright the KubeVirt Authors.
 */

package client

import (
	"net/http"
	"strconv"
	"strings"
)

type rtWrapper struct {
	origRoundTripper http.RoundTripper
}

func (r *rtWrapper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	var status string
	var resource string
	var verb string
	var host string

	response, err = r.origRoundTripper.RoundTrip(request)
	if err != nil {
		status = "<error>"
	} else {
		status = strconv.Itoa(response.StatusCode)
	}

	host = "none"
	if request.URL != nil {
		host = request.URL.Host
	}

	resource, verb = parseURLResourceOperation(request)
	if verb == "" {
		verb = "none"
	}
	if resource == "" {
		resource = "none"
	}
	requestResult.WithLabelValues(status, request.Method, host, resource, verb).Add(1)

	return response, err
}

func parseURLResourceOperation(request *http.Request) (resource string, verb string) {
	method := request.Method

	resource = ""
	verb = ""

	if request.URL == nil || request.URL.Path == "" || method == "" {
		return
	}

	for _, r := range resourceParsingRegexs {
		if resource != "" {
			break
		}
		match := r.FindStringSubmatch(request.URL.Path)
		if len(match) > 1 {
			resource = match[1]
		}
	}

	if resource == "" {
		return
	}

	switch method {
	case "GET":
		verb = "GET"
		if strings.Contains(request.URL.Path, "/watch/") {
			verb = "WATCH"
		} else if strings.HasSuffix(request.URL.Path, resource) {
			// If the resource is the last element in the url, then
			// we're asking to list all resources of that type instead
			// of getting an individual resource
			verb = "LIST"
		}
	case "PUT":
		verb = "UPDATE"
	case "PATCH":
		verb = "PATCH"
	case "POST":
		verb = "CREATE"
	case "DELETE":
		verb = "DELETE"
	}

	return resource, verb
}
