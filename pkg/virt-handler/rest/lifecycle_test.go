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

package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
)

var _ = Describe("Snapshot overlay directory", func() {
	const overlayVolume = "snap-scratch-1234"

	newRequest := func(body io.Reader) *restful.Request {
		httpRequest, err := http.NewRequest(http.MethodPut, "/", body)
		Expect(err).ToNot(HaveOccurred())
		if body == nil {
			httpRequest.Body = nil
		}
		return &restful.Request{Request: httpRequest}
	}

	newVMI := func(utilityVolumeNames ...string) *v1.VirtualMachineInstance {
		vmi := api.NewMinimalVMI("testvmi")
		for _, name := range utilityVolumeNames {
			vmi.Spec.UtilityVolumes = append(vmi.Spec.UtilityVolumes, v1.UtilityVolume{Name: name})
		}
		return vmi
	}

	optionsBody := func(volumeName string) io.Reader {
		body, err := json.Marshal(&v1.SnapshotOverlayOptions{VolumeName: volumeName})
		Expect(err).ToNot(HaveOccurred())
		return strings.NewReader(string(body))
	}

	It("should resolve an attached utility volume to its mount point", func() {
		overlayDir, err := overlayDirFromRequest(newRequest(optionsBody(overlayVolume)), newVMI(overlayVolume))

		Expect(err).ToNot(HaveOccurred())
		Expect(overlayDir).To(HaveSuffix("/" + overlayVolume))
	})

	It("should reject a volume that is not attached to the VMI", func() {
		_, err := overlayDirFromRequest(newRequest(optionsBody(overlayVolume)), newVMI("some-other-volume"))

		Expect(err).To(MatchError(ContainSubstring(overlayVolume)))
	})

	It("should reject a path instead of a volume name", func() {
		_, err := overlayDirFromRequest(newRequest(optionsBody("../../../etc")), newVMI(overlayVolume))

		Expect(err).To(HaveOccurred())
	})

	It("should reject a request with no body", func() {
		_, err := overlayDirFromRequest(newRequest(nil), newVMI(overlayVolume))

		Expect(err).To(HaveOccurred())
	})
})
