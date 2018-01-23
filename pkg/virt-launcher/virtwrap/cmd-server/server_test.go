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

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server/client"
)

var _ = Describe("Virt remote commands", func() {
	var domainManager *virtwrap.MockDomainManager
	var client cmdclient.LauncherClient

	var ctrl *gomock.Controller

	var err error
	var shareDir string
	var stop chan struct{}
	var stopped bool

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		stop = make(chan struct{})
		stopped = false
		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)

		RunServer(shareDir+"/server.sock", domainManager, stop)
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
		It("should start a vm", func() {
			vm := v1.NewVMReferenceFromName("testvm")
			domain := api.NewMinimalDomain("testvm")
			domainManager.EXPECT().SyncVM(vm, gomock.Any()).Return(&domain.Spec, nil)

			var secrets map[string]*k8sv1.Secret
			err := client.StartVirtualMachine(vm, secrets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should kill a vm", func() {
			vm := v1.NewVMReferenceFromName("testvm")
			domainManager.EXPECT().KillVM(vm)

			err := client.KillVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should shutdown a vm", func() {
			vm := v1.NewVMReferenceFromName("testvm")
			domainManager.EXPECT().SignalShutdownVM(vm)
			err := client.ShutdownVirtualMachine(vm)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should Sync Secrets", func() {
			vm := v1.NewVMReferenceFromName("testvm")
			usage := "fakeusage"
			usageId := "fakeusageid"
			secretValue := "fakesecretval"
			domainManager.EXPECT().SyncVMSecret(vm, usage, usageId, secretValue)
			err := client.SyncSecret(vm, usage, usageId, secretValue)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should list domains", func() {
			var list []*api.Domain

			list = append(list, api.NewMinimalDomain("testvm1"))
			list = append(list, api.NewMinimalDomain("testvm2"))
			list = append(list, api.NewMinimalDomain("testvm3"))

			domainManager.EXPECT().ListAllDomains().Return(list, nil)
			returnList, err := client.ListDomains()
			Expect(err).ToNot(HaveOccurred())

			Expect(len(returnList)).To(Equal(3))
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
	})
})
