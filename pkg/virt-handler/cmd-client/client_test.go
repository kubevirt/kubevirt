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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package cmdclient

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

var _ = Describe("Virt remote commands", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	var err error
	var shareDir string
	var socketsDir string
	var podsDir string
	var vmi *v1.VirtualMachineInstance
	var podSocketFile string

	host := "myhost"
	podUID := "poduid123"

	BeforeEach(func() {
		vmi = v1.NewMinimalVMI("testvmi")
		vmi.UID = types.UID("1234")
		vmi.Status = v1.VirtualMachineInstanceStatus{
			ActivePods: map[types.UID]string{
				types.UID(podUID): host,
			},
		}

		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		podsDir, err = ioutil.TempDir("", "pods")
		Expect(err).ToNot(HaveOccurred())

		socketsDir = filepath.Join(shareDir, "sockets")
		os.Mkdir(socketsDir, 0755)

		SetLegacyBaseDir(shareDir)
		SetPodsBaseDir(podsDir)

		podSocketFile = SocketFilePathOnHost(podUID)

		os.MkdirAll(filepath.Dir(podSocketFile), 0755)
		f, err := os.Create(podSocketFile)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

	})

	AfterEach(func() {
		os.RemoveAll(shareDir)
		os.RemoveAll(podsDir)
	})

	Context("client", func() {
		It("socket from UID", func() {
			sock, err := FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(sock).To(Equal(podSocketFile))

			sock = SocketOnGuest()
			Expect(sock).To(Equal(filepath.Join(shareDir, "sockets", StandardLauncherSocketFileName)))

			// falls back to returning a legacy socket name if it exists
			// This is for backwards compatiblity
			f, err := os.Create(filepath.Join(socketsDir, "1234_sock"))
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			sock, err = FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(sock).To(Equal(filepath.Join(shareDir, "sockets", "1234_sock")))

		})

		It("Find UID from namespace/name using socket info", func() {
			name := vmi.Name
			namespace := vmi.Namespace
			uid := string(vmi.UID)

			sock, err := FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())
			err = SetSocketInfo(sock, uid, name, namespace)
			Expect(err).ToNot(HaveOccurred())

			foundUid, err := FindLastKnownUIDForKey(name, namespace)
			Expect(err).ToNot(HaveOccurred())

			Expect(foundUid).To(Equal(uid))
		})

		It("Find UID from namespace/name using legacy watchdog file", func() {
			name := "testvmi"
			namespace := "default"
			uid := "1234"

			watchdogFile := filepath.Join(shareDir, "watchdog-files", namespace+"_"+name)
			os.MkdirAll(filepath.Join(shareDir, "watchdog-files"), 0755)

			sock := filepath.Join(shareDir, "sockets", uid+"_sock")
			os.MkdirAll(filepath.Dir(sock), 0755)
			f, err := os.Create(sock)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			f, err = os.Create(watchdogFile)
			Expect(err).ToNot(HaveOccurred())
			_, err = f.WriteString(uid)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			foundUid, err := FindLastKnownUIDForKey(name, namespace)
			Expect(err).ToNot(HaveOccurred())

			Expect(foundUid).To(Equal(uid))
		})

		It("Detect unresponsive socket", func() {
			sock, err := FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())

			os.RemoveAll(sock)

			// unresponsive is true when no socket file exists
			unresponsive := IsSocketUnresponsive(sock)
			Expect(unresponsive).To(BeTrue())

			// unresponsive is false when socket file exists
			os.Mkdir(filepath.Dir(sock), 0755)
			f, err := os.Create(sock)
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			unresponsive = IsSocketUnresponsive(sock)
			Expect(unresponsive).To(BeFalse())

			// unresponsive is true when marked as unresponsive
			MarkSocketUnresponsive(sock)
			unresponsive = IsSocketUnresponsive(sock)
			Expect(unresponsive).To(BeTrue())
		})

		It("Set and Get Socket Info Metadata", func() {
			sock, err := FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())

			err = SetSocketInfo(sock, "1234", "myname", "mynamespace")
			Expect(err).ToNot(HaveOccurred())

			info, err := GetSocketInfo(sock)
			Expect(err).ToNot(HaveOccurred())

			Expect(info.UID).To(Equal("1234"))
			Expect(info.Name).To(Equal("myname"))
			Expect(info.Namespace).To(Equal("mynamespace"))

		})
	})
})
