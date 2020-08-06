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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virtwrap

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("PostCopy", func() {

	table.DescribeTable("should return if vmi has local storage",
		func(vmi *v1.VirtualMachineInstance, expected bool) {
			result := vmiHasLocalStorage(vmi)
			Expect(result).To(Equal(expected))
		},
		table.Entry("has local volume", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{
					{
						VolumeSource: v1.VolumeSource{
							EmptyDisk: &v1.EmptyDiskSource{
								Capacity: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		}, true),
		table.Entry("does not have local volume", &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{
					{
						VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{
								Image: "container/image",
							},
						},
					},
				},
			},
		}, false),
	)

})
