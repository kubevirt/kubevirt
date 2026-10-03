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

package launcher_clients

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var _ = Describe("startDomainNotifyPipe", func() {

	const (
		shareDirName = "share"
		settling     = 200 * time.Millisecond
		timeout      = 5 * time.Second
	)

	var (
		manager    *launcherClientsManager
		detector   *isolation.MockPodIsolationDetector
		isoResult  *isolation.MockIsolationResult
		safeRoot   *safepath.Path
		vmi        *v1.VirtualMachineInstance
		stopChan   chan struct{}
		socketPath string
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())

		// podRoot stands in for the launcher pod's mount root
		podRoot := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(podRoot, shareDirName), 0777)).To(Succeed())
		socketPath = filepath.Join(podRoot, shareDirName, "domain-notify-pipe.sock")

		var err error
		safeRoot, err = safepath.JoinAndResolveWithRelativeRoot(podRoot)
		Expect(err).To(Succeed())

		isoResult = isolation.NewMockIsolationResult(ctrl)
		detector = isolation.NewMockPodIsolationDetector(ctrl)

		manager = &launcherClientsManager{
			virtShareDir:         shareDirName,
			launcherClients:      virtcache.LauncherClientInfoByVMI{},
			podIsolationDetector: detector,
		}

		vmi = api.NewMinimalVMI("testvmi")
		vmi.UID = "1234"

		stopChan = make(chan struct{})
	})

	// expectPipeInjection programs the detector to report an isolation result
	// rooted at the stand-in pod mount root.
	expectPipeInjection := func() {
		detector.EXPECT().Detect(vmi).Return(isoResult, nil)
		isoResult.EXPECT().MountRoot().Return(safeRoot, nil)
	}

	Context("failures", func() {

		It("should wrap the error when isolation detection fails", func() {
			detectErr := errors.New("no such pod")
			detector.EXPECT().Detect(vmi).Return(nil, detectErr)

			err := manager.startDomainNotifyPipe(stopChan, vmi)

			Expect(err).To(MatchError(ContainSubstring("failed to detect isolation for launcher pod when setting up notify pipe")))
			Expect(err).To(MatchError(ContainSubstring(detectErr.Error())))
			Expect(socketPath).ToNot(BeAnExistingFile())
		})

		It("should return the injection error unchanged when the mount root is unavailable", func() {
			mountErr := errors.New("mount root unavailable")
			detector.EXPECT().Detect(vmi).Return(isoResult, nil)
			isoResult.EXPECT().MountRoot().Return(nil, mountErr)

			err := manager.startDomainNotifyPipe(stopChan, vmi)

			Expect(err).To(MatchError(mountErr))
			Expect(socketPath).ToNot(BeAnExistingFile())
		})
	})

	Context("success", func() {

		It("should create the pipe socket in the pod share directory", func() {
			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
			Expect(socketPath).To(BeAnExistingFile())
		})

		It("should accept connections on the pipe socket", func() {
			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())

			conn, err := net.Dial("unix", socketPath)
			Expect(err).To(Succeed())
			Expect(conn.Close()).To(Succeed())
		})
	})

	Context("socket ownership", func() {

		// mockOwnership swaps in a mock ownership manager for the duration of
		// the spec and returns it.
		mockOwnership := func() *diskutils.MockOwnershipManagerInterface {
			original := diskutils.DefaultOwnershipManager
			DeferCleanup(func() {
				diskutils.DefaultOwnershipManager = original
			})
			manager := diskutils.NewMockOwnershipManagerInterface(gomock.NewController(GinkgoT()))
			diskutils.DefaultOwnershipManager = manager
			return manager
		}

		DescribeTable("should hand the pipe socket to the ownership manager for a nonroot VMI",
			func(makeNonRoot func(*v1.VirtualMachineInstance)) {
				makeNonRoot(vmi)

				var owned *safepath.Path
				mockOwnership().EXPECT().SetFileOwnership(gomock.Any()).Times(1).
					Do(func(path *safepath.Path) { owned = path })

				expectPipeInjection()
				DeferCleanup(func() { close(stopChan) })

				Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
				Expect(owned).ToNot(BeNil())
				Expect(owned.String()).To(ContainSubstring("domain-notify-pipe.sock"))
			},
			Entry("marked by a nonzero runtime user", func(vmi *v1.VirtualMachineInstance) {
				vmi.Status.RuntimeUser = 107
			}),
			Entry("marked by the deprecated nonroot annotation", func(vmi *v1.VirtualMachineInstance) {
				vmi.Annotations = map[string]string{v1.DeprecatedNonRootVMIAnnotation: ""}
			}),
		)

		It("should not touch ownership for a root VMI", func() {
			vmi.Status.RuntimeUser = 0
			vmi.Annotations = nil

			mockOwnership().EXPECT().SetFileOwnership(gomock.Any()).Times(0)

			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
		})
	})

	Context("shutdown", func() {

		It("should keep the pipe open while the stop channel is open", func() {
			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())

			Consistently(socketPath, settling).Should(BeAnExistingFile())
		})

		It("should tear the pipe down when the stop channel is closed", func() {
			expectPipeInjection()

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
			Expect(socketPath).To(BeAnExistingFile())

			close(stopChan)

			Eventually(socketPath, timeout).ShouldNot(BeAnExistingFile())

			_, err := net.Dial("unix", socketPath)
			Expect(err).To(MatchError(syscall.ENOENT))
		})
	})
})
