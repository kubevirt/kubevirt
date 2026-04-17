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

package cmdserver_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("GetVMStats", func() {
	var (
		domainManager *virtwrap.MockDomainManager
		ctrl          *gomock.Controller
		server        *cmdserver.Launcher
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)
		server = cmdserver.NewLauncher(domainManager, false)
	})

	It("should return success with empty data when nothing is requested", func() {
		request := &cmdv1.VMStatsRequest{}

		response, err := server.GetVMStats(context.TODO(), request)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Response.Success).To(BeTrue())
		Expect(response.DomainStats).To(BeEmpty())
		Expect(response.DirtyRateStats).To(BeEmpty())
		Expect(response.AgentData).To(BeNil())
		Expect(response.GuestAgentVersion).To(BeEmpty())
	})

	Context("DomainStats", func() {
		It("should return domain stats when requested", func() {
			expectedStats := &stats.DomainStats{
				Name: "test-domain",
				UUID: "test-uuid",
			}
			domainManager.EXPECT().GetDomainStats().Return(expectedStats, nil)

			request := &cmdv1.VMStatsRequest{DomainStats: &cmdv1.DomainStatsRequest{}}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.DomainStats).To(ContainSubstring("test-domain"))
			Expect(response.DomainStats).To(ContainSubstring("test-uuid"))
		})

		It("should return error when GetDomainStats fails", func() {
			domainManager.EXPECT().GetDomainStats().Return(nil, errors.New("stats error"))

			request := &cmdv1.VMStatsRequest{DomainStats: &cmdv1.DomainStatsRequest{}}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeFalse())
			Expect(response.Response.Message).To(ContainSubstring("stats error"))
		})
	})

	Context("DirtyRateStats", func() {
		It("should return dirty rate stats when requested", func() {
			expectedDirtyRate := &stats.DomainStatsDirtyRate{
				MegabytesPerSecondSet: true,
				MegabytesPerSecond:    42,
			}
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(expectedDirtyRate, nil)

			request := &cmdv1.VMStatsRequest{DirtyRate: &cmdv1.DirtyRateRequest{}}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.DirtyRateStats).To(ContainSubstring("42"))
		})

		It("should return error when GetDomainDirtyRateStats fails", func() {
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(nil, errors.New("dirty rate error"))

			request := &cmdv1.VMStatsRequest{DirtyRate: &cmdv1.DirtyRateRequest{}}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeFalse())
			Expect(response.Response.Message).To(ContainSubstring("dirty rate error"))
		})
	})

	Context("AgentData", func() {
		It("should return guest agent version and agent data", func() {
			guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{
				GAVersion: "5.2",
			}
			domainManager.EXPECT().GetGuestInfo().Return(guestInfo)
			domainManager.EXPECT().GetAgentData("key1").Return("value1", true)
			domainManager.EXPECT().GetAgentData("key2").Return("value2", true)

			request := &cmdv1.VMStatsRequest{
				AgentData: []*cmdv1.AgentDataRequest{{CommandKey: "key1"}, {CommandKey: "key2"}},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.GuestAgentVersion).To(Equal("5.2"))
			Expect(response.AgentData).To(Equal(map[string]string{"key1": "value1", "key2": "value2"}))
		})

		It("should omit keys not found in the agent store", func() {
			guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{
				GAVersion: "5.2",
			}
			domainManager.EXPECT().GetGuestInfo().Return(guestInfo)
			domainManager.EXPECT().GetAgentData("known-key").Return("some-data", true)
			domainManager.EXPECT().GetAgentData("unknown-key").Return("", false)

			request := &cmdv1.VMStatsRequest{
				AgentData: []*cmdv1.AgentDataRequest{{CommandKey: "known-key"}, {CommandKey: "unknown-key"}},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.AgentData).To(HaveKeyWithValue("known-key", "some-data"))
			Expect(response.AgentData).ToNot(HaveKey("unknown-key"))
		})

		It("should include keys with empty string values", func() {
			guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{
				GAVersion: "5.2",
			}
			domainManager.EXPECT().GetGuestInfo().Return(guestInfo)
			domainManager.EXPECT().GetAgentData("empty-key").Return("", true)

			request := &cmdv1.VMStatsRequest{
				AgentData: []*cmdv1.AgentDataRequest{{CommandKey: "empty-key"}},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.AgentData).To(HaveKeyWithValue("empty-key", ""))
		})

		It("should not call GetGuestInfo when agent data list is empty", func() {
			request := &cmdv1.VMStatsRequest{
				AgentData: []*cmdv1.AgentDataRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.GuestAgentVersion).To(BeEmpty())
			Expect(response.AgentData).To(BeNil())
		})
	})

	Context("combined request", func() {
		It("should return all requested data", func() {
			expectedStats := &stats.DomainStats{
				Name: "test-domain",
				UUID: "test-uuid",
			}
			expectedDirtyRate := &stats.DomainStatsDirtyRate{
				MegabytesPerSecondSet: true,
				MegabytesPerSecond:    100,
			}
			guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{
				GAVersion: "5.2",
			}

			domainManager.EXPECT().GetDomainStats().Return(expectedStats, nil)
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(expectedDirtyRate, nil)
			domainManager.EXPECT().GetGuestInfo().Return(guestInfo)
			domainManager.EXPECT().GetAgentData("key1").Return("value1", true)

			request := &cmdv1.VMStatsRequest{
				DomainStats: &cmdv1.DomainStatsRequest{},
				DirtyRate:   &cmdv1.DirtyRateRequest{},
				AgentData:   []*cmdv1.AgentDataRequest{{CommandKey: "key1"}},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.DomainStats).To(ContainSubstring("test-domain"))
			Expect(response.DirtyRateStats).To(ContainSubstring("100"))
			Expect(response.GuestAgentVersion).To(Equal("5.2"))
			Expect(response.AgentData).To(Equal(map[string]string{"key1": "value1"}))
		})

		It("should stop early when DomainStats fails", func() {
			domainManager.EXPECT().GetDomainStats().Return(nil, errors.New("stats error"))

			request := &cmdv1.VMStatsRequest{
				DomainStats: &cmdv1.DomainStatsRequest{},
				DirtyRate:   &cmdv1.DirtyRateRequest{},
				AgentData:   []*cmdv1.AgentDataRequest{{CommandKey: "key1"}},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeFalse())
			Expect(response.Response.Message).To(ContainSubstring("stats error"))
			Expect(response.DirtyRateStats).To(BeEmpty())
			Expect(response.AgentData).To(BeNil())
		})

		It("should stop early when DirtyRateStats fails after DomainStats succeeds", func() {
			expectedStats := &stats.DomainStats{
				Name: "test-domain",
			}
			domainManager.EXPECT().GetDomainStats().Return(expectedStats, nil)
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(nil, errors.New("dirty rate error"))

			request := &cmdv1.VMStatsRequest{
				DomainStats: &cmdv1.DomainStatsRequest{},
				DirtyRate:   &cmdv1.DirtyRateRequest{},
				AgentData:   []*cmdv1.AgentDataRequest{{CommandKey: "key1"}},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeFalse())
			Expect(response.Response.Message).To(ContainSubstring("dirty rate error"))
			Expect(response.DomainStats).ToNot(BeEmpty())
			Expect(response.AgentData).To(BeNil())
		})
	})
})
