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
	"net/url"
	"strconv"
	"strings"
)

const defaultNone = "none"

type rtWrapper struct {
	origRoundTripper http.RoundTripper
}

func (r *rtWrapper) RoundTrip(request *http.Request) (response *http.Response, err error) {
	var (
		status string
		host   string
	)

	response, err = r.origRoundTripper.RoundTrip(request)
	if err != nil {
		status = "<error>"
	} else {
		status = strconv.Itoa(response.StatusCode)
	}

	resource, verb := parseURLResourceOperation(request)

	if request.URL != nil {
		host = request.URL.Host
	} else {
		host = defaultNone
	}

	requestResult.WithLabelValues(status, request.Method, host, resource, verb).Add(1)

	return response, err
}

func parseURLResourceOperation(request *http.Request) (resource, verb string) {
	method := request.Method
	if request.URL == nil || method == "" {
		return defaultNone, defaultNone
	}

	resource = findResource(*request.URL)
	if resource == "" {
		return defaultNone, defaultNone
	}

	return resource, getVerbFromHTTPVerb(*request.URL, method)
}

func getVerbFromHTTPVerb(u url.URL, methodOrVerb string) (verb string) {
	switch methodOrVerb {
	case "GET":
		verb = determineGetVerb(u)
	case "PUT":
		verb = "UPDATE"
	case "PATCH":
		verb = "PATCH"
	case "POST":
		verb = "CREATE"
	case "DELETE":
		verb = "DELETE"
	default:
		verb = methodOrVerb
	}

	return verb
}

func determineGetVerb(u url.URL) string {
	if strings.Contains(u.Path, "/watch/") || u.Query().Get("watch") == "true" {
		return "WATCH"
	}

	if resource := findResource(u); resource == "" {
		return "none"
	} else if strings.HasSuffix(u.Path, resource) {
		return "LIST"
	}

	return "GET"
}

func findResource(u url.URL) (resource string) {
	for _, r := range resourceParsingRegexs {
		if match := r.FindStringSubmatch(u.Path); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}
