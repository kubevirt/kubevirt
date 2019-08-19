package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetWorkerNodes returns all nodes with the nodeWorkerRoleLabel label
func GetWorkerNodes(c client.Client) ([]corev1.Node, error) {
	workerNodes := &corev1.NodeList{}
	err := c.List(
		context.TODO(),
		workerNodes,
		client.InNamespace(NamespaceOpenShiftMachineAPI),
		client.MatchingLabels(map[string]string{WorkerNodeRoleLabel: ""}),
	)
	if err != nil {
		return nil, err
	}
	return workerNodes.Items, nil
}

// FilterReadyNodes fileter the list of nodes and returns the list with ready nodes
func FilterReadyNodes(nodes []corev1.Node) []corev1.Node {
	var readyNodes []corev1.Node
	for _, n := range nodes {
		if IsNodeReady(&n) {
			readyNodes = append(readyNodes, n)
		}
	}
	return readyNodes
}

// IsNodeReady returns true once node is ready
func IsNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
