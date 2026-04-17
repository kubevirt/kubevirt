package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/pflag"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilcache "k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	authnclientv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authzclientv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
)

const defaultTokenCacheTTL = 2 * time.Minute

type ResourceAttributes struct {
	Group       string
	Version     string
	Resource    string
	Namespace   string
	Name        string
	Subresource string
}

func (r *ResourceAttributes) Validate() error {
	if r.Group == "" {
		return fmt.Errorf("group is required")
	}
	if r.Version == "" {
		return fmt.Errorf("version is required")
	}
	if r.Resource == "" {
		return fmt.Errorf("resource is required")
	}
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Subresource == "" {
		return fmt.Errorf("subresource is required")
	}
	return nil
}

func (r *ResourceAttributes) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.Group, "auth-attr-group", r.Group, "Group to use for authorization")
	fs.StringVar(&r.Version, "auth-attr-version", r.Version, "Version to use for authorization")
	fs.StringVar(&r.Resource, "auth-attr-resource", r.Resource, "Resource to use for authorization")
	fs.StringVar(&r.Namespace, "auth-attr-namespace", r.Namespace, "Namespace to use for authorization")
	fs.StringVar(&r.Name, "auth-attr-name", r.Name, "Name to use for authorization")
	fs.StringVar(&r.Subresource, "auth-attr-subresource", r.Subresource, "Subresource to use for authorization")
}

// tokenReviewAuthn implements authenticator.Request via the TokenReview API.
type tokenReviewAuthn struct {
	client authnclientv1.AuthenticationV1Interface
	cache  *utilcache.Expiring
	ttl    time.Duration
}

func NewAuthenticator(client authnclientv1.AuthenticationV1Interface) authenticator.Request {
	return &tokenReviewAuthn{
		client: client,
		cache:  utilcache.NewExpiring(),
		ttl:    defaultTokenCacheTTL,
	}
}

func (a *tokenReviewAuthn) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	token := extractBearerToken(req)
	if token == "" {
		return nil, false, nil
	}

	if cached, ok := a.cache.Get(token); ok {
		return cached.(*authenticator.Response), true, nil
	}

	tr := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{Token: token},
	}
	result, err := a.client.TokenReviews().Create(req.Context(), tr, metav1.CreateOptions{})
	if err != nil {
		return nil, false, err
	}
	if !result.Status.Authenticated {
		return nil, false, nil
	}

	resp := &authenticator.Response{
		User: &user.DefaultInfo{
			Name:   result.Status.User.Username,
			UID:    result.Status.User.UID,
			Groups: result.Status.User.Groups,
			Extra:  convertAuthnExtra(result.Status.User.Extra),
		},
	}
	a.cache.Set(token, resp, a.ttl)
	return resp, true, nil
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

func convertAuthnExtra(in map[string]authenticationv1.ExtraValue) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = []string(v)
	}
	return out
}

// sarAuthz implements authorizer.Authorizer via the SubjectAccessReview API.
type sarAuthz struct {
	client authzclientv1.AuthorizationV1Interface
}

func NewAuthorizer(client authzclientv1.AuthorizationV1Interface) authorizer.Authorizer {
	return &sarAuthz{client: client}
}

func (a *sarAuthz) Authorize(ctx context.Context, attr authorizer.Attributes) (authorizer.Decision, string, error) {
	extra := convertAuthzExtra(attr.GetUser().GetExtra())

	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   attr.GetUser().GetName(),
			Groups: attr.GetUser().GetGroups(),
			UID:    attr.GetUser().GetUID(),
			Extra:  extra,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   attr.GetNamespace(),
				Verb:        attr.GetVerb(),
				Group:       attr.GetAPIGroup(),
				Version:     attr.GetAPIVersion(),
				Resource:    attr.GetResource(),
				Subresource: attr.GetSubresource(),
				Name:        attr.GetName(),
			},
		},
	}

	result, err := a.client.SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return authorizer.DecisionNoOpinion, "", err
	}
	if result.Status.Allowed {
		return authorizer.DecisionAllow, result.Status.Reason, nil
	}
	return authorizer.DecisionDeny, result.Status.Reason, nil
}

func convertAuthzExtra(in map[string][]string) map[string]authorizationv1.ExtraValue {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]authorizationv1.ExtraValue, len(in))
	for k, v := range in {
		out[k] = authorizationv1.ExtraValue(v)
	}
	return out
}
