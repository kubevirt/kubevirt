package rest

import (
	"github.com/emicklei/go-restful"
	"kubevirt.io/kubevirt/pkg/healthz"
)

var WebService *restful.WebService

func init() {
	WebService = new(restful.WebService)
	WebService.Path("/api/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	WebService.Route(WebService.GET("healthz").To(healthz.KubeConnectionHealthzFunc))
}
