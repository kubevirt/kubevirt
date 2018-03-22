package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"

	authorization "k8s.io/api/authorization/v1beta1"
	authorizationclient "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"

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
}

type authorizor struct {
	subjectAccessReview authorizationclient.SubjectAccessReviewInterface
}

func getUserGroups(header http.Header) ([]string, error) {
	groups, ok := header[groupHeader]
	if ok == false {
		return nil, fmt.Errorf("%s header is required for authorization", groupHeader)
	}
	return groups, nil
}

func getUserName(header http.Header) (string, error) {
	user, ok := header[userHeader]
	if ok == false {
		return "", fmt.Errorf("%s header is required for authorization", userHeader)
	}
	return user[0], nil
}

func getUserExtras(header http.Header) map[string]authorization.ExtraValue {

	var extras map[string]authorization.ExtraValue

	for k, v := range header {
		if strings.HasPrefix(k, userExtraHeaderPrefix) {
			extraKey := strings.TrimPrefix(k, userExtraHeaderPrefix)
			extras[extraKey] = v
		}
	}

	return extras
}

func generateAccessReview(req *restful.Request) (*authorization.SubjectAccessReview, error) {

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
	// /apis/subresources.kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm/console
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
	userExtras := getUserExtras(headers)

	if resource != "virtualmachines" {
		return nil, fmt.Errorf("unknown resource type %s", resource)
	}

	userName, err := getUserName(headers)
	if err != nil {
		return nil, err
	}

	userGroups, err := getUserGroups(headers)
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

func isInfoEndpoint(req *restful.Request) bool {

	httpRequest := req.Request
	if httpRequest == nil || httpRequest.URL == nil {
		return false
	}
	// URL example
	// /apis/subresources.kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm/console
	// The /apis/<group>/<version> part of the urls should be accessible without needing authorization
	pathSplit := strings.Split(httpRequest.URL.Path, "/")
	if len(pathSplit) <= 4 {
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
	if isInfoEndpoint(req) {
		return true, "", nil
	}

	if !isAuthenticated(req) {
		return false, "request is not authenticated", nil
	}

	r, err := generateAccessReview(req)
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

func NewAuthorizor() (VirtApiAuthorizor, error) {
	config, err := kubecli.GetConfig()
	if err != nil {
		return nil, err
	}
	config.QPS = clientQPS
	config.Burst = clientBurst

	client, err := authorizationclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	subjectAccessReview := client.SubjectAccessReviews()

	return &authorizor{
		subjectAccessReview: subjectAccessReview,
	}, err
}
