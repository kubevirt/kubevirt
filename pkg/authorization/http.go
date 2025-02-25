package authorization

import (
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func HttpWithBearerToken(config *rest.Config, httpClient *http.Client) (server.Filter, error) {
	return func(log logr.Logger, handler http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			authValue := req.Header.Get("Authorization")
			token := strings.TrimPrefix(authValue, "Bearer ")

			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			valid, err := ValidateToken(token)
			if err != nil || !valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			handler.ServeHTTP(w, req)
		}), nil
	}, nil
}
