package rest

import (
	"github.com/emicklei/go-restful"
	"kubevirt/core/pkg/api/v1"
	"kubevirt/core/pkg/healthz"
)

var WebService *restful.WebService

func init() {
	WebService = new(restful.WebService)
	WebService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	WebService.ApiVersion(v1.GroupVersion.String()).Doc("help")
	restful.Add(WebService)
	WebService.Route(WebService.GET("/api/v1/healthz").To(healthz.KubeConnectionHealthzFunc).Doc("Health endpoint"))
}
