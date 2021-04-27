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

package cmdserver

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("Virt remote commands", func() {
	var domainManager *virtwrap.MockDomainManager
	var client cmdclient.LauncherClient

	var ctrl *gomock.Controller

	var err error
	var shareDir string
	var stop chan struct{}
	var stopped bool
	var useEmulation bool
	var options *ServerOptions

	BeforeEach(func() {
		stop = make(chan struct{})
		stopped = false
		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)

		socketPath := filepath.Join(shareDir, "server.sock")

		useEmulation = true
		options = NewServerOptions(useEmulation)
		RunServer(socketPath, domainManager, stop, options)
		client, err = cmdclient.NewClient(socketPath)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if stopped == false {
			close(stop)
		}
		ctrl.Finish()
		client.Close()
		os.RemoveAll(shareDir)
	})

	Context("server", func() {
		It("should start a vmi", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domain := api.NewMinimalDomain("testvmi")
			domainManager.EXPECT().SyncVMI(vmi, useEmulation, &cmdv1.VirtualMachineOptions{}).Return(&domain.Spec, nil)

			err := client.SyncVirtualMachine(vmi, &cmdv1.VirtualMachineOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should kill a vmi", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domainManager.EXPECT().KillVMI(vmi)

			err := client.KillVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should shutdown a vmi", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domainManager.EXPECT().SignalShutdownVMI(vmi)
			err := client.ShutdownVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should pause a vmi", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domainManager.EXPECT().PauseVMI(vmi)
			err := client.PauseVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should unpause a vmi", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domainManager.EXPECT().UnpauseVMI(vmi)
			err := client.UnpauseVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should list domains", func() {
			var list []*api.Domain
			list = append(list, api.NewMinimalDomain("testvmi1"))

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domain, exists, err := client.GetDomain()
			Expect(err).ToNot(HaveOccurred())

			Expect(exists).To(BeTrue())
			Expect(domain).ToNot(Equal(nil))
			Expect(domain.ObjectMeta.Name).To(Equal("testvmi1"))
		})

		It("should list no domain if no domain is there yet", func() {
			var list []*api.Domain

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domain, exists, err := client.GetDomain()
			Expect(err).ToNot(HaveOccurred())

			Expect(exists).To(BeFalse())
			Expect(domain).ToNot(Equal(nil))
		})

		It("client should return disconnected after server stops", func() {
			err := client.Ping()
			Expect(err).ToNot(HaveOccurred())

			close(stop)
			stopped = true
			time.Sleep(time.Second)

			client.Close()

			err = client.Ping()
			Expect(err).To(HaveOccurred())
			Expect(cmdclient.IsDisconnected(err)).To(BeTrue())

			_, err = cmdclient.NewClient(shareDir + "/server.sock")
			Expect(err).To(HaveOccurred())
		})

		It("should return domain stats", func() {
			var list []*stats.DomainStats
			dom := api.NewMinimalDomain("testvmstats1")
			list = append(list, &stats.DomainStats{
				Name: dom.Spec.Name,
				UUID: dom.Spec.UUID,
			})

			domainManager.EXPECT().GetDomainStats().Return(list, nil)
			domStats, exists, err := client.GetDomainStats()
			Expect(err).ToNot(HaveOccurred())

			Expect(exists).To(BeTrue())
			Expect(domStats).ToNot(Equal(nil))
			Expect(domStats.Name).To(Equal(list[0].Name))
			Expect(domStats.UUID).To(Equal(list[0].UUID))
		})

		It("should return full user list", func() {
			userList := []v1.VirtualMachineInstanceGuestOSUser{
				{
					UserName: "testUser",
				},
			}

			domainManager.EXPECT().GetUsers().Return(userList, nil)

			fetchedList, err := client.GetUsers()
			Expect(err).ToNot(HaveOccurred(), "should fetch users without any issue")
			Expect(fetchedList.Items).To(Equal(userList), "fetched list should be the same")
		})

		It("should return full filesystem list", func() {
			fsList := []v1.VirtualMachineInstanceFileSystem{
				{
					DiskName:       "main",
					MountPoint:     "/",
					FileSystemType: "EXT4",
					UsedBytes:      3333,
					TotalBytes:     9999,
				},
			}

			domainManager.EXPECT().GetFilesystems().Return(fsList, nil)

			fetchedList, err := client.GetFilesystems()
			Expect(err).ToNot(HaveOccurred(), "should fetch filesystems without any issue")
			Expect(fetchedList.Items).To(Equal(fsList), "fetched list should be the same")
		})

		It("should finalize VM migration", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domainManager.EXPECT().FinalizeVirtualMachineMigration(vmi).Return(nil)

			Expect(client.FinalizeVirtualMachineMigration(vmi)).Should(Succeed())
		})

		It("should fail to finalize VM migration", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domainManager.EXPECT().FinalizeVirtualMachineMigration(vmi).Return(errors.New("error"))

			Expect(client.FinalizeVirtualMachineMigration(vmi)).ToNot(Succeed())
		})
	})

	Describe("Version mismatch", func() {

		var err error
		var ctrl *gomock.Controller
		var infoClient *info.MockCmdInfoClient

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			infoClient = info.NewMockCmdInfoClient(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("Should report error when server version mismatches", func() {

			fakeResponse := info.CmdInfoResponse{
				SupportedCmdVersions: []uint32{42},
			}
			infoClient.EXPECT().Info(gomock.Any(), gomock.Any()).Return(&fakeResponse, nil)

			By("Initializing the notifier")
			_, err = cmdclient.NewClientWithInfoClient(infoClient, nil)

			Expect(err).To(HaveOccurred(), "Should have returned error about incompatible versions")
			Expect(err.Error()).To(ContainSubstring("no compatible version found"), "Expected error message to contain 'no compatible version found'")

		})
	})

})
