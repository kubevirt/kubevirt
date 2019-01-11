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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE -imports restful=github.com/emicklei/go-restful

import (
	"fmt"
	"net/http"
	"strings"

	restful "github.com/emicklei/go-restful"
	authorization "k8s.io/api/authorization/v1beta1"
	authorizationclient "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	restclient "k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/kubecli"
)

const (
	userHeader            = "X-Remote-User"
	groupHeader           = "X-Remote-Group"
	userExtraHeaderPrefix = "X-Remote-Extra-"
	clientQPS             = 200
	clientBurst           = 400
)

type VirtApiAuthorizor interface {
	Authorize(req *restful.Request) (bool, string, error)
	AddUserHeaders(header []string)
	GetUserHeaders() []string
	AddGroupHeaders(header []string)
	GetGroupHeaders() []string
	AddExtraPrefixHeaders(header []string)
	GetExtraPrefixHeaders() []string
}

type authorizor struct {
	userHeaders             []string
	groupHeaders            []string
	userExtraHeaderPrefixes []string

	subjectAccessReview authorizationclient.SubjectAccessReviewInterface
}

func (a *authorizor) getUserGroups(header http.Header) ([]string, error) {

	for _, key := range a.groupHeaders {
		groups, ok := header[key]
		if ok {
			return groups, nil
		}
	}

	return nil, fmt.Errorf("a valid group header is required for authorization")
}

func (a *authorizor) getUserName(header http.Header) (string, error) {
	for _, key := range a.userHeaders {
		user, ok := header[key]
		if ok {
			return user[0], nil
		}
	}

	return "", fmt.Errorf("a valid user header is required for authorization")
}

func (a *authorizor) getUserExtras(header http.Header) map[string]authorization.ExtraValue {

	extras := map[string]authorization.ExtraValue{}

	for _, prefix := range a.userExtraHeaderPrefixes {
		for k, v := range header {
			if strings.HasPrefix(k, prefix) {
				extraKey := strings.TrimPrefix(k, prefix)
				extras[extraKey] = v
			}
		}
	}

	return extras
}

func (a *authorizor) AddUserHeaders(headers []string) {
	a.userHeaders = append(a.userHeaders, headers...)
}

func (a *authorizor) GetUserHeaders() []string {
	return a.userHeaders
}

func (a *authorizor) AddGroupHeaders(headers []string) {
	a.groupHeaders = append(a.groupHeaders, headers...)
}

func (a *authorizor) GetGroupHeaders() []string {
	return a.groupHeaders
}

func (a *authorizor) AddExtraPrefixHeaders(headers []string) {
	a.userExtraHeaderPrefixes = append(a.userExtraHeaderPrefixes, headers...)
}

func (a *authorizor) GetExtraPrefixHeaders() []string {
	return a.userExtraHeaderPrefixes
}

func (a *authorizor) generateAccessReview(req *restful.Request) (*authorization.SubjectAccessReview, error) {

	httpRequest := req.Request

	if httpRequest == nil {
		return nil, fmt.Errorf("empty http request")
	}
	headers := httpRequest.Header
	url := httpRequest.URL

	if url == nil {
		return nil, fmt.Errorf("no URL in http request")
	}

	// URL example
	// /apis/subresources.kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/testvmi/console
	pathSplit := strings.Split(url.Path, "/")
	if len(pathSplit) != 9 {
		return nil, fmt.Errorf("unknown api endpoint %s", url.Path)
	}

	group := pathSplit[2]
	version := pathSplit[3]
	namespace := pathSplit[5]
	resource := pathSplit[6]
	resourceName := pathSplit[7]
	subresource := pathSplit[8]
	userExtras := a.getUserExtras(headers)

	if resource != "virtualmachineinstances" && resource != "virtualmachines" {
		return nil, fmt.Errorf("unknown resource type %s", resource)
	}

	userName, err := a.getUserName(headers)
	if err != nil {
		return nil, err
	}

	userGroups, err := a.getUserGroups(headers)
	if err != nil {
		return nil, err
	}
	verb := strings.ToLower(httpRequest.Method)

	r := &authorization.SubjectAccessReview{}
	r.Spec = authorization.SubjectAccessReviewSpec{
		User:   userName,
		Groups: userGroups,
		Extra:  userExtras,
	}

	r.Spec.ResourceAttributes = &authorization.ResourceAttributes{
		Namespace:   namespace,
		Verb:        verb,
		Group:       group,
		Version:     version,
		Resource:    resource,
		Subresource: subresource,
		Name:        resourceName,
	}

	return r, nil
}

func isInfoOrHealthEndpoint(req *restful.Request) bool {

	httpRequest := req.Request
	if httpRequest == nil || httpRequest.URL == nil {
		return false
	}
	// URL example
	// /apis/subresources.kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/testvmi/console
	// The /apis/<group>/<version> part of the urls should be accessible without needing authorization
	pathSplit := strings.Split(httpRequest.URL.Path, "/")
	if len(pathSplit) <= 4 || (len(pathSplit) > 4 && (pathSplit[4] == "version" || pathSplit[4] == "healthz")) {
		return true
	}

	return false
}

func isAuthenticated(req *restful.Request) bool {
	// Peer cert is required for authentication.
	// If the peer's cert is provided, we are guaranteed
	// it has been validated against our client CA pool
	if req.Request == nil || req.Request.TLS == nil || len(req.Request.TLS.PeerCertificates) == 0 {
		return false
	}
	return true
}

func (a *authorizor) Authorize(req *restful.Request) (bool, string, error) {

	// Endpoints related to getting information about
	// what apis our server provides are authorized to
	// all users.
	if isInfoOrHealthEndpoint(req) {
		return true, "", nil
	}

	if !isAuthenticated(req) {
		return false, "request is not authenticated", nil
	}

	r, err := a.generateAccessReview(req)
	if err != nil {
		// only internal service errors are returned
		// as an error.
		// A failure to generate the access review request
		// means the client did not properly format the request.
		// Return that error as the "Reason" for the authorization failure.
		return false, fmt.Sprintf("%v", err), nil
	}

	result, err := a.subjectAccessReview.Create(r)
	if err != nil {
		return false, "internal server error", err
	}

	if result.Status.Allowed {
		return true, "", nil
	}

	return false, result.Status.Reason, nil
}

func NewAuthorizorFromConfig(config *restclient.Config) (VirtApiAuthorizor, error) {
	client, err := authorizationclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	subjectAccessReview := client.SubjectAccessReviews()

	a := &authorizor{
		subjectAccessReview: subjectAccessReview,
	}

	// add default headers
	a.userHeaders = append(a.userHeaders, userHeader)
	a.groupHeaders = append(a.groupHeaders, groupHeader)
	a.userExtraHeaderPrefixes = append(a.userExtraHeaderPrefixes, userExtraHeaderPrefix)

	return a, nil
}

func NewAuthorizor() (VirtApiAuthorizor, error) {
	config, err := kubecli.GetConfig()
	if err != nil {
		return nil, err
	}
	config.QPS = clientQPS
	config.Burst = clientBurst

	return NewAuthorizorFromConfig(config)
}
