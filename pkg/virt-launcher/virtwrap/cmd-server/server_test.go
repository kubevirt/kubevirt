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
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
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

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		stop = make(chan struct{})
		stopped = false
		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)

		useEmulation = true
		options = NewServerOptions(useEmulation)
		RunServer(shareDir+"/server.sock", domainManager, stop, options)
		client, err = cmdclient.GetClient(shareDir + "/server.sock")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if stopped == false {
			close(stop)
		}
		ctrl.Finish()
		os.RemoveAll(shareDir)
	})

	Context("server", func() {
		It("should start a vmi", func() {
			vmi := v1.NewVMIReferenceFromName("testvmi")
			domain := api.NewMinimalDomain("testvmi")
			domainManager.EXPECT().SyncVMI(vmi, useEmulation).Return(&domain.Spec, nil)

			err := client.SyncVirtualMachine(vmi)
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

		It("should list domains", func() {
			var list []*api.Domain
			list = append(list, api.NewMinimalDomain("testvmi1"))

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			domain, exists, err := client.GetDomain()
			Expect(err).ToNot(HaveOccurred())

			Expect(exists).To(Equal(true))
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
			Expect(cmdclient.IsDisconnected(err)).To(Equal(true))

			_, err = cmdclient.GetClient(shareDir + "/server.sock")
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

			Expect(exists).To(Equal(true))
			Expect(domStats).ToNot(Equal(nil))
		})
	})
})
