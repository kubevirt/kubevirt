// freezer_test.go
package main

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Freezer", func() {
	var (
		client    *cmdclient.MockLauncherClient
		config    *FreezerConfig
		guestInfo *v1.VirtualMachineInstanceGuestAgentInfo
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		client = cmdclient.NewMockLauncherClient(ctrl)
		config = &FreezerConfig{
			Name:      "test-vmi",
			Namespace: "default",
		}
		guestInfo = &v1.VirtualMachineInstanceGuestAgentInfo{
			GAVersion: "1.0",
		}
	})

	Describe("shouldFreezeVirtualMachine", func() {
		It("should return true if domain exists and is running", func() {
			client.EXPECT().GetDomain().Return(&api.Domain{
				Status: api.DomainStatus{
					Status: api.Running,
				},
			}, true, nil)

			result, err := shouldFreezeVirtualMachine(client)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("should return false if domain exists and is not running", func() {
			client.EXPECT().GetDomain().Return(&api.Domain{
				Status: api.DomainStatus{
					Status: api.Paused,
				},
			}, true, nil)

			result, err := shouldFreezeVirtualMachine(client)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

		It("should return false if domain does not exist", func() {
			client.EXPECT().GetDomain().Return(nil, false, nil)

			result, err := shouldFreezeVirtualMachine(client)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeFalse())
		})

		It("should return error if GetDomain fails", func() {
			client.EXPECT().GetDomain().Return(nil, false, errors.New("Error getting domain"))

			result, err := shouldFreezeVirtualMachine(client)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeFalse())
		})
	})

	Context("Freeze", func() {
		BeforeEach(func() {
			config.Freeze = true
		})
		It("should return error if get guest agent fails", func() {
			client.EXPECT().GetGuestInfo().Return(nil, errors.New("guest info error"))

			err := run(config, client)
			Expect(err).To(HaveOccurred())
		})

		It("returns nil and skip freeze if guest agent version is empty", func() {
			client.EXPECT().GetGuestInfo().Return(&v1.VirtualMachineInstanceGuestAgentInfo{}, nil)

			err := run(config, client)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns nil and skip freeze if vm domain not running", func() {
			client.EXPECT().GetGuestInfo().Return(guestInfo, nil)
			client.EXPECT().GetDomain().Return(&api.Domain{
				Status: api.DomainStatus{
					Status: api.Paused,
				},
			}, true, nil)

			err := run(config, client)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should succeed if Freeze VirtualMachine", func() {
			client.EXPECT().GetGuestInfo().Return(guestInfo, nil)
			client.EXPECT().GetDomain().Return(&api.Domain{Status: api.DomainStatus{Status: api.Running}}, true, nil)
			client.EXPECT().FreezeVirtualMachine(gomock.Any(), gomock.Any()).Return(nil)

			err := run(config, client)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if FreezeVirtualMachine fails", func() {
			client.EXPECT().GetGuestInfo().Return(guestInfo, nil)
			client.EXPECT().GetDomain().Return(&api.Domain{Status: api.DomainStatus{Status: api.Running}}, true, nil)
			client.EXPECT().FreezeVirtualMachine(gomock.Any(), gomock.Any()).Return(errors.New("freeze failed"))

			err := run(config, client)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("UnFreeze", func() {
		BeforeEach(func() {
			config.Unfreeze = true
		})
		It("should succeed if Unfreeze VirtualMachine", func() {
			client.EXPECT().GetGuestInfo().Return(guestInfo, nil)
			client.EXPECT().GetDomain().Return(&api.Domain{Status: api.DomainStatus{Status: api.Running}}, true, nil)
			client.EXPECT().UnfreezeVirtualMachine(gomock.Any()).Return(nil)

			err := run(config, client)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if UnfreezeVirtualMachine fails", func() {
			client.EXPECT().GetGuestInfo().Return(guestInfo, nil)
			client.EXPECT().GetDomain().Return(&api.Domain{Status: api.DomainStatus{Status: api.Running}}, true, nil)
			client.EXPECT().UnfreezeVirtualMachine(gomock.Any()).Return(errors.New("unfreeze failed"))

			err := run(config, client)
			Expect(err).To(HaveOccurred())
		})
	})
})
