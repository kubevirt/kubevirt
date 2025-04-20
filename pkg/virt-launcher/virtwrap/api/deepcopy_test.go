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
 */

package api

import (
	"reflect"

	fuzz "github.com/google/gofuzz"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Generated deepcopy functions", func() {

	var structs []interface{}
	ginkgo.BeforeEach(func() {

		structs = []interface{}{
			&Domain{},
			&DomainStatus{},
			&DomainList{},
			&DomainSpec{},
			&Features{},
			&FeatureHyperv{},
			&FeatureSpinlocks{},
			&FeatureVendorID{},
			&FeatureEnabled{},
			&FeatureState{},
			&Metadata{},
			&KubeVirtMetadata{},
			&GracePeriodMetadata{},
			&Commandline{},
			&Env{},
			&Resource{},
			&Memory{},
			&Devices{},
			&Disk{},
			&DiskAuth{},
			&DiskSecret{},
			&ReadOnly{},
			&DiskSource{},
			&DiskTarget{},
			&DiskDriver{},
			&DiskSourceHost{},
			&Serial{},
			&SerialTarget{},
			&Console{},
			&ConsoleTarget{},
			&Interface{},
			&LinkState{},
			&BandWidth{},
			&BootOrder{},
			&MAC{},
			&FilterRef{},
			&InterfaceSource{},
			&Model{},
			&InterfaceTarget{},
			&Alias{},
			&OS{},
			&OSType{},
			&SMBios{},
			&NVRam{},
			&Boot{},
			&BootMenu{},
			&BIOS{},
			&Loader{},
			&SysInfo{},
			&Entry{},
			&Clock{},
			&Timer{},
			&Channel{},
			&ChannelTarget{},
			&ChannelSource{},
			&Video{},
			&VideoModel{},
			&Graphics{},
			&GraphicsListen{},
			&Address{},
			&MemBalloon{},
			&Rng{},
			&RngBackend{},
			&RngRate{},
			&Watchdog{},
			&SecretUsage{},
			&SecretSpec{},
			&CPU{},
			&CPUTopology{},
			&VCPU{},
		}
	})

	ginkgo.It("should work for fuzzed structs", func() {
		for _, s := range structs {
			fuzz.New().NilChance(0).Fuzz(s)
			Expect(reflect.ValueOf(s).MethodByName("DeepCopy").Call(nil)[0].Interface()).To(Equal(s))
			if reflect.ValueOf(s).MethodByName("DeepCopyObject").IsValid() {
				Expect(reflect.ValueOf(s).MethodByName("DeepCopyObject").Call(nil)[0].Interface()).To(Equal(s))
			}
			new := reflect.New(reflect.TypeOf(s).Elem())
			reflect.ValueOf(s).MethodByName("DeepCopyInto").Call([]reflect.Value{new})
			Expect(new.Interface()).To(Equal(s))
		}

	})
})
