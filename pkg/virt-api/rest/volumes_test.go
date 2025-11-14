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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Add/Remove Volume Subresource api", func() {
	var (
		request    *restful.Request
		response   *restful.Response
		virtClient *kubecli.MockKubevirtClient
		vmClient   *kubecli.MockVirtualMachineInterface
		vmiClient  *kubecli.MockVirtualMachineInstanceInterface
		app        *SubresourceAPIApp

		kv = &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
	)

	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGates := func(featureGates ...string) {
		fgs := slices.Clone(featureGates)
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = fgs
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	BeforeEach(func() {
		request = restful.NewRequest(&http.Request{})
		request.PathParameters()["name"] = testVMName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault
		recorder := httptest.NewRecorder()
		response = restful.NewResponse(recorder)

		backend := ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		ctrl := gomock.NewController(GinkgoT())

		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()

		app = NewSubresourceAPIApp(virtClient, nil, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	newAddVolumeBody := func(opts *v1.AddVolumeOptions) io.ReadCloser {
		optsJson, _ := json.Marshal(opts)
		return &readCloserWrapper{bytes.NewReader(optsJson)}
	}
	newRemoveVolumeBody := func(opts *v1.RemoveVolumeOptions) io.ReadCloser {
		optsJson, _ := json.Marshal(opts)
		return &readCloserWrapper{bytes.NewReader(optsJson)}
	}

	VolumeUpdateTests := func(featureGate string) {
		DescribeTable("Should succeed with add/volume request", func(addOpts *v1.AddVolumeOptions, removeOpts *v1.RemoveVolumeOptions, isVM bool, code int, enableGate bool) {
			if enableGate {
				enableFeatureGates(featureGate)
			}
			if addOpts != nil {
				request.Request.Body = newAddVolumeBody(addOpts)
			} else {
				request.Request.Body = newRemoveVolumeBody(removeOpts)
			}

			vmi := libvmi.New(
				libvmi.WithName(request.PathParameter("name")),
				libvmi.WithNamespace(metav1.NamespaceDefault),
				libvmi.WithPersistentVolumeClaim("existingvol", "testpvcdiskclaim"),
				libvmi.WithPersistentVolumeClaim("hotpluggedPVC", "hotpluggedPVC"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			)
			vmi.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim.Hotpluggable = true

			if isVM {
				vm := libvmi.NewVirtualMachine(vmi)
				vm.Name = request.PathParameter("name")
				vm.Namespace = metav1.NamespaceDefault

				vmClient.EXPECT().Get(context.Background(), vm.Name, metav1.GetOptions{}).Return(vm, nil).AnyTimes()

				if featureGate == featuregate.HotplugVolumesGate {
					patchedVM := vm.DeepCopy()
					patchedVM.Status.VolumeRequests = append(patchedVM.Status.VolumeRequests, v1.VirtualMachineVolumeRequest{AddVolumeOptions: addOpts, RemoveVolumeOptions: removeOpts})

					if addOpts != nil {
						vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions) (interface{}, interface{}) {
								//check that dryRun option has been propagated to patch request
								Expect(opts.DryRun).To(BeEquivalentTo(addOpts.DryRun))
								return patchedVM, nil
							}).AnyTimes()
						app.VMAddVolumeRequestHandler(request, response)
					} else {
						vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions) (interface{}, interface{}) {
								//check that dryRun option has been propagated to patch request
								Expect(opts.DryRun).To(BeEquivalentTo(removeOpts.DryRun))
								return patchedVM, nil
							})
						app.VMRemoveVolumeRequestHandler(request, response)
					}
				} else {
					if addOpts != nil {
						vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
								//check that dryRun option has been propagated to patch request
								Expect(opts.DryRun).To(BeEquivalentTo(addOpts.DryRun))
								return vm, nil
							}).AnyTimes()
						app.VMAddVolumeRequestHandler(request, response)
					} else {
						vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
							func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
								//check that dryRun option has been propagated to patch request
								Expect(opts.DryRun).To(BeEquivalentTo(removeOpts.DryRun))
								return vm, nil
							})
						app.VMRemoveVolumeRequestHandler(request, response)
					}
				}
			} else {
				vmiClient.EXPECT().Get(context.Background(), vmi.Name, metav1.GetOptions{}).Return(vmi, nil).AnyTimes()
				if addOpts != nil {
					vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
							//check that dryRun option has been propagated to patch request
							Expect(opts.DryRun).To(BeEquivalentTo(addOpts.DryRun))
							return vmi, nil
						}).AnyTimes()
					app.VMIAddVolumeRequestHandler(request, response)
				} else {
					vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts metav1.PatchOptions, _ ...string) (interface{}, interface{}) {
							//check that dryRun option has been propagated to patch request
							Expect(opts.DryRun).To(BeEquivalentTo(removeOpts.DryRun))
							return vmi, nil
						}).AnyTimes()
					app.VMIRemoveVolumeRequestHandler(request, response)
				}
			}

			Expect(response.StatusCode()).To(Equal(code))
		},
			Entry("VM with a valid add volume request", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, true, http.StatusAccepted, true),
			Entry("VMI with a valid add volume request", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, false, http.StatusAccepted, true),
			Entry("VMI with an invalid add volume request that's missing a name", &v1.AddVolumeOptions{
				VolumeSource: &v1.HotplugVolumeSource{},
				Disk:         &v1.Disk{},
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a disk", &v1.AddVolumeOptions{
				Name:         "vol1",
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a volume", &v1.AddVolumeOptions{
				Name: "vol1",
				Disk: &v1.Disk{},
			}, nil, false, http.StatusBadRequest, true),
			Entry("VM with a valid remove volume request", nil, &v1.RemoveVolumeOptions{
				Name: "hotpluggedPVC",
			}, true, http.StatusAccepted, true),
			Entry("VMI with a valid remove volume request", nil, &v1.RemoveVolumeOptions{
				Name: "hotpluggedPVC",
			}, false, http.StatusAccepted, true),
			Entry("VMI with a invalid remove volume request missing a name", nil, &v1.RemoveVolumeOptions{}, false, http.StatusBadRequest, true),
			Entry("VMI with a valid remove volume request but no feature gate", nil, &v1.RemoveVolumeOptions{
				Name: "existingvol",
			}, false, http.StatusBadRequest, false),
			Entry("VM with a valid add volume request but no feature gate", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}, nil, true, http.StatusBadRequest, false),
			Entry("VM with a valid add volume request with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       withDryRun(),
			}, nil, true, http.StatusAccepted, true),
			Entry("VMI with a valid add volume request with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       withDryRun(),
			}, nil, false, http.StatusAccepted, true),
			Entry("VMI with an invalid add volume request that's missing a name with DryRun", &v1.AddVolumeOptions{
				VolumeSource: &v1.HotplugVolumeSource{},
				Disk:         &v1.Disk{},
				DryRun:       withDryRun(),
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a disk with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       withDryRun(),
			}, nil, false, http.StatusBadRequest, true),
			Entry("VMI with an invalid add volume request that's missing a volume with DryRun", &v1.AddVolumeOptions{
				Name:   "vol1",
				Disk:   &v1.Disk{},
				DryRun: withDryRun(),
			}, nil, false, http.StatusBadRequest, true),
			Entry("VM with a valid remove volume request with DryRun", nil, &v1.RemoveVolumeOptions{
				Name:   "hotpluggedPVC",
				DryRun: withDryRun(),
			}, true, http.StatusAccepted, true),
			Entry("VMI with a valid remove volume request with DryRun", nil, &v1.RemoveVolumeOptions{
				Name:   "hotpluggedPVC",
				DryRun: withDryRun(),
			}, false, http.StatusAccepted, true),
			Entry("VMI with a invalid remove volume request missing a name with DryRun", nil, &v1.RemoveVolumeOptions{
				DryRun: withDryRun(),
			}, false, http.StatusBadRequest, true),
			Entry("VMI with a valid remove volume request but no feature gate with DryRun", nil, &v1.RemoveVolumeOptions{
				Name:   "existingvol",
				DryRun: withDryRun(),
			}, false, http.StatusBadRequest, false),
			Entry("VM with a valid add volume request but no feature gate with DryRun", &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
				DryRun:       withDryRun(),
			}, nil, true, http.StatusBadRequest, false),
		)
	}

	Context("With DeclarativeHotplugVolumes feature gate", func() {
		VolumeUpdateTests(featuregate.DeclarativeHotplugVolumesGate)
	})

	Context("With HotplugVolumes feature gate", func() {
		VolumeUpdateTests(featuregate.HotplugVolumesGate)
	})

	DescribeTable("Should handle VMI with owner and", func(addOpts *v1.AddVolumeOptions, removeOpts *v1.RemoveVolumeOptions, code int, featuregates ...string) {
		enableFeatureGates(featuregates...)
		if addOpts != nil {
			request.Request.Body = newAddVolumeBody(addOpts)
		} else {
			request.Request.Body = newRemoveVolumeBody(removeOpts)
		}

		vmi := libvmi.New(
			libvmi.WithName(request.PathParameter("name")),
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithPersistentVolumeClaim("existingvol", "testpvcdiskclaim"),
			libvmi.WithPersistentVolumeClaim("hotpluggedPVC", "hotpluggedPVC"),
			libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
		)
		vmi.Spec.Volumes[1].VolumeSource.PersistentVolumeClaim.Hotpluggable = true
		vmi.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "kubevirt.io/v1",
				Kind:       "VirtualMachine",
				Name:       request.PathParameter("name"),
				UID:        types.UID("1234"),
				Controller: pointer.P(true),
			},
		}

		vmiClient.EXPECT().Get(context.Background(), vmi.Name, metav1.GetOptions{}).Return(vmi, nil).AnyTimes()

		if code == http.StatusAccepted {
			vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).Return(vmi, nil).AnyTimes()
		}

		if addOpts != nil {
			app.VMIAddVolumeRequestHandler(request, response)
		} else {
			app.VMIRemoveVolumeRequestHandler(request, response)
		}

		Expect(response.StatusCode()).To(Equal(code))
	},
		Entry("Reject Add with DeclarativeHotplugVolumes", &v1.AddVolumeOptions{
			Name:         "vol1",
			Disk:         &v1.Disk{},
			VolumeSource: &v1.HotplugVolumeSource{},
		}, nil, http.StatusBadRequest, featuregate.DeclarativeHotplugVolumesGate),
		Entry("Accept Add with HotplugVolumes", &v1.AddVolumeOptions{
			Name:         "vol1",
			Disk:         &v1.Disk{},
			VolumeSource: &v1.HotplugVolumeSource{},
		}, nil, http.StatusAccepted, featuregate.HotplugVolumesGate),
		Entry("Accept Add with both featuregates", &v1.AddVolumeOptions{
			Name:         "vol1",
			Disk:         &v1.Disk{},
			VolumeSource: &v1.HotplugVolumeSource{},
		}, nil, http.StatusAccepted, featuregate.HotplugVolumesGate, featuregate.DeclarativeHotplugVolumesGate),
		Entry("Reject Remove with DeclarativeHotplugVolumes", nil, &v1.RemoveVolumeOptions{
			Name: "hotpluggedPVC",
		}, http.StatusBadRequest, featuregate.DeclarativeHotplugVolumesGate),
		Entry("Accept Remove with HotplugVolumes", nil, &v1.RemoveVolumeOptions{
			Name: "hotpluggedPVC",
		}, http.StatusAccepted, featuregate.HotplugVolumesGate),
		Entry("Accept Remove with both featuregates", nil, &v1.RemoveVolumeOptions{
			Name: "hotpluggedPVC",
		}, http.StatusAccepted, featuregate.HotplugVolumesGate, featuregate.DeclarativeHotplugVolumesGate),
	)

	DescribeTable("Should generate expected vmi patch", func(volumeRequest *v1.VirtualMachineVolumeRequest, expectedPatchSet *patch.PatchSet) {

		vmi := api.NewMinimalVMI(request.PathParameter("name"))
		vmi.Namespace = metav1.NamespaceDefault
		vmi.Status.Phase = v1.Running
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "existingvol",
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "existingvol",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "testpvcdiskclaim",
				}},
			},
		})

		patch, err := generateVolumeRequestPatchVMI(&vmi.Spec, volumeRequest)
		Expect(err).ToNot(HaveOccurred())

		patchBytes, err := expectedPatchSet.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).To(Equal(patchBytes))
	},
		Entry("add volume request",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol1",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			},
			patch.New(
				patch.WithTest("/spec/volumes", []v1.Volume{{
					Name: "existingvol",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				}}),
				patch.WithTest("/spec/domain/devices/disks", []v1.Disk{{Name: "existingvol"}}),
				patch.WithReplace("/spec/volumes", []v1.Volume{
					{
						Name: "existingvol",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testpvcdiskclaim",
							}},
						},
					},
					{Name: "vol1"},
				}),
				patch.WithReplace("/spec/domain/devices/disks", []v1.Disk{{Name: "existingvol"}, {Name: "vol1"}}),
			),
		),
		Entry("remove volume request",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "existingvol",
				},
			},
			patch.New(
				patch.WithTest("/spec/volumes", []v1.Volume{{
					Name: "existingvol",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				}}),
				patch.WithTest("/spec/domain/devices/disks", []v1.Disk{{Name: "existingvol"}}),
				patch.WithReplace("/spec/volumes", []v1.Volume{}),
				patch.WithReplace("/spec/domain/devices/disks", []v1.Disk{}),
			)),
	)

	DescribeTable("Should generate expected vm patch (volume request)", func(volumeRequest *v1.VirtualMachineVolumeRequest, existingVolumeRequests []v1.VirtualMachineVolumeRequest, expectedPatchSet *patch.PatchSet, expectError bool) {

		vm := newMinimalVM(request.PathParameter("name"))
		vm.Namespace = metav1.NamespaceDefault

		if len(existingVolumeRequests) > 0 {
			vm.Status.VolumeRequests = existingVolumeRequests
		}

		generatedPatch, err := generateVMVolumeRequestPatch(vm, volumeRequest)
		if expectError {
			Expect(err).To(HaveOccurred())
			Expect(generatedPatch).To(BeEmpty())
			return
		}

		Expect(err).ToNot(HaveOccurred())
		expectedPatch, err := expectedPatchSet.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		Expect(generatedPatch).To(Equal(expectedPatch))
	},
		Entry("add volume request with no existing volumes",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol1",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			},
			nil,
			patch.New(
				patch.WithTest("/status/volumeRequests", nil),
				patch.WithAdd("/status/volumeRequests", []v1.VirtualMachineVolumeRequest{{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol1",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				}}),
			),
			false),
		Entry("add volume request that already exists should fail",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol1",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			},
			[]v1.VirtualMachineVolumeRequest{{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol1",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			}},
			nil,
			true),
		Entry("add volume request when volume requests alread exist",
			&v1.VirtualMachineVolumeRequest{AddVolumeOptions: &v1.AddVolumeOptions{
				Name:         "vol1",
				Disk:         &v1.Disk{},
				VolumeSource: &v1.HotplugVolumeSource{},
			}},
			[]v1.VirtualMachineVolumeRequest{{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol2",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			}},
			patch.New(
				patch.WithTest("/status/volumeRequests", []v1.VirtualMachineVolumeRequest{{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol2",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				}}),
				patch.WithReplace("/status/volumeRequests", []v1.VirtualMachineVolumeRequest{
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol2",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
					{
						AddVolumeOptions: &v1.AddVolumeOptions{
							Name:         "vol1",
							Disk:         &v1.Disk{},
							VolumeSource: &v1.HotplugVolumeSource{},
						},
					},
				}),
			),
			false),
		Entry("remove volume request with no existing volume request", &v1.VirtualMachineVolumeRequest{
			RemoveVolumeOptions: &v1.RemoveVolumeOptions{
				Name: "vol1",
			}},
			nil,
			patch.New(
				patch.WithTest("/status/volumeRequests", nil),
				patch.WithAdd("/status/volumeRequests", []v1.VirtualMachineVolumeRequest{
					{RemoveVolumeOptions: &v1.RemoveVolumeOptions{Name: "vol1"}},
				}),
			),
			false),
		Entry("remove volume request should replace add volume request",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "vol2",
				},
			},
			[]v1.VirtualMachineVolumeRequest{{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol2",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			}},
			patch.New(
				patch.WithTest("/status/volumeRequests", []v1.VirtualMachineVolumeRequest{{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name:         "vol2",
						Disk:         &v1.Disk{},
						VolumeSource: &v1.HotplugVolumeSource{},
					},
				}}),
				patch.WithReplace("/status/volumeRequests", []v1.VirtualMachineVolumeRequest{
					{RemoveVolumeOptions: &v1.RemoveVolumeOptions{Name: "vol2"}},
				}),
			),
			false),
		Entry("remove volume request that already exists should fail",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "vol2",
				},
			},
			[]v1.VirtualMachineVolumeRequest{{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "vol2",
				},
			}},
			nil,
			true),
	)

	DescribeTable("Should generate expected vm patch (declarative)", func(volumeRequest *v1.VirtualMachineVolumeRequest, expectedPatchSet *patch.PatchSet) {

		vm := libvmi.NewVirtualMachine(api.NewMinimalVMI(request.PathParameter("name")))
		vm.Namespace = metav1.NamespaceDefault
		vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "existingvol",
		})
		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: "existingvol",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "testpvcdiskclaim",
				}},
			},
		})

		patch, err := generateVolumeRequestPatchVM(&vm.Spec.Template.Spec, volumeRequest)
		Expect(err).ToNot(HaveOccurred())

		patchBytes, err := expectedPatchSet.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).To(Equal(patchBytes))
	},
		Entry("add volume request",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol1",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			},
			patch.New(
				patch.WithTest("/spec/template/spec/volumes", []v1.Volume{{
					Name: "existingvol",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				}}),
				patch.WithTest("/spec/template/spec/domain/devices/disks", []v1.Disk{{Name: "existingvol"}}),
				patch.WithReplace("/spec/template/spec/volumes", []v1.Volume{
					{
						Name: "existingvol",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testpvcdiskclaim",
							}},
						},
					},
					{Name: "vol1"},
				}),
				patch.WithReplace("/spec/template/spec/domain/devices/disks", []v1.Disk{{Name: "existingvol"}, {Name: "vol1"}}),
			),
		),
		Entry("remove volume request",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "existingvol",
				},
			},
			patch.New(
				patch.WithTest("/spec/template/spec/volumes", []v1.Volume{{
					Name: "existingvol",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testpvcdiskclaim",
						}},
					},
				}}),
				patch.WithTest("/spec/template/spec/domain/devices/disks", []v1.Disk{{Name: "existingvol"}}),
				patch.WithReplace("/spec/template/spec/volumes", []v1.Volume{}),
				patch.WithReplace("/spec/template/spec/domain/devices/disks", []v1.Disk{}),
			)),
	)

	DescribeTable("Should verify volume option", func(volumeRequest *v1.VirtualMachineVolumeRequest, existingVolumes []v1.Volume, expectedError string) {
		err := verifyVolumeOption(existingVolumes, volumeRequest)
		if expectedError != "" {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(expectedError))
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
	},
		Entry("add volume name which already exists should fail",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name:         "vol1",
					Disk:         &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{},
				},
			},
			[]v1.Volume{
				{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
			},
			"Unable to add volume [vol1] because volume with that name already exists"),
		Entry("add volume source which already exists should fail(existing dv)",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "dv1",
					Disk: &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
			},
			[]v1.Volume{
				{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
			},
			"Unable to add volume source [dv1] because it already exists"),
		Entry("add volume which source already exists should fail(existing pvc)",
			&v1.VirtualMachineVolumeRequest{
				AddVolumeOptions: &v1.AddVolumeOptions{
					Name: "pvc1",
					Disk: &v1.Disk{},
					VolumeSource: &v1.HotplugVolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "pvc1",
						}},
					},
				},
			},
			[]v1.Volume{
				{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "pvc1",
						}},
					},
				},
			},
			"Unable to add volume source [pvc1] because it already exists"),
		Entry("remove volume which doesnt exist should fail",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "vol1",
				},
			},
			[]v1.Volume{},
			"Unable to remove volume [vol1] because it does not exist"),
		Entry("remove volume which wasnt hotplugged should fail(existing dv)",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "dv1",
				},
			},
			[]v1.Volume{
				{
					Name: "vol1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
			},
			"Unable to remove volume [vol1] because it is not hotpluggable"),
		Entry("remove volume which wasnt hotplugged should fail(existing cloudInit)",
			&v1.VirtualMachineVolumeRequest{
				RemoveVolumeOptions: &v1.RemoveVolumeOptions{
					Name: "cloudinitdisk",
				},
			},
			[]v1.Volume{
				{
					Name: "cloudinitdisk",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{},
					},
				},
			},
			"Unable to remove volume [cloudinitdisk] because it is not hotpluggable"),
	)
})
