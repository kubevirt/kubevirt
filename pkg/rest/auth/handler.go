package auth

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

// Middleware performs delegated authentication and authorization for HTTP handlers.
type Middleware struct {
	authn authenticator.Request
	authz authorizer.Authorizer
	attrs ResourceAttributes
}

func NewMiddlewareFromKubevirtClient(client kubecli.KubevirtClient, attrs ResourceAttributes) *Middleware {
	authn := NewAuthenticator(client.AuthenticationV1())
	authz := NewAuthorizer(client.AuthorizationV1())

	return NewMiddleware(authn, authz, attrs)
}

func NewMiddleware(authn authenticator.Request, authz authorizer.Authorizer, attrs ResourceAttributes) *Middleware {
	return &Middleware{authn: authn, authz: authz, attrs: attrs}
}

// Handler wraps next with authn+authz checks.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok, err := m.authn.AuthenticateRequest(r)
		if err != nil {
			log.Log.Infof("auth: authentication error: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		record := authorizer.AttributesRecord{
			User:            resp.User,
			Verb:            httpMethodToVerb(r.Method),
			Namespace:       m.attrs.Namespace,
			APIGroup:        m.attrs.Group,
			APIVersion:      m.attrs.Version,
			Resource:        m.attrs.Resource,
			Subresource:     m.attrs.Subresource,
			Name:            m.attrs.Name,
			ResourceRequest: true,
		}

		decision, reason, err := m.authz.Authorize(r.Context(), record)
		if err != nil {
			msg := fmt.Sprintf("authorization error (user=%s, verb=%s, resource=%s/%s)",
				resp.User.GetName(), record.Verb, record.Resource, record.Subresource)
			log.Log.Errorf("auth: %s: %v", msg, err)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		if decision != authorizer.DecisionAllow {
			log.Log.Infof("auth: authorization denied user=%s verb=%s resource=%s/%s reason=%q",
				resp.User.GetName(), record.Verb, record.Resource, record.Subresource, reason)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RouteFunction returns a go-restful RouteFunction that applies the middleware.
func (m *Middleware) RouteFunction(next http.Handler) restful.RouteFunction {
	return func(req *restful.Request, resp *restful.Response) {
		m.Handler(next).ServeHTTP(resp.ResponseWriter, req.Request)
	}
}

func httpMethodToVerb(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodGet, http.MethodHead:
		return "get"
	case http.MethodPut:
		return "update"
	case http.MethodPatch:
		return "patch"
	case http.MethodDelete:
		return "delete"
	default:
		return "*"
	}
}
