package migrations

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

type NetworkAccessibilityManager struct {
	virtClient kubecli.KubevirtClient
}

type NetworkPriority = string

const (
	NetworkPriorityTheHighest NetworkPriority = "0"
	NetworkPriorityDecreased NetworkPriority = "1"
	NetworkPriorityDeferred NetworkPriority = "2"
)

func NewNetworkAccessibilityManager(virtClient kubecli.KubevirtClient) *NetworkAccessibilityManager  {
	return &NetworkAccessibilityManager{
		virtClient: virtClient,
	}
}

func (m NetworkAccessibilityManager) SetTheHighestNetworkPriority(ctx context.Context, pod types.NamespacedName) error {
	patchBytes, err := patch.New(
		patch.WithTest(fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(virtv1.NetworkPriorityLabel)), NetworkPriorityDeferred),
		patch.WithReplace(fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(virtv1.NetworkPriorityLabel)), NetworkPriorityTheHighest),
	).GeneratePayload()
	if err != nil {
		return fmt.Errorf("generate patch to set new network priority %s=%s for the pod %s: %w", virtv1.NetworkPriorityLabel, NetworkPriorityTheHighest, pod, err)
	}

	_, err = m.virtClient.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("apply patch to set new network priority %s=%s for the pod %s: %w", virtv1.NetworkPriorityLabel, NetworkPriorityTheHighest, pod, err)
	}

	return nil
}

func (m NetworkAccessibilityManager) DecreaseNetworkPriority(ctx context.Context, pod types.NamespacedName) error {
	patchBytes, err := patch.New(
		patch.WithReplace(fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(virtv1.NetworkPriorityLabel)), NetworkPriorityDecreased),
	).GeneratePayload()
	if err != nil {
		return fmt.Errorf("generate patch to set new network priority %s=%s for the pod %s: %w", virtv1.NetworkPriorityLabel, NetworkPriorityDecreased, pod, err)
	}

	_, err = m.virtClient.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("apply patch to set new network priority %s=%s for the pod %s: %w", virtv1.NetworkPriorityLabel, NetworkPriorityDecreased, pod, err)
	}

	return nil
}
