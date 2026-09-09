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

package virtualmachine

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/expand"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type vmExpander interface {
	Expand(vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
}

// ExpandSpecREST serves the expand-spec subresource (GET) of the
// subresources.kubevirt.io API group as a rest.Getter storage.
type ExpandSpecREST struct {
	virtClient kubecli.KubevirtClient
	expander   vmExpander
}

func NewExpandSpecREST(virtClient kubecli.KubevirtClient, clusterConfig *virtconfig.ClusterConfig) *ExpandSpecREST {
	return &ExpandSpecREST{
		virtClient: virtClient,
		expander: expand.New(
			clusterConfig,
			find.NewSpecFinder(nil, nil, nil, virtClient),
			preferenceFind.NewSpecFinder(nil, nil, nil, virtClient),
		),
	}
}

var (
	_ = rest.Storage(&ExpandSpecREST{})
	_ = rest.Getter(&ExpandSpecREST{})
	_ = rest.GroupVersionKindProvider(&ExpandSpecREST{})
)

func (r *ExpandSpecREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *ExpandSpecREST) Destroy() {}

// GroupVersionKind makes the aggregated API server encode the expand-spec
// response as kubevirt.io/v1 VirtualMachine, matching the apiVersion the
// legacy go-restful handler returned instead of the serving group
// subresources.kubevirt.io/v1. This keeps existing clients working, they can
// decode the response with their kubevirt.io/v1 scheme without any change.
func (r *ExpandSpecREST) GroupVersionKind(schema.GroupVersion) schema.GroupVersionKind {
	return v1.VirtualMachineGroupVersionKind
}

func (r *ExpandSpecREST) Get(ctx context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to expand a VirtualMachine spec")
	}

	vm, err := r.virtClient.VirtualMachine(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, apierrors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %w", name, err))
	}

	expandedVM, err := r.expander.Expand(vm)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	return expandedVM, nil
}
