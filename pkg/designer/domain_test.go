/*
 * This file is part of the kubevirt project
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

package designer_test

import (
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	_ "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/designer"
	"kubevirt.io/kubevirt/pkg/logging"

	/* XXX without this we get unregistered PVC type.
	 * THis must be triggering some other init method.
	 * Figure it out & import it directly */
	_ "kubevirt.io/kubevirt/pkg/virt-handler"
)

var _ = Describe("DomainMap", func() {
	RegisterFailHandler(Fail)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

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
			vm := v1.VM{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: k8sv1.NamespaceDefault,
				},
			}

			disk1 := v1.Disk{
				Type:   "PersistentVolumeClaim",
				Device: "disk",
				Source: v1.DiskSource{
					Name: "test-claim",
				},
				Target: v1.DiskTarget{
					Device: "vda",
				},
			}

			disk2 := v1.Disk{
				Type:   "network",
				Device: "cdrom",
				Source: v1.DiskSource{
					Name:     "iqn.2009-02.com.test:for.me/1",
					Protocol: "iscsi",
					Host: &v1.DiskSourceHost{
						Name: "127.0.0.2",
						Port: "3260",
					},
				},
				Target: v1.DiskTarget{
					Device: "hda",
				},
			}

			domain := v1.DomainSpec{}
			domain.Devices.Disks = []v1.Disk{disk1, disk2}
			vm.Spec.Domain = &domain

			restClient := getRestClient(server.URL())
			domSpec, err := designer.MapDomainSpec(&vm, restClient)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(domSpec.Devices.Disks)).To(Equal(2))
			newDisk := domSpec.Devices.Disks[0]
			Expect(newDisk.Type).To(Equal("network"))
			Expect(newDisk.Driver.Type).To(Equal("raw"))
			Expect(newDisk.Driver.Name).To(Equal("qemu"))
			Expect(newDisk.Device).To(Equal("disk"))
			Expect(newDisk.Source.Protocol).To(Equal("iscsi"))
			Expect(newDisk.Source.Name).To(Equal("iqn.2009-02.com.test:for.all/1"))
			Expect(newDisk.Source.Host.Name).To(Equal("127.0.0.1"))
			Expect(newDisk.Source.Host.Port).To(Equal("6543"))

			newDisk = domSpec.Devices.Disks[1]
			Expect(newDisk.Type).To(Equal("network"))
			Expect(newDisk.Device).To(Equal("cdrom"))
			Expect(newDisk.Source.Protocol).To(Equal("iscsi"))
			Expect(newDisk.Source.Name).To(Equal("iqn.2009-02.com.test:for.me/1"))
			Expect(newDisk.Source.Host.Name).To(Equal("127.0.0.2"))
			Expect(newDisk.Source.Host.Port).To(Equal("3260"))
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

func TestVMs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DomainMap")
}
