/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package rest

import (
	"context"
	"fmt"
	"io"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

type vmiFetcher func(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError)

// EvacuateCancelVMI cancels a pending evacuation of a VirtualMachineInstance.
// It is served by the aggregated apiserver under the nested subresource path
// virtualmachineinstances/evacuate/cancel
func (app *SubresourceAPIApp) EvacuateCancelVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	return app.evacuateCancel(ctx, namespace, name, body, app.FetchVirtualMachineInstance)
}

func (app *SubresourceAPIApp) EvacuateCancelVM(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	return app.evacuateCancel(ctx, namespace, name, body, app.FetchVirtualMachineInstanceForVM)
}

func (app *SubresourceAPIApp) evacuateCancel(ctx context.Context, namespace, name string, body io.ReadCloser, fetcher vmiFetcher) *errors.StatusError {
	vmi, statusErr := fetcher(namespace, name)
	if statusErr != nil {
		return statusErr
	}

	if statusErr = app.validateEvacuationNode(ctx, vmi); statusErr != nil {
		return statusErr
	}

	if vmi.Status.EvacuationNodeName == "" {
		return nil
	}

	if body == nil {
		return errors.NewBadRequest("No body")
	}

	opts := &v1.EvacuateCancelOptions{}
	defer body.Close()
	if statusErr = decodeBodyReader(body, opts); statusErr != nil {
		return statusErr
	}

	const path = "/status/evacuationNodeName"
	patchBytes, err := patch.New(patch.WithTest(path, opts.EvacuationNodeName),
		patch.WithRemove(path)).GeneratePayload()
	if err != nil {
		return errors.NewInternalError(err)
	}

	if _, err = app.virtClient.VirtualMachineInstance(namespace).Patch(ctx, vmi.GetName(), types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: opts.DryRun}); err != nil {
		log.Log.Object(vmi).V(2).Reason(err).Info("Failed to patching VMI")
		return errors.NewInternalError(err)
	}

	return nil
}

// validateEvacuationNode checks if the node hosting a VirtualMachineInstance (VMI) has a taint
// defined by NodeDrainTaintKey. This is part of a legacy mechanism for triggering VMI evacuation,
// which is now deprecated and should no longer be used. The recommended approach is to use node drain
// with taint-based eviction via Kubernetes eviction API.
//
// If EvacuationNodeName is not set in the VMI (e.g., due to compatibility with older versions),
// evacuation is not supported and an error will be returned. This function will eventually be removed.
func (app *SubresourceAPIApp) validateEvacuationNode(ctx context.Context, vmi *v1.VirtualMachineInstance) *errors.StatusError {
	var taintKey *string
	if taintKey = app.clusterConfig.GetMigrationConfiguration().NodeDrainTaintKey; taintKey == nil {
		return nil
	}

	// Use EvacuationNodeName if available, fallback to current node if empty.
	// Missing EvacuationNodeName indicates outdated VMI spec (pre-evacuation support).
	evacuationNodeName := vmi.Status.EvacuationNodeName
	if evacuationNodeName == "" {
		evacuationNodeName = vmi.Status.NodeName
	}

	taint := &k8sv1.Taint{
		Key:    *taintKey,
		Effect: k8sv1.TaintEffectNoSchedule,
	}

	node, err := app.k8sClient.CoreV1().Nodes().Get(ctx, evacuationNodeName, k8smetav1.GetOptions{})
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

	return nil
}
