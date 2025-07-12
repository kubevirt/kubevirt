package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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

		ctx := request.Request.Context()

		if statusErr = app.validateEvacuationNode(ctx, vmi); statusErr != nil {
			writeError(statusErr, response)
			return
		}

		if vmi.Status.EvacuationNodeName == "" {
			response.WriteHeader(http.StatusOK)
			return
		}

		opts := &v1.EvacuateCancelOptions{}
		if request.Request.Body != nil {
			defer request.Request.Body.Close()
			if err := decodeBody(request, opts); err != nil {
				writeError(err, response)
				return
			}
		}

		const path = "/status/evacuationNodeName"
		patchBytes, err := patch.New(patch.WithTest(path, vmi.Status.EvacuationNodeName), patch.WithRemove(path)).GeneratePayload()
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}

		_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(ctx, vmi.GetName(), types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: opts.DryRun})
		if err != nil {
			log.Log.Object(vmi).V(2).Reason(err).Info("Failed to patching VMI")
			writeError(errors.NewInternalError(err), response)
			return
		}

		response.WriteHeader(http.StatusOK)
	}
}

// validateEvacuationNode checks if the node hosting a VirtualMachineInstance (VMI) has a taint
// defined by NodeDrainTaintKey. This is part of a legacy mechanism for triggering VMI evacuation,
// which is now deprecated and should no longer be used. The recommended approach is to use node drain
// with taint-based eviction via Kubernetes eviction API.
//
// If EvacuationNodeName is not set in the VMI (e.g., due to compatibility with older versions),
// evacuation is not supported and an error will be returned. This function will eventually be removed.
func (app *SubresourceAPIApp) validateEvacuationNode(ctx context.Context, vmi *v1.VirtualMachineInstance) *errors.StatusError {
	// Use EvacuationNodeName if available, fallback to current node if empty.
	// Missing EvacuationNodeName indicates outdated VMI spec (pre-evacuation support).
	evacuationNodeName := vmi.Status.EvacuationNodeName
	if evacuationNodeName == "" {
		evacuationNodeName = vmi.Status.NodeName
	}

	if taintKey := app.clusterConfig.GetMigrationConfiguration().NodeDrainTaintKey; taintKey != nil {
		taint := &k8sv1.Taint{
			Key:    *taintKey,
			Effect: k8sv1.TaintEffectNoSchedule,
		}

		node, err := app.virtCli.CoreV1().Nodes().Get(ctx, evacuationNodeName, k8smetav1.GetOptions{})
		if err != nil {
			return errors.NewInternalError(err)
		}

		for _, t := range node.Spec.Taints {
			if t.MatchTaint(taint) {
				return errors.NewBadRequest(fmt.Sprintf(
					"Node %q has NodeDrainTaintKey %q",
					node.Name, taint.String(),
				))
			}
		}
	}
	return nil
}
