package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"kubevirt.io/client-go/kubecli"
)

func PatchNode(client kubecli.KubevirtClient, original, modified *corev1.Node) error {
	originalBytes, err := json.Marshal(original)
	if err != nil {
		return fmt.Errorf("could not serialize original object: %v", err)
	}
	modifiedBytes, err := json.Marshal(modified)
	if err != nil {
		return fmt.Errorf("could not serialize modified object: %v", err)
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, modifiedBytes, corev1.Node{})
	if err != nil {
		return fmt.Errorf("could not create merge patch: %v", err)
	}
	if _, err := client.CoreV1().Nodes().Patch(context.Background(), original.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("could not patch the node: %v", err)
	}
	return nil
}
