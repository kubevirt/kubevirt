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

package v1alpha1

import (
	"context"
	"fmt"

	templateapi "kubevirt.io/virt-template-api/core"
	"kubevirt.io/virt-template-api/core/subresourcesv1alpha1"
)

const subresourceURLFmt = "/apis/%s/%s"

type VirtualMachineTemplateExpansion interface {
	Process(ctx context.Context, name string, options subresourcesv1alpha1.ProcessOptions) (*subresourcesv1alpha1.ProcessedVirtualMachineTemplate, error)
	CreateVirtualMachine(ctx context.Context, name string, options subresourcesv1alpha1.ProcessOptions) (*subresourcesv1alpha1.ProcessedVirtualMachineTemplate, error)
}

func (c *virtualMachineTemplates) Process(ctx context.Context, name string, options subresourcesv1alpha1.ProcessOptions) (*subresourcesv1alpha1.ProcessedVirtualMachineTemplate, error) {
	result := &subresourcesv1alpha1.ProcessedVirtualMachineTemplate{}
	err := c.GetClient().
		Post().
		AbsPath(fmt.Sprintf(subresourceURLFmt, subresourcesv1alpha1.GroupVersion.Group, subresourcesv1alpha1.GroupVersion.Version)).
		Namespace(c.GetNamespace()).
		Resource(templateapi.PluralResourceName).
		Name(name).
		SubResource("process").
		Body(&options).
		Do(ctx).
		Into(result)
	return result, err
}

func (c *virtualMachineTemplates) CreateVirtualMachine(ctx context.Context, name string, options subresourcesv1alpha1.ProcessOptions) (*subresourcesv1alpha1.ProcessedVirtualMachineTemplate, error) {
	result := &subresourcesv1alpha1.ProcessedVirtualMachineTemplate{}
	err := c.GetClient().
		Post().
		AbsPath(fmt.Sprintf(subresourceURLFmt, subresourcesv1alpha1.GroupVersion.Group, subresourcesv1alpha1.GroupVersion.Version)).
		Namespace(c.GetNamespace()).
		Resource(templateapi.PluralResourceName).
		Name(name).
		SubResource("create").
		Body(&options).
		Do(ctx).
		Into(result)
	return result, err
}
