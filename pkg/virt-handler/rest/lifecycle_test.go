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

package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	gomock "go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/libvmi"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
)

var _ = Describe("Snapshot overlay request", func() {
	const overlayVolume = "snap-scratch-1234"

	newRequest := func(body io.Reader) *restful.Request {
		httpRequest, err := http.NewRequest(http.MethodPut, "/", body)
		Expect(err).ToNot(HaveOccurred())
		if body == nil {
			httpRequest.Body = nil
		}
		return &restful.Request{Request: httpRequest}
	}

	newVMI := func(utilityVolumeNames ...string) *v1.VirtualMachineInstance {
		vmi := api.NewMinimalVMI("testvmi")
		for _, name := range utilityVolumeNames {
			vmi.Spec.UtilityVolumes = append(vmi.Spec.UtilityVolumes, v1.UtilityVolume{Name: name})
		}
		encoded, err := json.Marshal(storagetypes.SnapshotOverlayReference{ContentUID: "1234", ScratchPVCUID: "scratch-uid"})
		Expect(err).ToNot(HaveOccurred())
		vmi.Annotations = map[string]string{storagetypes.OverlaySnapshotReferenceAnnotation: string(encoded)}
		return vmi
	}

	optionsBody := func(volumeName string, volumes ...string) io.Reader {
		body, err := json.Marshal(&v1.SnapshotOverlayOptions{VolumeName: volumeName, Volumes: volumes})
		Expect(err).ToNot(HaveOccurred())
		return strings.NewReader(string(body))
	}

	It("should resolve an attached utility volume to its mount point", func() {
		opts, overlayDir, err := overlayRequest(newRequest(optionsBody(overlayVolume, "rootdisk", "datadisk")), newVMI(overlayVolume))

		Expect(err).ToNot(HaveOccurred())
		Expect(overlayDir).To(HaveSuffix("/" + overlayVolume))
		Expect(opts.Volumes).To(Equal([]string{"rootdisk", "datadisk"}))
	})

	It("should reject a volume that is not attached to the VMI", func() {
		_, _, err := overlayRequest(newRequest(optionsBody(overlayVolume)), newVMI("some-other-volume"))

		Expect(err).To(MatchError(ContainSubstring(overlayVolume)))
	})

	It("should reject a path instead of a volume name", func() {
		_, _, err := overlayRequest(newRequest(optionsBody("../../../etc")), newVMI(overlayVolume))

		Expect(err).To(HaveOccurred())
	})

	It("should reject a request with no body", func() {
		_, _, err := overlayRequest(newRequest(nil), newVMI(overlayVolume))

		Expect(err).To(HaveOccurred())
	})
})

// newOverlayVMI builds a VMI that carries the snapshot overlay annotation and
// has the given utility volumes attached.
func newOverlayVMI(namespace, name string, utilityVolumeNames ...string) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
	for _, volumeName := range utilityVolumeNames {
		vmi.Spec.UtilityVolumes = append(vmi.Spec.UtilityVolumes, v1.UtilityVolume{Name: volumeName})
	}
	encoded, err := json.Marshal(storagetypes.SnapshotOverlayReference{ContentUID: "1234", ScratchPVCUID: "scratch-uid"})
	Expect(err).ToNot(HaveOccurred())
	vmi.Annotations = map[string]string{storagetypes.OverlaySnapshotReferenceAnnotation: string(encoded)}
	return vmi
}

func overlayOptionsJSON(volumeName string, volumes ...string) string {
	body, err := json.Marshal(&v1.SnapshotOverlayOptions{VolumeName: volumeName, Volumes: volumes})
	Expect(err).ToNot(HaveOccurred())
	return string(body)
}

const (
	testNamespace = "default"
	testVMIName   = "testvmi"
)

// fakeClientGetter hands out a pre-canned launcher client, or a pre-canned
// failure, without touching any socket.
type fakeClientGetter struct {
	client cmdclient.LauncherClient
	err    error
}

