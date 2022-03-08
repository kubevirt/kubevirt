package client

import (
	"context"
	"encoding/json"
	"reflect"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var createdNodes = make(map[string]nodeInfo, 0)
var nodeSpecMap = make(map[string]nodeSpec, 0)

type nodeSpec struct {
	labels        map[string]string
	unschedulable bool
	taints        []v1.Taint
}

type nodeInfo struct {
	name string
	ctx  context.Context
}

// nodes implements NodesInterface
type nodes struct {
	v12.NodeInterface
}

// newNodes returns a Nodes
func newNodes(c *TestCoreV1Client) *nodes {
	return &nodes{
		c.CoreV1Client.Nodes(),
	}
}

func (c *nodes) Clean() {
	for _, node := range createdNodes {
		err := c.NodeInterface.Delete(node.ctx, node.name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			panic(err)
		}
	}

	for name, nodeSpec := range nodeSpecMap {
		currentNode, err := c.NodeInterface.Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		if reflect.DeepEqual(currentNode.Labels, nodeSpec.labels) &&
			currentNode.Spec.Unschedulable == nodeSpec.unschedulable &&
			reflect.DeepEqual(currentNode.Spec.Taints, nodeSpec.taints) {
			return
		}

		oldNode, err := json.Marshal(currentNode)
		Expect(err).ToNot(HaveOccurred())
		newNode := currentNode.DeepCopy()

		newNode.Spec.Taints = nodeSpec.taints
		newNode.Labels = nodeSpec.labels
		newNode.Spec.Unschedulable = nodeSpec.unschedulable
		newJson, err := json.Marshal(newNode)
		Expect(err).ToNot(HaveOccurred())

		patch, err := strategicpatch.CreateTwoWayMergePatch(oldNode, newJson, currentNode)
		Expect(err).ToNot(HaveOccurred())

		_, err = c.NodeInterface.Patch(context.Background(), currentNode.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
	}
}

func (c *nodes) Create(ctx context.Context, node *v1.Node, opts metav1.CreateOptions) (result *v1.Node, err error) {
	created, err := c.NodeInterface.Create(ctx, node, opts)
	if err == nil && opts.DryRun == nil {
		createdNodes[node.Name] = nodeInfo{created.Name, ctx}
	}

	return created, err
}

func (c *nodes) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	err := c.NodeInterface.Delete(ctx, name, opts)
	if _, exist := createdNodes[name]; exist && err == nil && opts.DryRun == nil {
		delete(createdNodes, name)
	}

	return err
}

func (c *nodes) Update(ctx context.Context, node *v1.Node, opts metav1.UpdateOptions) (result *v1.Node, err error) {
	if opts.DryRun == nil {
		c.saveNodeSpecs(ctx, node.Name)
	}

	return c.NodeInterface.Update(ctx, node, opts)
}

func (c *nodes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Node, err error) {
	if opts.DryRun == nil {
		c.saveNodeSpecs(ctx, name)
	}

	return c.NodeInterface.Patch(ctx, name, pt, data, opts, subresources...)
}

func (c *nodes) saveNodeSpecs(ctx context.Context, name string) {
	if _, exist := createdNodes[name]; exist {
		//node will be deleted
		return
	}

	if _, exist := nodeSpecMap[name]; exist {
		//node information has already been saved
		return
	}

	//save the information before the first update
	currentNode, err := c.NodeInterface.Get(ctx, name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	nodeSpecMap[currentNode.Name] = nodeSpec{
		labels:        currentNode.Labels,
		unschedulable: currentNode.Spec.Unschedulable,
		taints:        currentNode.Spec.Taints,
	}
}
