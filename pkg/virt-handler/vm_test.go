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

package virthandler

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/config-disk"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

var _ = Describe("VM", func() {
	var domainManager *virtwrap.MockDomainManager
	var vmInterface *kubecli.MockVMInterface
	var virtClient *kubecli.MockKubevirtClient

	var ctrl *gomock.Controller
	var controller *VirtualMachineController
	var vmSource *framework.FakeControllerSource
	var vmInformer cache.SharedIndexInformer
	var domainSource *framework.FakeControllerSource
	var domainInformer cache.SharedIndexInformer
	var watchdogSource *framework.FakeControllerSource
	var watchdogInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue

	var vmFeeder *testutils.VirtualMachineFeeder
	var domainFeeder *testutils.DomainFeeder

	var recorder record.EventRecorder

	var shareDir string
	var stop chan struct{}

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		stop = make(chan struct{})
		shareDir, err := ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		host := "master"

		Expect(err).ToNot(HaveOccurred())

		vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		domainInformer, domainSource = testutils.NewFakeInformerFor(&api.Domain{})
		watchdogInformer, watchdogSource = testutils.NewFakeInformerFor(&api.Domain{})
		recorder = record.NewFakeRecorder(100)

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)
		domainManager = virtwrap.NewMockDomainManager(ctrl)
		virtClient.EXPECT().VM(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

		configDiskClient := configdisk.NewConfigDiskClient(virtClient)

		controller = NewController(domainManager, recorder, virtClient, host, configDiskClient, shareDir, 1, vmInformer, domainInformer, watchdogInformer)
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)
		domainFeeder = testutils.NewDomainFeeder(mockQueue, domainSource)

		go vmInformer.Run(stop)
		go domainInformer.Run(stop)
		go watchdogInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, domainInformer.HasSynced, watchdogInformer.HasSynced)).To(BeTrue())
	})

	Context("VM controller gets informed about a Domain change through the Domain controller", func() {

		It("should kill the Domain if no cluster wide equivalent exists", func() {
			domain := api.NewMinimalDomain("testvm")
			domainFeeder.Add(domain)
			vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))

			domainManager.EXPECT().RemoveVMSecrets(v1.NewVMReferenceFromName("testvm")).Return(nil)
			domainManager.EXPECT().KillVM(v1.NewVMReferenceFromName("testvm"))

			controller.Execute()
		})

		It("should leave the Domain alone if the VM is migrating to its host", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.MigrationNodeName = "master"

			vmInterface.EXPECT().Get("testvm", gomock.Any()).Return(vm, nil)
			mockQueue.Add("default/testvm")
			controller.Execute()
		})

		It("should re-enqueue if the Key is unparseable", func() {
			Expect(mockQueue.Len()).Should(Equal(0))
			mockQueue.Add("a/b/c/d/e")
			controller.Execute()
			Expect(mockQueue.NumRequeues("a/b/c/d/e")).To(Equal(1))
		})

		table.DescribeTable("should leave the VM alone if it is in the final phase", func(phase v1.VMPhase) {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = phase
			vmFeeder.Add(vm)
			controller.Execute()
			// expect no errors and no mock interactions
			Expect(mockQueue.NumRequeues("default/testvm")).To(Equal(0))
		},
			table.Entry("succeeded", v1.Succeeded),
			table.Entry("failed", v1.Failed),
		)
	})

	AfterEach(func() {
		close(stop)
		ctrl.Finish()
		os.RemoveAll(shareDir)
	})

	var _ = Describe("PVC", func() {
		RegisterFailHandler(Fail)

		log.Log.SetIOWriter(GinkgoWriter)

		var (
			expectedPVC k8sv1.PersistentVolumeClaim
			expectedPV  k8sv1.PersistentVolume
			server      *ghttp.Server
		)

		BeforeEach(func() {
			expectedPVC = k8sv1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1",
				},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					VolumeName: "disk-01",
				},
				Status: k8sv1.PersistentVolumeClaimStatus{
					Phase: k8sv1.ClaimBound,
				},
			}

			source := k8sv1.ISCSIVolumeSource{
				IQN:          "iqn.2009-02.com.test:for.all",
				Lun:          1,
				TargetPortal: "127.0.0.1:6543",
			}

			expectedPV = k8sv1.PersistentVolume{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolume",
					APIVersion: "v1",
				},
				Spec: k8sv1.PersistentVolumeSpec{
					PersistentVolumeSource: k8sv1.PersistentVolumeSource{
						ISCSI: &source,
					},
				},
			}

			server = ghttp.NewServer()
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/persistentvolumeclaims/test-claim"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedPVC),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/persistentvolumes/disk-01"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedPV),
				),
			)
		})

		AfterEach(func() {
			server.Close()
		})

		Context("Map Source Disks", func() {
			It("looks up and applies PVC", func() {
				vm := v1.VirtualMachine{}

				disk := v1.Disk{
					Type: "PersistentVolumeClaim",
					Source: v1.DiskSource{
						Name: "test-claim",
					},
					Target: v1.DiskTarget{
						Device: "vda",
					},
				}
				disk.Type = "PersistentVolumeClaim"

				domain := v1.DomainSpec{}
				domain.Devices.Disks = []v1.Disk{disk}
				vm.Spec.Domain = &domain

				restClient := getRestClient(server.URL())
				vmCopy, err := MapPersistentVolumes(&vm, restClient, k8sv1.NamespaceDefault)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(vmCopy.Spec.Domain.Devices.Disks)).To(Equal(1))
				newDisk := vmCopy.Spec.Domain.Devices.Disks[0]
				Expect(newDisk.Type).To(Equal("network"))
				Expect(newDisk.Driver.Type).To(Equal("raw"))
				Expect(newDisk.Driver.Name).To(Equal("qemu"))
				Expect(newDisk.Device).To(Equal("disk"))
				Expect(newDisk.Source.Protocol).To(Equal("iscsi"))
				Expect(newDisk.Source.Name).To(Equal("iqn.2009-02.com.test:for.all/1"))
			})
			It("should fail on unsupported PV disk types", func() {
				expectedPV.Spec.ISCSI = nil
				expectedPV.Spec.CephFS = &k8sv1.CephFSPersistentVolumeSource{}
				disk := v1.Disk{
					Type: "PersistentVolumeClaim",
					Source: v1.DiskSource{
						Name: "test-claim",
					},
					Target: v1.DiskTarget{
						Device: "vda",
					},
				}
				disk.Type = "PersistentVolumeClaim"
				_, err := mapPVToDisk(&disk, &expectedPV)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported"))
			})
		})
	})
})

func getRestClient(url string) *rest.RESTClient {
	gv := schema.GroupVersion{Group: "", Version: "v1"}
	restConfig, err := clientcmd.BuildConfigFromFlags(url, "")
	Expect(err).NotTo(HaveOccurred())
	restConfig.GroupVersion = &gv
	restConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	restConfig.APIPath = "/api"
	restConfig.ContentType = runtime.ContentTypeJSON
	restClient, err := rest.RESTClientFor(restConfig)
	Expect(err).NotTo(HaveOccurred())
	return restClient
}
