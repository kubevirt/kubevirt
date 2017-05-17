package filter

import (
	"strings"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/logging"
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
		logging.DefaultLogger().Info().
			With("remoteAddress", strings.Split(req.Request.RemoteAddr, ":")[0]).
			With("username", username).
			With("method", req.Request.Method).
			With("url", req.Request.URL.RequestURI()).
			With("proto", req.Request.Proto).
			With("statusCode", resp.StatusCode()).
			Log("contentLength", resp.ContentLength())
	}
}
