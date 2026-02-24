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

package fake

import (
	"context"

	"k8s.io/client-go/testing"

	templateapi "kubevirt.io/virt-template-api/core"
	"kubevirt.io/virt-template-api/core/subresourcesv1alpha1"
)

var virtualmachinetemplatesResource = subresourcesv1alpha1.GroupVersion.WithResource(templateapi.PluralResourceName)

func (f *fakeVirtualMachineTemplates) Process(_ context.Context, name string, options subresourcesv1alpha1.ProcessOptions) (*subresourcesv1alpha1.ProcessedVirtualMachineTemplate, error) {
	obj, err := f.Fake.Invokes(
		testing.NewCreateSubresourceAction(virtualmachinetemplatesResource, name, "process", f.Namespace(), &options),
		&subresourcesv1alpha1.ProcessedVirtualMachineTemplate{},
	)
	if obj == nil {
		return nil, err
	}
	return obj.(*subresourcesv1alpha1.ProcessedVirtualMachineTemplate), err
}

func (f *fakeVirtualMachineTemplates) CreateVirtualMachine(_ context.Context, name string, options subresourcesv1alpha1.ProcessOptions) (*subresourcesv1alpha1.ProcessedVirtualMachineTemplate, error) {
	obj, err := f.Fake.Invokes(
		testing.NewCreateSubresourceAction(virtualmachinetemplatesResource, name, "create", f.Namespace(), &options),
		&subresourcesv1alpha1.ProcessedVirtualMachineTemplate{},
	)
	if obj == nil {
		return nil, err
	}
	return obj.(*subresourcesv1alpha1.ProcessedVirtualMachineTemplate), err
}
