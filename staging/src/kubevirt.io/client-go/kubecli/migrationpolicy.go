package kubecli

import (
	"context"

	v12 "kubevirt.io/api/core/v1"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/client-go/api/v1"
)

func (k *kubevirt) MigrationPolicy(namespace string) MigrationPolicyInterface {
	return &mp{k.restClient, namespace, "migrationpolicies"}
}

type mp struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (m *mp) Get(name string, options *k8smetav1.GetOptions) (*v12.MigrationPolicy, error) {
	policy := &v12.MigrationPolicy{}
	err := m.restClient.Get().
		Resource(m.resource).
		Namespace(m.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(policy)
	policy.SetGroupVersionKind(v1.MigrationPolicyKind)
	return policy, err
}

func (m *mp) List(opts *k8smetav1.ListOptions) (*v12.MigrationPolicyList, error) {
	policyList := &v12.MigrationPolicyList{}
	err := m.restClient.Get().
		Resource(m.resource).
		Namespace(m.namespace).
		VersionedParams(opts, scheme.ParameterCodec).
		Do(context.Background()).
		Into(policyList)
	for _, policy := range policyList.Items {
		policy.SetGroupVersionKind(v1.MigrationPolicyKind)
	}

	return policyList, err
}

func (m *mp) Create(policy *v12.MigrationPolicy) (*v12.MigrationPolicy, error) {
	result := &v12.MigrationPolicy{}
	err := m.restClient.Post().
		Namespace(m.namespace).
		Resource(m.resource).
		Body(policy).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.MigrationPolicyKind)
	return policy, err
}

func (m *mp) Update(policy *v12.MigrationPolicy) (*v12.MigrationPolicy, error) {
	result := &v12.MigrationPolicy{}
	err := m.restClient.Put().
		Name(policy.ObjectMeta.Name).
		Namespace(m.namespace).
		Resource(m.resource).
		Body(policy).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.MigrationPolicyKind)
	return policy, err
}

func (m *mp) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return m.restClient.Delete().
		Namespace(m.namespace).
		Resource(m.resource).
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()
}

func (m *mp) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v12.MigrationPolicy, error) {
	result := &v12.MigrationPolicy{}
	err := m.restClient.Patch(pt).
		Namespace(m.namespace).
		Resource(m.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return result, err
}

func (m *mp) UpdateStatus(policy *v12.MigrationPolicy) (*v12.MigrationPolicy, error) {
	result := &v12.MigrationPolicy{}
	err := m.restClient.Put().
		Name(policy.Name).
		Namespace(m.namespace).
		Resource(m.resource).
		SubResource("status").
		Body(policy).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.MigrationPolicyKind)
	return policy, err
}

func (m *mp) PatchStatus(name string, pt types.PatchType, data []byte) (*v12.MigrationPolicy, error) {
	result := &v12.MigrationPolicy{}
	err := m.restClient.Patch(pt).
		Namespace(m.namespace).
		Resource(m.resource).
		SubResource("status").
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return result, err
}
