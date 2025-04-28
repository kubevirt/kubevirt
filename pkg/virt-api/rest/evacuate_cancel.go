package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/drain/evacuation"
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
			response.WriteHeader(http.StatusOK)
			return
		}

		if statusErr = app.validateEvacuationNode(vmi.Status.EvacuationNodeName); statusErr != nil {
			writeError(statusErr, response)
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

func (app *SubresourceAPIApp) validateEvacuationNode(evacuationNodeName string) *errors.StatusError {
	if taintKey := app.clusterConfig.GetMigrationConfiguration().NodeDrainTaintKey; taintKey != nil {
		taint := &k8sv1.Taint{
			Key:    *taintKey,
			Effect: k8sv1.TaintEffectNoSchedule,
		}

		node, err := app.virtCli.CoreV1().Nodes().Get(context.Background(), evacuationNodeName, k8smetav1.GetOptions{})
		if err != nil {
			return errors.NewInternalError(err)
		}

		if evacuation.NodeHasTaint(taint, node) {
			return errors.NewBadRequest(fmt.Sprintf("Node %q has NodeDrainTaintKey %q", node.Name, taint.String()))
		}
	}
	return nil
}
