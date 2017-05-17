package rest

import (
	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
)

var WebService *restful.WebService

func init() {
	WebService = new(restful.WebService)
	WebService.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	WebService.Route(WebService.GET("/apis/" + v1.GroupVersion.String() + "/healthz").To(healthz.KubeConnectionHealthzFunc).Doc("Health endpoint"))
}
