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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package statsconv

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	libvirt "libvirt.org/libvirt-go"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/statsconv/util"
)

var _ = Describe("StatsConverter", func() {
	var mockDomainIdent *MockDomainIdentifier
	var ctrl *gomock.Controller
	var testStats []libvirt.DomainStats

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockDomainIdent = NewMockDomainIdentifier(ctrl)
		testStats, _ = util.LoadStats()
	})

	Context("on conversion attempt", func() {
		It("should handle empty input", func() {
			in := &libvirt.DomainStats{}
			inMem := []libvirt.DomainMemoryStat{}
			devAliasMap := make(map[string]string)
			out := stats.DomainStats{}
			mockDomainIdent.EXPECT().GetName().Return("testName", nil)
			mockDomainIdent.EXPECT().GetUUIDString().Return("testUUID", nil)
			ident := DomainIdentifier(mockDomainIdent)

			err := Convert_libvirt_DomainStats_to_stats_DomainStats(ident, in, inMem, nil, devAliasMap, &out)

			Expect(err).To(BeNil())
			Expect(out.Name).To(Equal("testName"))
			Expect(out.UUID).To(Equal("testUUID"))
		})

		It("should handle valid input", func() {
			in := &testStats[0]
			inMem := []libvirt.DomainMemoryStat{}
			devAliasMap := make(map[string]string)
			out := stats.DomainStats{}
			mockDomainIdent.EXPECT().GetName().Return("testName", nil)
			mockDomainIdent.EXPECT().GetUUIDString().Return("testUUID", nil)
			ident := DomainIdentifier(mockDomainIdent)

			err := Convert_libvirt_DomainStats_to_stats_DomainStats(ident, in, inMem, nil, devAliasMap, &out)

			Expect(err).To(BeNil())
			// very very basic sanity check
			Expect(out.Cpu).To(Not(BeNil()))
			Expect(out.Memory).To(Not(BeNil()))
			Expect(len(out.Vcpu)).To(Equal(len(testStats[0].Vcpu)))
			Expect(len(out.Net)).To(Equal(len(testStats[0].Net)))
			Expect(len(out.Block)).To(Equal(len(testStats[0].Block)))
		})

		It("should convert valid input", func() {
			in := &testStats[0]
			inMem := []libvirt.DomainMemoryStat{}
			devAliasMap := make(map[string]string)
			out := stats.DomainStats{}
			mockDomainIdent.EXPECT().GetName().Return("testName", nil)
			mockDomainIdent.EXPECT().GetUUIDString().Return("testUUID", nil)
			ident := DomainIdentifier(mockDomainIdent)

			err := Convert_libvirt_DomainStats_to_stats_DomainStats(ident, in, inMem, nil, devAliasMap, &out)

			Expect(err).To(BeNil())

			loaded := new(bytes.Buffer)
			enc := json.NewEncoder(loaded)
			err = enc.Encode(out)
			Expect(err).To(BeNil())

			equal, err := JSONEqual(loaded, strings.NewReader(util.Testdataexpected))
			Expect(err).To(BeNil())
			if !equal {
				enc := json.NewEncoder(os.Stderr)
				err = enc.Encode(out)
				Expect(err).To(BeNil())
			}
			Expect(equal).To(BeTrue())
		})
	})
})

func JSONEqual(a, b io.Reader) (bool, error) {
	var j, j2 interface{}
	d := json.NewDecoder(a)
	if err := d.Decode(&j); err != nil {
		return false, err
	}
	d = json.NewDecoder(b)
	if err := d.Decode(&j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}
