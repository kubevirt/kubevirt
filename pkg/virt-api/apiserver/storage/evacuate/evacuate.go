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

package evacuate

import (
	"context"
	"fmt"
	"io"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const unmarshalRequestErrFmt = "Can not unmarshal Request body to struct, error: %s"

type vmiFetcher func(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError)

// Handler contains the VirtualMachine and VirtualMachineInstance evacuate-cancel
// operations served by the GenericAPIServer storage Connecters.
type Handler struct {
	virtCli       kubecli.KubevirtClient
	clusterConfig *virtconfig.ClusterConfig
}

func NewHandler(virtCli kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig) *Handler {
	return &Handler{
		virtCli:       virtCli,
		clusterConfig: clusterConfig,
	}
}

// EvacuateCancelVMI cancels a pending evacuation of a VirtualMachineInstance.
func (h *Handler) EvacuateCancelVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	return h.evacuateCancel(ctx, namespace, name, body, h.fetchVirtualMachineInstance)
}

// EvacuateCancelVM cancels a pending evacuation of a VirtualMachine's VMI.
func (h *Handler) EvacuateCancelVM(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	return h.evacuateCancel(ctx, namespace, name, body, h.fetchVirtualMachineInstanceForVM)
}

func (h *Handler) evacuateCancel(ctx context.Context, namespace, name string, body io.ReadCloser, fetcher vmiFetcher) *errors.StatusError {
	vmi, statusErr := fetcher(namespace, name)
	if statusErr != nil {
		return statusErr
	}

	if statusErr = h.validateEvacuationNode(ctx, vmi); statusErr != nil {
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

	if _, err = h.virtCli.VirtualMachineInstance(namespace).Patch(ctx, vmi.GetName(), types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: opts.DryRun}); err != nil {
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
func (h *Handler) validateEvacuationNode(ctx context.Context, vmi *v1.VirtualMachineInstance) *errors.StatusError {
	var taintKey *string
	if taintKey = h.clusterConfig.GetMigrationConfiguration().NodeDrainTaintKey; taintKey == nil {
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

	node, err := h.virtCli.CoreV1().Nodes().Get(ctx, evacuationNodeName, k8smetav1.GetOptions{})
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

func (h *Handler) fetchVirtualMachineInstance(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}
	return vmi, nil
}

func (h *Handler) fetchVirtualMachineInstanceForVM(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vm, err := h.virtCli.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	if !vm.Status.Created {
		return nil, errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf("VMI is not started"))
	}

	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}

	for _, ref := range vmi.OwnerReferences {
		if ref.UID == vm.UID {
			return vmi, nil
		}
	}

	return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s] for vm: %v", name, err))
}

func decodeBodyReader(body io.Reader, bodyStruct interface{}) *errors.StatusError {
	err := yaml.NewYAMLOrJSONDecoder(body, 1024).Decode(bodyStruct)
	switch err {
	case io.EOF, nil:
		return nil
	default:
		return errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err))
	}
}
