package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (app *SubresourceAPIApp) EvacuateCancelHandler(fetcher vmiFetcher) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		name := request.PathParameter("name")
		namespace := request.PathParameter("namespace")

		vmi, statusErr := fetcher(namespace, name)
		if statusErr != nil {
			writeError(statusErr, response)
			return
		}

		if vmi.Status.EvacuationNodeName == "" {
			writeError(errors.NewBadRequest(fmt.Sprintf("vmi %s/%s is not evacuated", namespace, name)), response)
			return
		}

		opts := &v1.EvacuateCancelOptions{}
		if request.Request.Body != nil {
			defer request.Request.Body.Close()
			err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
			switch err {
			case io.EOF, nil:
				break
			default:
				writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
				return
			}
		}

		patchBytes, err := patch.GenerateTestReplacePatch("/status/evacuationNodeName", vmi.Status.EvacuationNodeName, "")
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}

		_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(context.Background(), vmi.GetName(), types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: opts.DryRun})
		if err != nil {
			log.Log.Object(vmi).V(2).Reason(err).Info("Failed to patching VMI")
			writeError(errors.NewInternalError(err), response)
			return
		}

		response.WriteHeader(http.StatusOK)
	}
}