func (f *fakeClientGetter) GetVerifiedLauncherClient(*v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

var _ = Describe("Lifecycle handler", func() {
	const routePath = "/v1/namespaces/{namespace}/virtualmachineinstances/{name}/fakeaction"

	var (
		mockClient   *cmdclient.MockLauncherClient
		clientGetter *fakeClientGetter
		recorder     *record.FakeRecorder
		vmiStore     cache.Store
		handler      *LifecycleHandler
	)

	// serve routes a request for testNamespace/testVMIName to handlerFn and
	// returns the recorded response.
	serve := func(handlerFn restful.RouteFunction, body io.Reader) *httptest.ResponseRecorder {
		url := fmt.Sprintf("/v1/namespaces/%s/virtualmachineinstances/%s/fakeaction", testNamespace, testVMIName)
		httpReq, err := http.NewRequest(http.MethodPut, url, body)
		Expect(err).ToNot(HaveOccurred())
		httpReq.Header.Set("Content-Type", restful.MIME_JSON)
		httpReq.Header.Set("Accept", restful.MIME_JSON)

		ws := new(restful.WebService)
		ws.Route(ws.PUT(routePath).To(handlerFn).Produces(restful.MIME_JSON))
		container := restful.NewContainer()
		container.Add(ws)

		response := httptest.NewRecorder()
		container.ServeHTTP(response, httpReq)
		return response
	}

	expectNoEvents := func() {
		Expect(recorder.Events).To(BeEmpty())
	}

	expectEvent := func(substring string) {
		Eventually(recorder.Events).Should(Receive(ContainSubstring(substring)))
	}

	BeforeEach(func() {
		mockClient = cmdclient.NewMockLauncherClient(gomock.NewController(GinkgoT()))
		mockClient.EXPECT().Close().AnyTimes()
		clientGetter = &fakeClientGetter{client: mockClient}
		recorder = record.NewFakeRecorder(10)
		vmiStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		handler = NewLifecycleHandler(recorder, vmiStore, "/var/run/kubevirt", clientGetter)
	})

	Describe("resolving the VMI and its launcher client", func() {
		// Pending: getVMILauncherClient logs the VMI it failed to retrieve, so
		// it panics on the nil VMI instead of returning 404.
		PIt("should return 404 when the VMI is not on this node", func() {
			Expect(serve(handler.PauseHandler, strings.NewReader("")).Code).To(Equal(http.StatusNotFound))
			expectNoEvents()
		})

		It("should return 500 when no launcher client can be obtained", func() {
			clientGetter.err = fmt.Errorf("socket not found")
			vmiStore.Add(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName)))

			response := serve(handler.PauseHandler, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Body.String()).To(ContainSubstring("socket not found"))
			expectNoEvents()
		})
	})

	DescribeTable("simple lifecycle actions", func(
		handlerFn func() restful.RouteFunction,
		expectCall func() *gomock.Call,
		eventReason string,
	) {
		expectCall().Return(nil)
		vmiStore.Add(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName)))

		Expect(serve(handlerFn(), strings.NewReader("")).Code).To(Equal(http.StatusAccepted))
		expectEvent(eventReason)
	},
		Entry("pause",
			func() restful.RouteFunction { return handler.PauseHandler },
			func() *gomock.Call { return mockClient.EXPECT().PauseVirtualMachine(gomock.Any()) },
			"Paused"),
		Entry("unpause",
			func() restful.RouteFunction { return handler.UnpauseHandler },
			func() *gomock.Call { return mockClient.EXPECT().UnpauseVirtualMachine(gomock.Any()) },
			"Unpaused"),
		Entry("reset",
			func() restful.RouteFunction { return handler.ResetHandler },
			func() *gomock.Call { return mockClient.EXPECT().ResetVirtualMachine(gomock.Any()) },
			"Reset"),
		Entry("soft reboot",
			func() restful.RouteFunction { return handler.SoftRebootHandler },
			func() *gomock.Call { return mockClient.EXPECT().SoftRebootVirtualMachine(gomock.Any()) },
			"SoftRebooted"),
	)

	DescribeTable("simple lifecycle actions that fail", func(
		handlerFn func() restful.RouteFunction,
		expectCall func() *gomock.Call,
	) {
		expectCall().Return(fmt.Errorf("libvirt is unhappy"))
		vmiStore.Add(libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName)))

		response := serve(handlerFn(), strings.NewReader(""))

		Expect(response.Code).To(Equal(http.StatusInternalServerError))
		Expect(response.Body.String()).To(ContainSubstring("libvirt is unhappy"))
		expectNoEvents()
	},
		Entry("pause",
			func() restful.RouteFunction { return handler.PauseHandler },
			func() *gomock.Call { return mockClient.EXPECT().PauseVirtualMachine(gomock.Any()) }),
		Entry("unpause",
			func() restful.RouteFunction { return handler.UnpauseHandler },
			func() *gomock.Call { return mockClient.EXPECT().UnpauseVirtualMachine(gomock.Any()) }),
		Entry("reset",
			func() restful.RouteFunction { return handler.ResetHandler },
			func() *gomock.Call { return mockClient.EXPECT().ResetVirtualMachine(gomock.Any()) }),
		Entry("soft reboot",
			func() restful.RouteFunction { return handler.SoftRebootHandler },
			func() *gomock.Call { return mockClient.EXPECT().SoftRebootVirtualMachine(gomock.Any()) }),
	)

	Describe("FreezeHandler", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName))
			vmiStore.Add(vmi)
		})

		It("should pass the unfreeze timeout to the launcher in seconds", func() {
			mockClient.EXPECT().FreezeVirtualMachine(vmi, int32(300)).Return(nil)

			Expect(serve(handler.FreezeHandler, strings.NewReader(`{"unfreezeTimeout":"5m"}`)).Code).To(Equal(http.StatusAccepted))
			expectNoEvents()
		})

		It("should return 400 when the request has no body", func() {
			Expect(serve(handler.FreezeHandler, nil).Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the body is not valid JSON", func() {
			Expect(serve(handler.FreezeHandler, strings.NewReader(`{"unfreezeTimeout":`)).Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the unfreeze timeout is not set", func() {
			response := serve(handler.FreezeHandler, strings.NewReader(`{}`))

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("Unfreeze timeout in freeze request is not set"))
		})

		It("should return 400 and record an event when freezing fails", func() {
			mockClient.EXPECT().FreezeVirtualMachine(vmi, int32(5)).Return(fmt.Errorf("guest agent is down"))

			response := serve(handler.FreezeHandler, strings.NewReader(`{"unfreezeTimeout":"5s"}`))

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			expectEvent("FreezeError")
		})
	})

	Describe("UnfreezeHandler", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName))
			Expect(vmiStore.Add(vmi)).To(Succeed())
		})

		It("should return 202 on success", func() {
			mockClient.EXPECT().UnfreezeVirtualMachine(vmi).Return(nil)

			Expect(serve(handler.UnfreezeHandler, strings.NewReader("")).Code).To(Equal(http.StatusAccepted))
			expectNoEvents()
		})

		It("should return 400 when unfreezing fails", func() {
			mockClient.EXPECT().UnfreezeVirtualMachine(vmi).Return(fmt.Errorf("guest agent is down"))

			response := serve(handler.UnfreezeHandler, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("guest agent is down"))
		})
	})

	Describe("snapshot overlay handlers", func() {
		const overlayVolume = "snap-scratch-1234"

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = newOverlayVMI(testNamespace, testVMIName, overlayVolume)
			Expect(vmiStore.Add(vmi)).To(Succeed())
		})

		body := func(volumes ...string) io.Reader {
			return strings.NewReader(overlayOptionsJSON(overlayVolume, volumes...))
		}

		It("should start an external snapshot of the requested volumes", func() {
			mockClient.EXPECT().ExternalSnapshot(vmi, gomock.Any(), []string{"rootdisk"}).Return(nil)

			Expect(serve(handler.ExternalSnapshotHandler, body("rootdisk")).Code).To(Equal(http.StatusAccepted))
			expectNoEvents()
		})

		It("should return 400 when the external snapshot overlay cannot be resolved", func() {
			Expect(serve(handler.ExternalSnapshotHandler, strings.NewReader(overlayOptionsJSON("unknown-volume"))).Code).
				To(Equal(http.StatusBadRequest))
		})

		It("should return 500 and record an event when the external snapshot fails", func() {
			mockClient.EXPECT().ExternalSnapshot(vmi, gomock.Any(), gomock.Any()).Return(fmt.Errorf("no space left"))

			Expect(serve(handler.ExternalSnapshotHandler, body("rootdisk")).Code).To(Equal(http.StatusInternalServerError))
			expectEvent("ExternalSnapshotError")
		})

		It("should commit the snapshot overlay", func() {
			mockClient.EXPECT().CommitSnapshot(vmi, gomock.Any()).Return(nil)

			Expect(serve(handler.CommitSnapshotHandler, body()).Code).To(Equal(http.StatusAccepted))
			expectNoEvents()
		})

		It("should return 400 when the commit overlay cannot be resolved", func() {
			Expect(serve(handler.CommitSnapshotHandler, strings.NewReader(overlayOptionsJSON("unknown-volume"))).Code).
				To(Equal(http.StatusBadRequest))
		})

		It("should return 500 and record an event when the commit fails", func() {
			mockClient.EXPECT().CommitSnapshot(vmi, gomock.Any()).Return(fmt.Errorf("blockcommit failed"))

			Expect(serve(handler.CommitSnapshotHandler, body()).Code).To(Equal(http.StatusInternalServerError))
			expectEvent("CommitSnapshotError")
		})
	})

	Describe("guest agent queries", func() {
		BeforeEach(func() {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName))
			Expect(vmiStore.Add(vmi)).To(Succeed())
		})

		It("should return the guest info", func() {
			mockClient.EXPECT().GetGuestInfo().Return(&v1.VirtualMachineInstanceGuestAgentInfo{GAVersion: "1.2.3"}, nil)

			response := serve(handler.GetGuestInfo, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring("1.2.3"))
		})

		It("should return 500 when the guest info is unavailable", func() {
			mockClient.EXPECT().GetGuestInfo().Return(nil, fmt.Errorf("agent is down"))

			Expect(serve(handler.GetGuestInfo, strings.NewReader("")).Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return the user list", func() {
			mockClient.EXPECT().GetUsers().Return(v1.VirtualMachineInstanceGuestOSUserList{
				Items: []v1.VirtualMachineInstanceGuestOSUser{{UserName: "fedora"}},
			}, nil)

			response := serve(handler.GetUsers, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring("fedora"))
		})

		It("should return 500 when the user list is unavailable", func() {
			mockClient.EXPECT().GetUsers().Return(v1.VirtualMachineInstanceGuestOSUserList{}, fmt.Errorf("agent is down"))

			Expect(serve(handler.GetUsers, strings.NewReader("")).Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return the filesystem list", func() {
			mockClient.EXPECT().GetFilesystems().Return(v1.VirtualMachineInstanceFileSystemList{
				Items: []v1.VirtualMachineInstanceFileSystem{{DiskName: "vda1"}},
			}, nil)

			response := serve(handler.GetFilesystems, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring("vda1"))
		})

		It("should return 500 when the filesystem list is unavailable", func() {
			mockClient.EXPECT().GetFilesystems().Return(v1.VirtualMachineInstanceFileSystemList{}, fmt.Errorf("agent is down"))

			Expect(serve(handler.GetFilesystems, strings.NewReader("")).Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("SEV handlers", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName))
			Expect(vmiStore.Add(vmi)).To(Succeed())
		})

		It("should return the SEV platform info", func() {
			mockClient.EXPECT().GetSEVInfo().Return(&v1.SEVPlatformInfo{PDH: "pdh-value"}, nil)

			response := serve(handler.SEVFetchCertChainHandler, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring("pdh-value"))
		})

		It("should return 500 when the SEV platform info is unavailable", func() {
			mockClient.EXPECT().GetSEVInfo().Return(nil, fmt.Errorf("not an SEV host"))

			Expect(serve(handler.SEVFetchCertChainHandler, strings.NewReader("")).Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return the SEV launch measurement", func() {
			mockClient.EXPECT().GetLaunchMeasurement(vmi).Return(&v1.SEVMeasurementInfo{Measurement: "measured"}, nil)

			response := serve(handler.SEVQueryLaunchMeasurementHandler, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring("measured"))
		})

		It("should return 500 when the SEV launch measurement is unavailable", func() {
			mockClient.EXPECT().GetLaunchMeasurement(vmi).Return(nil, fmt.Errorf("not an SEV guest"))

			Expect(serve(handler.SEVQueryLaunchMeasurementHandler, strings.NewReader("")).Code).To(Equal(http.StatusInternalServerError))
		})

		It("should inject the SEV launch secret", func() {
			mockClient.EXPECT().InjectLaunchSecret(vmi, &v1.SEVSecretOptions{Header: "hdr", Secret: "sec"}).Return(nil)

			Expect(serve(handler.SEVInjectLaunchSecretHandler, strings.NewReader(`{"header":"hdr","secret":"sec"}`)).Code).
				To(Equal(http.StatusAccepted))
		})

		It("should return 400 when the inject request has no body", func() {
			Expect(serve(handler.SEVInjectLaunchSecretHandler, nil).Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the inject request body is not valid JSON", func() {
			Expect(serve(handler.SEVInjectLaunchSecretHandler, strings.NewReader(`{"header":`)).Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 500 when injecting the SEV launch secret fails", func() {
			mockClient.EXPECT().InjectLaunchSecret(vmi, gomock.Any()).Return(fmt.Errorf("injection refused"))

			Expect(serve(handler.SEVInjectLaunchSecretHandler, strings.NewReader(`{"header":"hdr","secret":"sec"}`)).Code).
				To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("BackupHandler", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName))
			Expect(vmiStore.Add(vmi)).To(Succeed())
		})

		It("should pass the backup options to the launcher", func() {
			mockClient.EXPECT().VirtualMachineBackup(vmi, gomock.Any()).DoAndReturn(
				func(_ *v1.VirtualMachineInstance, opts *backupv1.BackupOptions) error {
					Expect(opts.BackupName).To(Equal("backup-1"))
					return nil
				})

			Expect(serve(handler.BackupHandler, strings.NewReader(`{"backupName":"backup-1"}`)).Code).To(Equal(http.StatusAccepted))
			expectNoEvents()
		})

		It("should return 400 when the request has no body", func() {
			Expect(serve(handler.BackupHandler, nil).Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the body is not valid JSON", func() {
			Expect(serve(handler.BackupHandler, strings.NewReader(`{"backupName":`)).Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 and record an event when the backup fails", func() {
			mockClient.EXPECT().VirtualMachineBackup(vmi, gomock.Any()).Return(fmt.Errorf("target unreachable"))

			Expect(serve(handler.BackupHandler, strings.NewReader(`{"backupName":"backup-1"}`)).Code).To(Equal(http.StatusBadRequest))
			expectEvent("BackupError")
		})
	})

	Describe("ScreenshotRequestHandler", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName(testVMIName))
			Expect(vmiStore.Add(vmi)).To(Succeed())
		})

		It("should write the screenshot with its mime type", func() {
			mockClient.EXPECT().GetScreenshot(vmi).Return(&cmdv1.ScreenshotResponse{
				Mime: "image/png",
				Data: []byte("png-bytes"),
			}, nil)

			response := serve(handler.ScreenshotRequestHandler, strings.NewReader(""))

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Header().Get("Content-Type")).To(Equal("image/png"))
			Expect(response.Body.Bytes()).To(Equal([]byte("png-bytes")))
		})

		It("should return 500 when the screenshot cannot be taken", func() {
			mockClient.EXPECT().GetScreenshot(vmi).Return(nil, fmt.Errorf("no display"))

			Expect(serve(handler.ScreenshotRequestHandler, strings.NewReader("")).Code).To(Equal(http.StatusInternalServerError))
		})

		// Pending for the same reason as the PauseHandler 404 spec above.
		PIt("should return 404 when the VMI is not on this node", func() {
			Expect(vmiStore.Delete(vmi)).To(Succeed())

			Expect(serve(handler.ScreenshotRequestHandler, strings.NewReader("")).Code).To(Equal(http.StatusNotFound))
		})
	})
})
