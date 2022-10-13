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
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authclientv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/util/flowcontrol"

	"kubevirt.io/client-go/kubecli"
)

const (
	userHeader            = "X-Remote-User"
	groupHeader           = "X-Remote-Group"
	userExtraHeaderPrefix = "X-Remote-Extra-"
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
	client                  authclientv1.SubjectAccessReviewInterface
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

func (a *authorizor) getUserExtras(header http.Header) map[string]authv1.ExtraValue {
	extras := map[string]authv1.ExtraValue{}

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

func (a *authorizor) generateAccessReview(req *restful.Request) (*authv1.SubjectAccessReview, error) {
	if req.Request == nil {
		return nil, fmt.Errorf("empty http request")
	}
	if req.Request.URL == nil {
		return nil, fmt.Errorf("no URL in http request")
	}

	userName, err := a.getUserName(req.Request.Header)
	if err != nil {
		return nil, err
	}

	userGroups, err := a.getUserGroups(req.Request.Header)
	if err != nil {
		return nil, err
	}

	r := &authv1.SubjectAccessReview{}
	r.Spec = authv1.SubjectAccessReviewSpec{
		User:   userName,
		Groups: userGroups,
		Extra:  a.getUserExtras(req.Request.Header),
	}

	// URL example
	// /apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi/console
	pathSplit := strings.Split(req.Request.URL.Path, "/")
	if len(pathSplit) < 5 {
		return nil, fmt.Errorf("unknown api endpoint: %s", req.Request.URL.Path)
	}

	// "namespaces" after version means that the URL points to a namespaced subresource
	if pathSplit[4] == "namespaces" {
		if err := addNamespacedAttributes(req, r, pathSplit); err != nil {
			return nil, err
		}
	} else {
		if err := addNonNamespacedAttributes(req, r, pathSplit); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func addNamespacedAttributes(req *restful.Request, r *authv1.SubjectAccessReview, pathSplit []string) error {
	// URL example
	// /apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi/console
	if len(pathSplit) < 9 {
		return fmt.Errorf("unknown api endpoint %s", req.Request.URL.Path)
	}

	resource := pathSplit[6]
	if resource != "virtualmachineinstances" && resource != "virtualmachines" {
		return fmt.Errorf("unknown resource type %s", resource)
	}

	resourceName := pathSplit[7]
	verb, err := mapHttpVerbToRbacVerb(req.Request.Method, resourceName)
	if err != nil {
		return err
	}

	r.Spec.ResourceAttributes = &authv1.ResourceAttributes{
		Namespace:   pathSplit[5],
		Verb:        verb,
		Group:       pathSplit[2],
		Version:     pathSplit[3],
		Resource:    resource,
		Subresource: pathSplit[8],
		Name:        resourceName,
	}

	return nil
}

func addNonNamespacedAttributes(req *restful.Request, r *authv1.SubjectAccessReview, pathSplit []string) error {
	// URL example
	// /apis/subresources.kubevirt.io/v1alpha3/expand-spec
	if len(pathSplit) != 5 {
		return fmt.Errorf("unknown api endpoint %s", req.Request.URL.Path)
	}

	resource := pathSplit[4]
	if resource != "expand-spec" {
		return fmt.Errorf("unknown resource type %s", resource)
	}

	verb, err := mapHttpVerbToRbacVerb(req.Request.Method, "")
	if err != nil {
		return err
	}

	// Even though there is no CRD for this endpoint, it is still considered a resource request by Kubernetes.
	// Kubernetes only considers requests to endpoints other than /api/v1/... or /apis/<group>/<version>/...  as
	// non-resource requests.
	// See: https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb
	r.Spec.ResourceAttributes = &authv1.ResourceAttributes{
		Verb:     verb,
		Group:    pathSplit[2],
		Version:  pathSplit[3],
		Resource: resource,
	}

	return nil
}

func mapHttpVerbToRbacVerb(httpVerb string, name string) (string, error) {
	// see https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb
	// if name is empty, we assume plural verbs
	switch strings.ToLower(httpVerb) {
	case strings.ToLower(http.MethodPost):
		return "create", nil
	case strings.ToLower(http.MethodGet), strings.ToLower(http.MethodHead):
		if name != "" {
			return "get", nil
		} else {
			return "list", nil
		}
	case strings.ToLower(http.MethodPut):
		return "update", nil
	case strings.ToLower(http.MethodPatch):
		return "patch", nil
	case strings.ToLower(http.MethodDelete):
		if name != "" {
			return "delete", nil
		} else {
			return "deletecollection", nil
		}
	default:
		return "", fmt.Errorf("unknown http verb in request: %v", httpVerb)
	}
}

func isInfoOrHealthEndpoint(req *restful.Request) bool {
	if req.Request == nil || req.Request.URL == nil {
		return false
	}

	// URL example
	// /apis/subresources.kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances/testvmi/console
	// The /apis/<group>/<version> part of the urls should be accessible without needing authorization
	pathSplit := strings.Split(req.Request.URL.Path, "/")
	if len(pathSplit) < 5 {
		return true
	}

	noAuthEndpoints := []string{
		"version",
		"healthz",
		"guestfs",
		// the profiler endpoints are blocked by a feature gate
		// to restrict the usage to development environments
		"start-cluster-profiler",
		"stop-cluster-profiler",
		"dump-cluster-profiler",
	}
	for _, endpoint := range noAuthEndpoints {
		if pathSplit[4] == endpoint {
			return true
		}
	}

	return false
}

func isAuthenticated(req *restful.Request) bool {
	// Peer cert is required for authentication.
	// If the peer's cert is provided, we are guaranteed
	// it has been validated against our CA pool containing the requestheader CA
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

	result, err := a.client.Create(context.Background(), r, metav1.CreateOptions{})
	if err != nil {
		return false, "internal server error", err
	}

	if result.Status.Allowed {
		return true, "", nil
	}

	return false, result.Status.Reason, nil
}

func NewAuthorizorFromClient(client authclientv1.SubjectAccessReviewInterface) VirtApiAuthorizor {
	return &authorizor{
		userHeaders:             []string{userHeader},
		groupHeaders:            []string{groupHeader},
		userExtraHeaderPrefixes: []string{userExtraHeaderPrefix},
		client:                  client,
	}
}

func NewAuthorizor(rateLimiter flowcontrol.RateLimiter) (VirtApiAuthorizor, error) {
	config, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return nil, err
	}
	config.RateLimiter = rateLimiter

	client, err := authclientv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return NewAuthorizorFromClient(client.SubjectAccessReviews()), nil
}
