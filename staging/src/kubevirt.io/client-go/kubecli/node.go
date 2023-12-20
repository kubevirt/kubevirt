package kubecli

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
	v1 "kubevirt.io/api/core/v1"
)

func (k *kubevirt) ShadowNodeClient() ShadowNodeInterface {
	return &shadowNode{
		restClient: k.restClient,
		config:     k.config,
		resource:   "shadownodes",
	}

}

type shadowNode struct {
	restClient *rest.RESTClient
	config     *rest.Config
	resource   string
}

func (n *shadowNode) Update(node *v1.ShadowNode) (*v1.ShadowNode, error) {
	updatedNode := &v1.ShadowNode{}
	err := n.restClient.Put().
		Resource(n.resource).
		Name(node.Name).
		Body(node).
		Do(context.TODO()).
		Into(updatedNode)

	return updatedNode, err
}

func (n *shadowNode) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.ShadowNode, error) {
	updatedNode := &v1.ShadowNode{}
	err := n.restClient.Patch(pt).
		Resource(n.resource).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Body(data).
		Do(ctx).
		Into(updatedNode)
	return updatedNode, err
}

func (n *shadowNode) Create(ctx context.Context, shadowNode *v1.ShadowNode, opts metav1.CreateOptions) (*v1.ShadowNode, error) {
	updatedNode := &v1.ShadowNode{}
	err := n.restClient.Post().
		Resource(n.resource).
		Body(shadowNode).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(updatedNode)

	return updatedNode, err
}

func (n *shadowNode) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ShadowNode, error) {
	updatedNode := &v1.ShadowNode{}
	err := n.restClient.Get().
		Resource(n.resource).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(updatedNode)

	return updatedNode, err
}

func (n *shadowNode) List(ctx context.Context, opts metav1.ListOptions) (*v1.ShadowNodeList, error) {
	shadowNodeList := &v1.ShadowNodeList{}

	err := n.restClient.Get().
		Resource(n.resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(shadowNodeList)

	return shadowNodeList, err
}
