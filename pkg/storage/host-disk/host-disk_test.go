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

package hostdisk

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/diskimg"
)

var _ = Describe("HostDisk", func() {
	Describe("Images", func() {
		It("should only return images for DiskOrCreate HostDisks", func() {
			vmi := libvmi.New(
				libvmi.WithHostDisk("exists", "/data/exists/disk.img", v1.HostDiskExists),
				libvmi.WithHostDiskAndCapacity("create", "/data/create/my.img", v1.HostDiskExistsOrCreate, "64Mi"),
				libvmi.WithHostDisk("unknown", "/data/unknown/disk.img", "UnknownType"),
				libvmi.WithPersistentVolumeClaim("pvc", "claim"),
			)

			Expect(Images(vmi)).To(Equal([]diskimg.Image{{
				Dir:  GetMountedHostDiskDir("create"),
				Name: "my.img",
				Size: 64 << 20,
			}}))
		})
	})
})
