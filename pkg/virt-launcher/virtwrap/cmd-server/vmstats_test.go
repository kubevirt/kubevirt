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
	"fmt"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

func extractVMStatsDetails(err error) *cmdv1.VMStatsResponse {
	st, ok := grpcstatus.FromError(err)
	ExpectWithOffset(1, ok).To(BeTrue(), "expected gRPC status error")
	ExpectWithOffset(1, st.Code()).To(Equal(codes.Internal))
	details := st.Details()
	ExpectWithOffset(1, details).To(HaveLen(1))
	resp, ok := details[0].(*cmdv1.VMStatsResponse)
	ExpectWithOffset(1, ok).To(BeTrue(), "expected VMStatsResponse in error details")
	return resp
}

var _ = Describe("GetVMStats", func() {
	var (
		domainManager *virtwrap.MockDomainManager
		ctrl          *gomock.Controller
		server        *cmdserver.Launcher
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)
		server = cmdserver.NewLauncher(domainManager, cmdserver.NewServerOptions(false).WithVMStatsCollector(true))
	})

	It("should return success with empty data when nothing is requested", func() {
		request := &cmdv1.VMStatsRequest{}

		response, err := server.GetVMStats(context.TODO(), request)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Response.Success).To(BeTrue())
		Expect(response.DomainStats).To(BeNil())
		Expect(response.DirtyRateStats).To(BeNil())
		Expect(response.GuestAgentVersion).To(BeNil())
		Expect(response.GuestGetLoad).To(BeNil())
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
			Expect(response.DomainStats.Response.Success).To(BeTrue())
			Expect(response.DomainStats.DomainStats).To(ContainSubstring("test-domain"))
			Expect(response.DomainStats.DomainStats).To(ContainSubstring("test-uuid"))
		})

		It("should return error when GetDomainStats fails", func() {
			domainManager.EXPECT().GetDomainStats().Return(nil, errors.New("stats error"))

			request := &cmdv1.VMStatsRequest{DomainStats: &cmdv1.DomainStatsRequest{}}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			partialResp := extractVMStatsDetails(err)
			Expect(partialResp.Response.Success).To(BeFalse())
			Expect(partialResp.DomainStats.Response.Success).To(BeFalse())
			Expect(partialResp.DomainStats.Response.Message).To(ContainSubstring("stats error"))
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
			Expect(response.DirtyRateStats.Response.Success).To(BeTrue())
			Expect(response.DirtyRateStats.DirtyRateMbs).To(Equal(int64(42)))
		})

		It("should return error when GetDomainDirtyRateStats fails", func() {
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(nil, errors.New("dirty rate error"))

			request := &cmdv1.VMStatsRequest{DirtyRate: &cmdv1.DirtyRateRequest{}}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			partialResp := extractVMStatsDetails(err)
			Expect(partialResp.Response.Success).To(BeFalse())
			Expect(partialResp.DirtyRateStats.Response.Success).To(BeFalse())
			Expect(partialResp.DirtyRateStats.Response.Message).To(ContainSubstring("dirty rate error"))
		})
	})

	Context("AgentData", func() {
		It("should return guest agent version and agent data", func() {
			domainManager.EXPECT().GetGuestAgentVersion().Return("5.2")
			domainManager.EXPECT().GetAgentData("guest-get-load").Return("load-data", nil)
			domainManager.EXPECT().GetAgentData("guest-get-time").Return("time-data", nil)

			request := &cmdv1.VMStatsRequest{
				GuestGetLoad: &cmdv1.AgentLoadRequest{},
				GuestGetTime: &cmdv1.AgentTimeRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.GuestAgentVersion.Success).To(BeTrue())
			Expect(response.GuestAgentVersion.Message).To(Equal("5.2"))
			Expect(response.GuestGetLoad.Success).To(BeTrue())
			Expect(response.GuestGetLoad.Message).To(Equal("load-data"))
			Expect(response.GuestGetTime.Success).To(BeTrue())
			Expect(response.GuestGetTime.Message).To(Equal("time-data"))
		})

		It("should report per-command errors without failing other commands", func() {
			domainManager.EXPECT().GetGuestAgentVersion().Return("5.2")
			domainManager.EXPECT().GetAgentData("guest-get-load").Return("load-data", nil)
			domainManager.EXPECT().GetAgentData("guest-get-time").Return("", fmt.Errorf("agent not responding"))

			request := &cmdv1.VMStatsRequest{
				GuestGetLoad: &cmdv1.AgentLoadRequest{},
				GuestGetTime: &cmdv1.AgentTimeRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			partialResp := extractVMStatsDetails(err)
			Expect(partialResp.Response.Success).To(BeFalse())
			Expect(partialResp.GuestGetLoad.Success).To(BeTrue())
			Expect(partialResp.GuestGetLoad.Message).To(Equal("load-data"))
			Expect(partialResp.GuestGetTime.Success).To(BeFalse())
			Expect(partialResp.GuestGetTime.Message).To(ContainSubstring("agent not responding"))
		})

		It("should not call GetGuestAgentVersion when no agent data is requested", func() {
			request := &cmdv1.VMStatsRequest{}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.GuestAgentVersion).To(BeNil())
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
			domainManager.EXPECT().GetDomainStats().Return(expectedStats, nil)
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(expectedDirtyRate, nil)
			domainManager.EXPECT().GetGuestAgentVersion().Return("5.2")
			domainManager.EXPECT().GetAgentData("guest-get-load").Return("load-data", nil)

			request := &cmdv1.VMStatsRequest{
				DomainStats:  &cmdv1.DomainStatsRequest{},
				DirtyRate:    &cmdv1.DirtyRateRequest{},
				GuestGetLoad: &cmdv1.AgentLoadRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Response.Success).To(BeTrue())
			Expect(response.DomainStats.Response.Success).To(BeTrue())
			Expect(response.DomainStats.DomainStats).To(ContainSubstring("test-domain"))
			Expect(response.DirtyRateStats.Response.Success).To(BeTrue())
			Expect(response.DirtyRateStats.DirtyRateMbs).To(Equal(int64(100)))
			Expect(response.GuestAgentVersion.Message).To(Equal("5.2"))
			Expect(response.GuestGetLoad.Success).To(BeTrue())
			Expect(response.GuestGetLoad.Message).To(Equal("load-data"))
		})

		It("should continue filling other fields when DomainStats fails", func() {
			expectedDirtyRate := &stats.DomainStatsDirtyRate{
				MegabytesPerSecondSet: true,
				MegabytesPerSecond:    100,
			}
			domainManager.EXPECT().GetDomainStats().Return(nil, errors.New("stats error"))
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(expectedDirtyRate, nil)
			domainManager.EXPECT().GetGuestAgentVersion().Return("5.2")
			domainManager.EXPECT().GetAgentData("guest-get-load").Return("load-data", nil)

			request := &cmdv1.VMStatsRequest{
				DomainStats:  &cmdv1.DomainStatsRequest{},
				DirtyRate:    &cmdv1.DirtyRateRequest{},
				GuestGetLoad: &cmdv1.AgentLoadRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			partialResp := extractVMStatsDetails(err)
			Expect(partialResp.Response.Success).To(BeFalse())
			Expect(partialResp.Response.Message).To(ContainSubstring("stats error"))
			Expect(partialResp.DomainStats.Response.Success).To(BeFalse())
			Expect(partialResp.DirtyRateStats.Response.Success).To(BeTrue())
			Expect(partialResp.DirtyRateStats.DirtyRateMbs).To(Equal(int64(100)))
			Expect(partialResp.GuestAgentVersion.Message).To(Equal("5.2"))
			Expect(partialResp.GuestGetLoad.Success).To(BeTrue())
			Expect(partialResp.GuestGetLoad.Message).To(Equal("load-data"))
		})

		It("should continue filling other fields when DirtyRateStats fails", func() {
			expectedStats := &stats.DomainStats{
				Name: "test-domain",
			}
			domainManager.EXPECT().GetDomainStats().Return(expectedStats, nil)
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(nil, errors.New("dirty rate error"))
			domainManager.EXPECT().GetGuestAgentVersion().Return("5.2")
			domainManager.EXPECT().GetAgentData("guest-get-load").Return("load-data", nil)

			request := &cmdv1.VMStatsRequest{
				DomainStats:  &cmdv1.DomainStatsRequest{},
				DirtyRate:    &cmdv1.DirtyRateRequest{},
				GuestGetLoad: &cmdv1.AgentLoadRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			partialResp := extractVMStatsDetails(err)
			Expect(partialResp.Response.Success).To(BeFalse())
			Expect(partialResp.Response.Message).To(ContainSubstring("dirty rate error"))
			Expect(partialResp.DomainStats.Response.Success).To(BeTrue())
			Expect(partialResp.DomainStats.DomainStats).ToNot(BeEmpty())
			Expect(partialResp.DirtyRateStats.Response.Success).To(BeFalse())
			Expect(partialResp.GuestAgentVersion.Message).To(Equal("5.2"))
			Expect(partialResp.GuestGetLoad.Success).To(BeTrue())
			Expect(partialResp.GuestGetLoad.Message).To(Equal("load-data"))
		})

		It("should collect multiple errors when both DomainStats and DirtyRate fail", func() {
			domainManager.EXPECT().GetDomainStats().Return(nil, errors.New("stats error"))
			domainManager.EXPECT().GetDomainDirtyRateStats(gomock.Any()).Return(nil, errors.New("dirty rate error"))
			domainManager.EXPECT().GetGuestAgentVersion().Return("5.2")
			domainManager.EXPECT().GetAgentData("guest-get-load").Return("load-data", nil)

			request := &cmdv1.VMStatsRequest{
				DomainStats:  &cmdv1.DomainStatsRequest{},
				DirtyRate:    &cmdv1.DirtyRateRequest{},
				GuestGetLoad: &cmdv1.AgentLoadRequest{},
			}
			response, err := server.GetVMStats(context.TODO(), request)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
			partialResp := extractVMStatsDetails(err)
			Expect(partialResp.Response.Success).To(BeFalse())
			Expect(partialResp.Response.Message).To(ContainSubstring("stats error"))
			Expect(partialResp.Response.Message).To(ContainSubstring("dirty rate error"))
			Expect(partialResp.DomainStats.Response.Success).To(BeFalse())
			Expect(partialResp.DirtyRateStats.Response.Success).To(BeFalse())
			Expect(partialResp.GuestAgentVersion.Message).To(Equal("5.2"))
			Expect(partialResp.GuestGetLoad.Success).To(BeTrue())
			Expect(partialResp.GuestGetLoad.Message).To(Equal("load-data"))
		})
	})

	It("should have the same command keys as agentDataCommandTTLs", func() {
		fieldKeys := cmdserver.AgentDataCommandKeys()
		ttlKeys := virtwrap.AgentDataCommandTTLKeys()

		sort.Strings(fieldKeys)
		sort.Strings(ttlKeys)
		Expect(fieldKeys).To(Equal(ttlKeys))
	})
})
