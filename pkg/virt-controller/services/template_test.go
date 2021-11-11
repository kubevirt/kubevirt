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

package services

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"kubevirt.io/client-go/api"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"

	"github.com/golang/mock/gomock"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"

	k6tconfig "kubevirt.io/kubevirt/pkg/config"

	testutils2 "kubevirt.io/client-go/testutils"

	v1 "kubevirt.io/api/core/v1"
	fakenetworkclient "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Template", func() {
	var configFactory func(string) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, TemplateService)
	var qemuGid int64 = 107
	var defaultArch = "amd64"

	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	var svc TemplateService

	ctrl := gomock.NewController(GinkgoT())
	virtClient := kubecli.NewMockKubevirtClient(ctrl)

	var config *virtconfig.ClusterConfig
	var kvInformer cache.SharedIndexInformer

	kv := &v1.KubeVirt{
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

	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
	}

	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)
	}

	BeforeEach(func() {
		configFactory = func(cpuArch string) (*virtconfig.ClusterConfig, cache.SharedIndexInformer, TemplateService) {
			config, _, kvInformer := testutils.NewFakeClusterConfigUsingKVWithCPUArch(kv, cpuArch)

			svc = NewTemplateService("kubevirt/virt-launcher",
				240,
				"/var/run/kubevirt",
				"/var/lib/kubevirt",
				"/var/run/kubevirt-ephemeral-disks",
				"/var/run/kubevirt/container-disks",
				"/var/run/kubevirt/hotplug-disks",
				"pull-secret-1",
				pvcCache,
				virtClient,
				config,
				qemuGid,
			)
			// Set up mock clients
			networkClient := fakenetworkclient.NewSimpleClientset()
			virtClient.EXPECT().NetworkClient().Return(networkClient).AnyTimes()
			// Sadly, we cannot pass desired attachment objects into
			// Clientset constructor because UnsafeGuessKindToResource
			// calculates incorrect object kind (without dashes). Instead
			// of that, we use tracker Create function to register objects
			// under explicitly defined schema name
			gvr := schema.GroupVersionResource{
				Group:    "k8s.cni.cncf.io",
				Version:  "v1",
				Resource: "network-attachment-definitions",
			}
			for _, name := range []string{"default", "test1"} {
				network := &networkv1.NetworkAttachmentDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: "default",
					},
				}
				err := networkClient.Tracker().Create(gvr, network, "default")
				Expect(err).To(Not(HaveOccurred()))
			}
			// create a network in a different namespace
			network := &networkv1.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test1",
					Namespace: "other-namespace",
				},
			}
			err := networkClient.Tracker().Create(gvr, network, "other-namespace")
			Expect(err).To(Not(HaveOccurred()))
			return config, kvInformer, svc
		}
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	Describe("Rendering", func() {

		newMinimalWithContainerDisk := func(name string) *v1.VirtualMachineInstance {
			vmi := api.NewMinimalVMI(name)
			vmi.Annotations = map[string]string{v1.NonRootVMIAnnotation: ""}

			volumes := []v1.Volume{
				{
					Name: "containerdisk",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "my-image-1",
						},
					},
				},
			}

			vmi.Spec.Volumes = volumes
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					Name: "containerdisk",
				},
			}

			return vmi
		}

		Context("with NonRoot feature-gate", func() {
			var vmi *v1.VirtualMachineInstance
			BeforeEach(func() {
				vmi = newMinimalWithContainerDisk("random")
			})

			qemu := int64(util.NonRootUID)
			runAsQemuUser := func(container *kubev1.Container) {
				ExpectWithOffset(1, container.SecurityContext.RunAsUser).NotTo(BeNil(), fmt.Sprintf("RunAsUser must be set, %s", container.Name))
				ExpectWithOffset(1, *container.SecurityContext.RunAsUser).To(Equal(qemu))
			}
			runAsNonRootUser := func(container *kubev1.Container) {
				ExpectWithOffset(1, container.SecurityContext.RunAsNonRoot).NotTo(BeNil(), fmt.Sprintf("RunAsNonRoot must be set, %s", container.Name))
				ExpectWithOffset(1, *container.SecurityContext.RunAsNonRoot).To(BeTrue())
			}

			type checkContainerFunc func(*kubev1.Container)

			table.DescribeTable("all containers", func(assertFunc checkContainerFunc) {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).NotTo(HaveOccurred())

				for _, container := range pod.Spec.InitContainers {
					assertFunc(&container)
				}

				for _, container := range pod.Spec.Containers {
					assertFunc(&container)
				}

			},
				table.Entry("run as qemu user", runAsQemuUser),
				table.Entry("run as nonroot user", runAsNonRootUser),
			)

		})
		Context("launch template with correct parameters", func() {
			table.DescribeTable("should check annotations", func(vmiAnnotation, podExpectedAnnotation map[string]string) {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(&v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "testvmi",
						Namespace:   "testns",
						UID:         "1234",
						Annotations: vmiAnnotation,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(pod.ObjectMeta.Annotations).To(Equal(podExpectedAnnotation))
			},
				table.Entry("and don't contain kubectl annotation",
					map[string]string{
						"kubectl.kubernetes.io/last-applied-configuration": "open",
					},
					map[string]string{
						"kubevirt.io/domain":                   "testvmi",
						"pre.hook.backup.velero.io/container":  "compute",
						"pre.hook.backup.velero.io/command":    "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"post.hook.backup.velero.io/container": "compute",
						"post.hook.backup.velero.io/command":   "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"kubevirt.io/migrationTransportUnix":   "true",
					},
				),
				table.Entry("and don't contain kubevirt annotation added by apiserver",
					map[string]string{
						"kubevirt.io/latest-observed-api-version":  "source",
						"kubevirt.io/storage-observed-api-version": ".com",
					},
					map[string]string{
						"kubevirt.io/domain":                   "testvmi",
						"pre.hook.backup.velero.io/container":  "compute",
						"pre.hook.backup.velero.io/command":    "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"post.hook.backup.velero.io/container": "compute",
						"post.hook.backup.velero.io/command":   "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"kubevirt.io/migrationTransportUnix":   "true",
					},
				),
				table.Entry("and contain kubevirt domain annotation",
					map[string]string{
						"kubevirt.io/domain": "fedora",
					},
					map[string]string{
						"kubevirt.io/domain":                   "fedora",
						"pre.hook.backup.velero.io/container":  "compute",
						"pre.hook.backup.velero.io/command":    "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"post.hook.backup.velero.io/container": "compute",
						"post.hook.backup.velero.io/command":   "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"kubevirt.io/migrationTransportUnix":   "true",
					},
				),
				table.Entry("and contain kubernetes annotation",
					map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
					},
					map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
						"kubevirt.io/domain":                             "testvmi",
						"pre.hook.backup.velero.io/container":            "compute",
						"pre.hook.backup.velero.io/command":              "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"post.hook.backup.velero.io/container":           "compute",
						"post.hook.backup.velero.io/command":             "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"kubevirt.io/migrationTransportUnix":             "true",
					},
				),
				table.Entry("and contain kubevirt ignitiondata annotation",
					map[string]string{
						"kubevirt.io/ignitiondata": `{
							"ignition" :  {
								"version": "3"
							 },
						}`,
					},
					map[string]string{
						"kubevirt.io/domain": "testvmi",
						"kubevirt.io/ignitiondata": `{
							"ignition" :  {
								"version": "3"
							 },
						}`,
						"pre.hook.backup.velero.io/container":  "compute",
						"pre.hook.backup.velero.io/command":    "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"post.hook.backup.velero.io/container": "compute",
						"post.hook.backup.velero.io/command":   "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
						"kubevirt.io/migrationTransportUnix":   "true",
					},
				),
			)

			table.DescribeTable("should work", func(arch string, ovmfPath string) {
				config, kvInformer, svc = configFactory(arch)
				trueVar := true
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
					"test":                              "shouldBeInPod",
				}

				pod, err := svc.RenderLaunchManifest(&v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "testvmi",
						Namespace:   "testns",
						UID:         "1234",
						Annotations: annotations,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:       "virt-launcher",
					v1.CreatedByLabel: "1234",
				}))
				Expect(pod.ObjectMeta.Annotations).To(Equal(map[string]string{
					v1.DomainAnnotation:                    "testvmi",
					"test":                                 "shouldBeInPod",
					hooks.HookSidecarListAnnotationName:    `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
					"pre.hook.backup.velero.io/container":  "compute",
					"pre.hook.backup.velero.io/command":    "[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
					"post.hook.backup.velero.io/container": "compute",
					"post.hook.backup.velero.io/command":   "[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"testvmi\", \"--namespace\", \"testns\"]",
					"kubevirt.io/migrationTransportUnix":   "true",
				}))
				Expect(pod.ObjectMeta.OwnerReferences).To(Equal([]metav1.OwnerReference{{
					APIVersion:         v1.VirtualMachineInstanceGroupVersionKind.GroupVersion().String(),
					Kind:               v1.VirtualMachineInstanceGroupVersionKind.Kind,
					Name:               "testvmi",
					UID:                "1234",
					Controller:         &trueVar,
					BlockOwnerDeletion: &trueVar,
				}}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvmi-"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					v1.NodeSchedulable: "true",
				}))

				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/usr/bin/virt-launcher",
					"--qemu-timeout", validateAndExtractQemuTimeoutArg(pod.Spec.Containers[0].Command),
					"--name", "testvmi",
					"--uid", "1234",
					"--namespace", "testns",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks",
					"--container-disk-dir", "/var/run/kubevirt/container-disks",
					"--grace-period-seconds", "45",
					"--hook-sidecars", "1",
					"--ovmf-path", ovmfPath}))
				Expect(pod.Spec.Containers[1].Name).To(Equal("hook-sidecar-0"))
				Expect(pod.Spec.Containers[1].Image).To(Equal("some-image:v1"))
				Expect(pod.Spec.Containers[1].ImagePullPolicy).To(Equal(kubev1.PullPolicy("IfNotPresent")))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
				Expect(len(pod.Spec.InitContainers)).To(Equal(0))
				By("setting the right hostname")
				Expect(pod.Spec.Hostname).To(Equal("testvmi"))
				Expect(pod.Spec.Subdomain).To(BeEmpty())

				hasPodNameEnvVar := false
				for _, ev := range pod.Spec.Containers[0].Env {
					if ev.Name == ENV_VAR_POD_NAME && ev.ValueFrom.FieldRef.FieldPath == "metadata.name" {
						hasPodNameEnvVar = true
						break
					}
				}
				Expect(hasPodNameEnvVar).To(BeTrue())

			},
				table.Entry("on amd64", "amd64", "/usr/share/OVMF"),
				table.Entry("on arm64", "arm64", "/usr/share/AAVMF"),
			)
		})
		Context("with SELinux types", func() {
			It("should run under the SELinux type virt_launcher.process if none specified", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: []v1.Volume{}, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				if pod.Spec.SecurityContext != nil {
					Expect(pod.Spec.SecurityContext.SELinuxOptions).ToNot(BeNil())
					Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal("virt_launcher.process"))
				}
			})
			It("should run under the corresponding SELinux type if specified", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				kvConfig := kv.DeepCopy()
				kvConfig.Spec.Configuration.SELinuxLauncherType = "spc_t"
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: []v1.Volume{}, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.SecurityContext).ToNot(BeNil())
				Expect(pod.Spec.SecurityContext.SELinuxOptions).ToNot(BeNil())
				Expect(pod.Spec.SecurityContext.SELinuxOptions.Type).To(Equal("spc_t"))
			})
			It("should have a level of s0 on all but compute if a type is specified", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				kvConfig := kv.DeepCopy()
				kvConfig.Spec.Configuration.SELinuxLauncherType = "spc_t"
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

				volumes := []v1.Volume{
					{
						Name: "containerdisk",
						VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{
								Image: "my-image-1",
							},
						},
					},
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `
[
  {
    "image": "some-image:v1",
    "imagePullPolicy": "IfNotPresent"
  },
  {
    "image": "another-image:v1",
    "imagePullPolicy": "Always"
  }
]
`,
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
						Annotations: annotations,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: volumes,
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				for _, c := range pod.Spec.Containers {
					if c.Name != "compute" {
						Expect(c.SecurityContext.SELinuxOptions.Level).To(Equal("s0"))
					}
				}
			})
		})
		table.DescribeTable("should check if proper environment variable is ",
			func(debugLogsAnnotationValue string, exceptedValues []string) {
				config, kvInformer, svc = configFactory(defaultArch)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
						Labels: map[string]string{
							debugLogs: debugLogsAnnotationValue,
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pod.Spec.Containers)).To(Equal(1))
				debugLogsValue := ""
				for _, ev := range pod.Spec.Containers[0].Env {
					if ev.Name == ENV_VAR_LIBVIRT_DEBUG_LOGS {
						debugLogsValue = ev.Value
						break
					}
				}
				Expect(exceptedValues).To(ContainElements(debugLogsValue))

			},
			table.Entry("defined when debug annotation is on with lowercase true", "true", []string{"1"}),
			table.Entry("defined when debug annotation is on with mixed case true", "TRuE", []string{"1"}),
			table.Entry("not defined when debug annotation is off", "false", []string{"0", ""}),
		)

		Context("without debug log annotation", func() {
			It("should NOT add the corresponding environment variable", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
						Labels: map[string]string{
							debugLogs: "false",
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pod.Spec.Containers)).To(Equal(1))
				debugLogsValue := ""
				for _, ev := range pod.Spec.Containers[0].Env {
					if ev.Name == ENV_VAR_LIBVIRT_DEBUG_LOGS {
						debugLogsValue = ev.Value
						break
					}
				}
				Expect(debugLogsValue).To(Or(Equal(""), Equal("0")))
			})
		})

		Context("with access credentials", func() {
			It("should add volume with secret referenced by cloud-init user secret ref", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: []v1.Volume{
							{
								Name: "cloud-init-user-data-secret-ref",
								VolumeSource: v1.VolumeSource{
									CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{

										UserData: "somedata",
									},
								},
							},
						},
						AccessCredentials: []v1.AccessCredential{
							{
								SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
									Source: v1.SSHPublicKeyAccessCredentialSource{
										Secret: &v1.AccessCredentialSecretSource{
											SecretName: "my-pkey",
										},
									},
									PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
										ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				volumeFound := false
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "my-pkey-access-cred" {
						volumeFound = true
					}
				}
				Expect(volumeFound).To(BeTrue(), "could not find ssh key secret volume")

				volumeMountFound := false
				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == "my-pkey-access-cred" {
						volumeMountFound = true
					}
				}
				Expect(volumeMountFound).To(BeTrue(), "could not find ssh key secret volume mount")
			})
			It("should add volume with secret referenced by qemu agent access cred", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						AccessCredentials: []v1.AccessCredential{
							{
								SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
									Source: v1.SSHPublicKeyAccessCredentialSource{
										Secret: &v1.AccessCredentialSecretSource{
											SecretName: "my-pkey",
										},
									},
									PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
										QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{},
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				volumeFound := false
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "my-pkey-access-cred" {
						volumeFound = true
					}
				}
				Expect(volumeFound).To(BeTrue(), "could not find ssh key secret volume")

				volumeMountFound := false
				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == "my-pkey-access-cred" {
						volumeMountFound = true
					}
				}
				Expect(volumeMountFound).To(BeTrue(), "could not find ssh key secret volume mount")
			})
			It("should add volume with secret referenced by user/password", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						AccessCredentials: []v1.AccessCredential{
							{
								UserPassword: &v1.UserPasswordAccessCredential{
									Source: v1.UserPasswordAccessCredentialSource{
										Secret: &v1.AccessCredentialSecretSource{
											SecretName: "my-pkey",
										},
									},
									PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
										QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				volumeFound := false
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "my-pkey-access-cred" {
						volumeFound = true
					}
				}
				Expect(volumeFound).To(BeTrue(), "could not find ssh key secret volume")

				volumeMountFound := false
				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == "my-pkey-access-cred" {
						volumeMountFound = true
					}
				}
				Expect(volumeMountFound).To(BeTrue(), "could not find ssh key secret volume mount")
			})
		})

		Context("with cloud-init user secret", func() {
			It("should add volume with secret referenced by cloud-init user secret ref", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "cloud-init-user-data-secret-ref",
								VolumeSource: v1.VolumeSource{
									CloudInitNoCloud: &v1.CloudInitNoCloudSource{
										UserDataSecretRef: &kubev1.LocalObjectReference{
											Name: "some-secret",
										},
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				cloudInitVolumeFound := false
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "cloud-init-user-data-secret-ref-udata" {
						cloudInitVolumeFound = true
					}
				}
				Expect(cloudInitVolumeFound).To(BeTrue(), "could not find cloud init user secret volume")

				cloudInitVolumeMountFound := false
				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == "cloud-init-user-data-secret-ref-udata" {
						cloudInitVolumeMountFound = true
					}
				}
				Expect(cloudInitVolumeMountFound).To(BeTrue(), "could not find cloud init user secret volume mount")
			})
		})
		Context("with cloud-init network data secret", func() {
			It("should add volume with secret referenced by cloud-init network data secret ref", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "cloud-init-network-data-secret-ref",
								VolumeSource: v1.VolumeSource{
									CloudInitNoCloud: &v1.CloudInitNoCloudSource{
										NetworkDataSecretRef: &kubev1.LocalObjectReference{
											Name: "some-secret",
										},
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				cloudInitVolumeFound := false
				for _, volume := range pod.Spec.Volumes {
					if volume.Name == "cloud-init-network-data-secret-ref-ndata" {
						cloudInitVolumeFound = true
					}
				}
				Expect(cloudInitVolumeFound).To(BeTrue(), "could not find cloud init network secret volume")

				cloudInitVolumeMountFound := false
				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					if volumeMount.Name == "cloud-init-network-data-secret-ref-ndata" {
						cloudInitVolumeMountFound = true
					}
				}
				Expect(cloudInitVolumeMountFound).To(BeTrue(), "could not find cloud init network secret volume mount")
			})
		})
		Context("with container disk", func() {

			It("should add init containers to inject binary and pre-pull container disks", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				volumes := []v1.Volume{
					{
						Name: "containerdisk1",
						VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{
								Image: "my-image-1",
							},
						},
					},
					{
						Name: "containerdisk2",
						VolumeSource: v1.VolumeSource{
							ContainerDisk: &v1.ContainerDiskSource{
								Image: "my-image-2",
							},
						},
					},
				}

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.InitContainers)).To(Equal(3))
				Expect(pod.Spec.InitContainers[0].VolumeMounts[0].MountPath).To(Equal("/init/usr/bin"))
				Expect(pod.Spec.InitContainers[0].VolumeMounts[0].Name).To(Equal("virt-bin-share-dir"))
				Expect(pod.Spec.InitContainers[0].Command).To(Equal([]string{"/usr/bin/cp",
					"/usr/bin/container-disk",
					"/init/usr/bin/container-disk",
				}))
				Expect(pod.Spec.InitContainers[0].Image).To(Equal("kubevirt/virt-launcher"))

				Expect(pod.Spec.InitContainers[1].Args).To(Equal([]string{"--no-op"}))
				Expect(pod.Spec.InitContainers[1].Image).To(Equal("my-image-1"))
				Expect(pod.Spec.InitContainers[2].Args).To(Equal([]string{"--no-op"}))
				Expect(pod.Spec.InitContainers[2].Image).To(Equal("my-image-2"))

			})

		})
		Context("migration over unix sockets", func() {
			It("virt-launcher should have a MigrationTransportUnixAnnotation", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := api.NewMinimalVMI("fake-vmi")

				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				_, ok := pod.Annotations[v1.MigrationTransportUnixAnnotation]
				Expect(ok).To(BeTrue())
			})
		})
		Context("with multus annotation", func() {
			It("should add multus networks in the pod annotation", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "test1"},
								}},
							{Name: "other-test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "other-namespace/test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(BeTrue())
				expectedIfaces := ("[" +
					"{\"interface\":\"net1\",\"name\":\"default\",\"namespace\":\"default\"}," +
					"{\"interface\":\"net2\",\"name\":\"test1\",\"namespace\":\"default\"}," +
					"{\"interface\":\"net3\",\"name\":\"test1\",\"namespace\":\"other-namespace\"}" +
					"]")
				Expect(value).To(Equal(expectedIfaces))
			})
			It("should add default multus networks in the multus default-network annotation", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "default", Default: true},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["v1.multus-cni.io/default-network"]
				Expect(ok).To(BeTrue())
				Expect(value).To(Equal("default"))
				value, ok = pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(BeTrue())
				Expect(value).To(Equal("[{\"interface\":\"net1\",\"name\":\"test1\",\"namespace\":\"default\"}]"))
			})
			It("should add MAC address in the pod annotation", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
								Interfaces: []v1.Interface{
									{
										Name: "test1",
										InterfaceBindingMethod: v1.InterfaceBindingMethod{
											SRIOV: &v1.InterfaceSRIOV{},
										},
										MacAddress: "de:ad:00:00:be:af",
									},
								},
							},
						},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(BeTrue())
				expectedIfaces := ("[" +
					"{\"interface\":\"net1\",\"name\":\"default\",\"namespace\":\"default\"}," +
					"{\"interface\":\"net2\",\"mac\":\"de:ad:00:00:be:af\",\"name\":\"test1\",\"namespace\":\"default\"}" +
					"]")
				Expect(value).To(Equal(expectedIfaces))
			})
		})
		Context("with masquerade interface", func() {
			It("should add the istio annotation", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
								Interfaces: []v1.Interface{
									{Name: "default",
										InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations[ISTIO_KUBEVIRT_ANNOTATION]
				Expect(ok).To(BeTrue())
				Expect(value).To(Equal("k6t-eth0"))
			})
		})
		Context("With Istio sidecar.istio.io/inject annotation", func() {
			var (
				vmi v1.VirtualMachineInstance
				pod *kubev1.Pod
			)
			BeforeEach(func() {
				vmi = v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
						Annotations: map[string]string{
							istio.ISTIO_INJECT_ANNOTATION: "true",
						},
					},
				}
				var err error
				pod, err = svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should mount default serviceAccountToken", func() {
				Expect(*pod.Spec.AutomountServiceAccountToken).To(Equal(true))
			})
		})
		Context("with node selectors", func() {
			table.DescribeTable("should add node selectors to template", func(arch string, ovmfPath string) {
				config, kvInformer, svc = configFactory(arch)

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				vmi := v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default", UID: "1234", Annotations: annotations}, Spec: v1.VirtualMachineInstanceSpec{NodeSelector: nodeSelector, Domain: v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}}}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))

				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:       "virt-launcher",
					v1.CreatedByLabel: "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvmi-"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/usr/bin/virt-launcher",
					"--qemu-timeout", validateAndExtractQemuTimeoutArg(pod.Spec.Containers[0].Command),
					"--name", "testvmi",
					"--uid", "1234",
					"--namespace", "default",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks",
					"--container-disk-dir", "/var/run/kubevirt/container-disks",
					"--grace-period-seconds", "45",
					"--hook-sidecars", "1",
					"--ovmf-path", ovmfPath}))
				Expect(pod.Spec.Containers[1].Name).To(Equal("hook-sidecar-0"))
				Expect(pod.Spec.Containers[1].Image).To(Equal("some-image:v1"))
				Expect(pod.Spec.Containers[1].ImagePullPolicy).To(Equal(kubev1.PullPolicy("IfNotPresent")))
				Expect(pod.Spec.Containers[1].VolumeMounts[0].MountPath).To(Equal(hooks.HookSocketsSharedDirectory))

				Expect(pod.Spec.Volumes[0].EmptyDir).ToNot(BeNil())

				Expect(pod.Spec.Containers[0].VolumeMounts[5].MountPath).To(Equal("/var/run/kubevirt/sockets"))

				Expect(pod.Spec.Volumes[1].EmptyDir.Medium).To(Equal(kubev1.StorageMedium("")))

				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
			},
				table.Entry("on amd64", "amd64", "/usr/share/OVMF"),
				table.Entry("on arm64", "arm64", "/usr/share/AAVMF"),
			)

			It("should add node selector for node discovery feature to template", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				enableFeatureGate(virtconfig.CPUNodeDiscoveryGate)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{
								Model: "Conroe",
								Features: []v1.CPUFeature{
									{
										Name:   "lahf_lm",
										Policy: "require",
									},
									{
										Name:   "mmx",
										Policy: "disable",
									},
									{
										Name:   "ssse3",
										Policy: "forbid",
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				cpuModelLabel, err := CPUModelLabelFromCPUModel(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(cpuModelLabel, "true"))

				cpuFeatureLabels := CPUFeatureLabelsFromCPUFeatures(&vmi)
				for _, featureLabel := range cpuFeatureLabels {
					Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(featureLabel, "true"))
				}
			})

			It("should add node selectors from kubevirt-config configMap", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				kvConfig := kv.DeepCopy()
				nodeSelectors := map[string]string{"kubernetes.io/hostname": "node02", "node-role.kubernetes.io/compute": "true"}
				kvConfig.Spec.Configuration.DeveloperConfiguration.NodeSelectors = nodeSelectors
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: []v1.Volume{}, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue("kubernetes.io/hostname", "node02"))
				Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue("node-role.kubernetes.io/compute", "true"))
			})

			It("should not add node selector for hyperv nodes if VMI does not request hyperv features", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				enableFeatureGate(virtconfig.HypervStrictCheckGate)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(NFD_KVM_INFO_PREFIX))))
				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(v1.CPUModelVendorLabel))))
			})

			It("should not add node selector for hyperv nodes if VMI requests hyperv features, but feature gate is disabled", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				enabled := true
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{
									SyNIC: &v1.FeatureState{
										Enabled: &enabled,
									},
									Reenlightenment: &v1.FeatureState{
										Enabled: &enabled,
									},
									EVMCS: &v1.FeatureState{
										Enabled: &enabled,
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(NFD_KVM_INFO_PREFIX))))
				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(v1.CPUModelVendorLabel))))
			})

			It("should add node selector for hyperv nodes if VMI requests hyperv features which depend on host kernel", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				enableFeatureGate(virtconfig.HypervStrictCheckGate)

				enabled := true
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{
									SyNIC: &v1.FeatureState{
										Enabled: &enabled,
									},
									SyNICTimer: &v1.SyNICTimer{
										Enabled: &enabled,
									},
									Frequencies: &v1.FeatureState{
										Enabled: &enabled,
									},
									IPI: &v1.FeatureState{
										Enabled: &enabled,
									},
									EVMCS: &v1.FeatureState{
										Enabled: &enabled,
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"synic", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"synictimer", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"frequencies", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"ipi", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(v1.CPUModelVendorLabel+IntelVendorName, "true"))
			})

			It("should not add node selector for hyperv nodes if VMI requests hyperv features which do not depend on host kernel", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				enableFeatureGate(virtconfig.HypervStrictCheckGate)

				var retries uint32 = 4095
				enabled := true
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{
									Relaxed: &v1.FeatureState{
										Enabled: &enabled,
									},
									VAPIC: &v1.FeatureState{
										Enabled: &enabled,
									},
									Spinlocks: &v1.FeatureSpinlocks{
										Enabled: &enabled,
										Retries: &retries,
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(NFD_KVM_INFO_PREFIX))))
			})

			It("should add default cpu/memory resources to the sidecar container if cpu pinning was requested", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "testvmi",
						Namespace:   "default",
						UID:         "1234",
						Annotations: annotations,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						NodeSelector: nodeSelector,
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{
								Cores:                 2,
								DedicatedCPUPlacement: true,
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				cpu := resource.MustParse("200m")
				mem := resource.MustParse("64M")
				Expect(pod.Spec.Containers[1].Resources.Limits.Memory().Cmp(mem)).To(BeZero())
				Expect(pod.Spec.Containers[1].Resources.Limits.Cpu().Cmp(cpu)).To(BeZero())

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_SYS_NICE {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted SYS_NICE capability")
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(v1.CPUManager, "true"))
			})
			It("should allocate 1 more cpu when isolateEmulatorThread requested", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{
								Cores:                 2,
								DedicatedCPUPlacement: true,
								IsolateEmulatorThread: true,
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				cpu := resource.MustParse("3")
				Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().Cmp(cpu)).To(BeZero())
			})
			It("should add node affinity to pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				nodeAffinity := kubev1.NodeAffinity{}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{NodeAffinity: &nodeAffinity},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{NodeAffinity: &nodeAffinity}))
			})

			It("should add pod affinity to pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				podAffinity := kubev1.PodAffinity{}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{PodAffinity: &podAffinity},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{PodAffinity: &podAffinity}))
			})

			It("should add pod anti-affinity to pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				podAntiAffinity := kubev1.PodAntiAffinity{}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{PodAntiAffinity: &podAntiAffinity},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{PodAntiAffinity: &podAntiAffinity}))
			})

			It("should add tolerations to pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				podToleration := kubev1.Toleration{Key: "test"}
				var tolerationSeconds int64 = 14
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Tolerations: []kubev1.Toleration{
							{
								Key:               podToleration.Key,
								TolerationSeconds: &tolerationSeconds,
							},
						},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Tolerations).To(BeEquivalentTo([]kubev1.Toleration{{Key: podToleration.Key, TolerationSeconds: &tolerationSeconds}}))
			})

			It("should add the scheduler name to the pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						SchedulerName: "test-scheduler",
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.SchedulerName).To(Equal("test-scheduler"))
			})

			It("should use the hostname and subdomain if specified on the vm", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
						Hostname:  "myhost",
						Subdomain: "mydomain",
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Hostname).To(Equal(vmi.Spec.Hostname))
				Expect(pod.Spec.Subdomain).To(Equal(vmi.Spec.Subdomain))
			})

			It("should add vmi labels to pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvmi",
						Namespace: "default",
						UID:       "1234",
						Labels: map[string]string{
							"key1": "val1",
							"key2": "val2",
						},
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Labels).To(Equal(
					map[string]string{
						"key1":            "val1",
						"key2":            "val2",
						v1.AppLabel:       "virt-launcher",
						v1.CreatedByLabel: "1234",
					},
				))
			})

			It("should not add empty affinity to pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeNil())
			})
		})
		Context("with cpu and memory constraints", func() {
			table.DescribeTable("should add cpu and memory constraints to a template", func(arch string, requestMemory string, limitMemory string) {
				config, kvInformer, svc = configFactory(arch)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("1G"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("2m"),
									kubev1.ResourceMemory: resource.MustParse("2G"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1m"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().String()).To(Equal("2m"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal(requestMemory))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal(limitMemory))
			},
				table.Entry("on amd64", "amd64", "1180211045", "2180211045"),
				table.Entry("on arm64", "arm64", "1314428773", "2314428773"),
			)
			table.DescribeTable("should overcommit guest overhead if selected, by only adding the overhead to memory limits", func(arch string, limitMemory string) {
				config, kvInformer, svc = configFactory(arch)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Resources: v1.ResourceRequirements{
								OvercommitGuestOverhead: true,
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("1G"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("2G"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal("1G"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal(limitMemory))
			},
				table.Entry("on amd64", "amd64", "2180211045"),
				table.Entry("on arm64", "arm64", "2314428773"),
			)
			table.DescribeTable("should not add unset resources", func(arch string, requestMemory int) {
				config, kvInformer, svc = configFactory(arch)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1m"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(requestMemory)))

				// Limits for KVM and TUN devices should be requested.
				Expect(pod.Spec.Containers[0].Resources.Limits).ToNot(BeNil())
			},
				table.Entry("on amd64", "amd64", 260),
				table.Entry("on arm64", "arm64", 394),
			)

			table.DescribeTable("should check autoattachGraphicsDevicse", func(arch string, autoAttach *bool, memory int) {
				config, kvInformer, svc = configFactory(arch)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}
				vmi.Spec.Domain.Devices = v1.Devices{
					AutoattachGraphicsDevice: autoAttach,
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memory)))
			},
				table.Entry("and consider graphics overhead if it is not set on amd64", "amd64", nil, 260),
				table.Entry("and consider graphics overhead if it is set to true on amd64", "amd64", True(), 260),
				table.Entry("and not consider graphics overhead if it is set to false on amd64", "amd64", False(), 243),
				table.Entry("and consider graphics overhead if it is not set on arm64", "arm64", nil, 394),
				table.Entry("and consider graphics overhead if it is set to true on arm64", "arm64", True(), 394),
				table.Entry("and not consider graphics overhead if it is set to false on arm64", "arm64", False(), 377),
			)
			It("should calculate vcpus overhead based on guest toplogy", func() {
				config, kvInformer, svc = configFactory(defaultArch)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				coresMemVal := pod.Spec.Containers[0].Resources.Requests.Memory()
				vmi.Spec.Domain.CPU = &v1.CPU{Sockets: 3}
				pod, err = svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				socketsMemVal := pod.Spec.Containers[0].Resources.Requests.Memory()
				Expect(coresMemVal.Cmp(*socketsMemVal)).To(Equal(0))
			})
			It("should calculate vmipod cpu request based on vcpus and cpu_allocation_ratio", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 3},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("300m"))
			})
			It("should allocate equal amount of cpus to vmipod as vcpus with allocation_ratio set to 1", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				kvConfig := kv.DeepCopy()
				kvConfig.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio = 1
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 3},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("3"))
			})
			It("should allocate proportinal amount of cpus to vmipod as vcpus with allocation_ratio set to 10", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 3},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("300m"))
			})
			It("should override the calculated amount of cpus if the user has explicitly specified cpu request", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				kvConfig := kv.DeepCopy()
				kvConfig.Spec.Configuration.DeveloperConfiguration.CPUAllocationRatio = 16
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							CPU: &v1.CPU{Cores: 5},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU: resource.MustParse("150m"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("150m"))
			})
		})

		Context("with hugepages constraints", func() {
			table.DescribeTable("should add to the template constraints ", func(arch, pagesize string, memorySize int) {
				config, kvInformer, svc = configFactory(arch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Memory: &v1.Memory{
								Hugepages: &v1.Hugepages{
									PageSize: pagesize,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memorySize)))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memorySize)))

				hugepageType := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + pagesize)
				hugepagesRequest := pod.Spec.Containers[0].Resources.Requests[hugepageType]
				hugepagesLimit := pod.Spec.Containers[0].Resources.Limits[hugepageType]
				Expect(hugepagesRequest.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))
				Expect(hugepagesLimit.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))

				Expect(len(pod.Spec.Volumes)).To(Equal(8))
				Expect(pod.Spec.Volumes[3].EmptyDir).ToNot(BeNil())
				Expect(pod.Spec.Volumes[3].EmptyDir.Medium).To(Equal(kubev1.StorageMediumHugePages))

				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(7))
				Expect(pod.Spec.Containers[0].VolumeMounts[6].MountPath).To(Equal("/dev/hugepages"))
			},
				table.Entry("hugepages-2Mi on amd64", "amd64", "2Mi", 179),
				table.Entry("hugepages-1Gi on amd64", "amd64", "1Gi", 179),
				table.Entry("hugepages-2Mi on arm64", "arm64", "2Mi", 313),
				table.Entry("hugepages-1Gi on arm64", "arm64", "1Gi", 313),
			)
			table.DescribeTable("should account for difference between guest and container requested memory ", func(arch string, memorySize int) {
				config, kvInformer, svc = configFactory(arch)
				guestMem := resource.MustParse("64M")
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
							Memory: &v1.Memory{
								Hugepages: &v1.Hugepages{
									PageSize: "1Gi",
								},
								Guest: &guestMem,
							},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("70M"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("70M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				guestRequestMemDiff := vmi.Spec.Domain.Resources.Requests.Memory()
				guestRequestMemDiff.Sub(guestMem)

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memorySize) + guestRequestMemDiff.ToDec().ScaledValue(resource.Mega)))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memorySize) + guestRequestMemDiff.ToDec().ScaledValue(resource.Mega)))

				hugepageType := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + "1Gi")
				hugepagesRequest := pod.Spec.Containers[0].Resources.Requests[hugepageType]
				hugepagesLimit := pod.Spec.Containers[0].Resources.Limits[hugepageType]
				Expect(hugepagesRequest.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))
				Expect(hugepagesLimit.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))

				Expect(len(pod.Spec.Volumes)).To(Equal(8))
				Expect(pod.Spec.Volumes[3].EmptyDir).ToNot(BeNil())
				Expect(pod.Spec.Volumes[3].EmptyDir.Medium).To(Equal(kubev1.StorageMediumHugePages))

				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(7))
				Expect(pod.Spec.Containers[0].VolumeMounts[6].MountPath).To(Equal("/dev/hugepages"))
			},
				table.Entry("on amd64", "amd64", 179),
				table.Entry("on arm64", "arm64", 313),
			)
		})

		Context("with file mode pvc source", func() {
			It("should add volume to template", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				namespace := "testns"
				pvcName := "pvcFile"
				pvc := kubev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
				}
				err := pvcCache.Add(&pvc)
				Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")

				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred(), "Render manifest successfully")

				Expect(pod.Spec.Containers[0].VolumeDevices).To(BeEmpty(), "No devices in manifest for 1st container")

				Expect(pod.Spec.Containers[0].VolumeMounts).ToNot(BeEmpty(), "Some mounts in manifest for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(7), "7 mounts in manifest for 1st container")
				Expect(pod.Spec.Containers[0].VolumeMounts[6].Name).To(Equal(volumeName), "1st mount in manifest for 1st container has correct name")

				Expect(pod.Spec.Volumes).ToNot(BeEmpty(), "Found some volumes in manifest")
				Expect(len(pod.Spec.Volumes)).To(Equal(8), "Found 8 volumes in manifest")
				Expect(pod.Spec.Volumes[3].PersistentVolumeClaim).ToNot(BeNil(), "Found PVC volume")
				Expect(pod.Spec.Volumes[3].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "Found PVC volume with correct name")
			})
		})

		Context("with blockdevice mode pvc source", func() {
			It("should add device to template", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				namespace := "testns"
				pvcName := "pvcDevice"
				mode := kubev1.PersistentVolumeBlock
				pvc := kubev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
					Spec: kubev1.PersistentVolumeClaimSpec{
						VolumeMode: &mode,
					},
				}
				err := pvcCache.Add(&pvc)
				Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")
				volumeName := "pvc-volume"
				ephemeralVolumeName := "ephemeral-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}},
						},
					},
					{
						Name: ephemeralVolumeName,
						VolumeSource: v1.VolumeSource{
							Ephemeral: &v1.EphemeralVolumeSource{
								PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred(), "Render manifest successfully")

				Expect(pod.Spec.Containers[0].VolumeDevices).ToNot(BeEmpty(), "Found some devices for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeDevices)).To(Equal(2), "Found 1 device for 1st container")
				Expect(pod.Spec.Containers[0].VolumeDevices[0].Name).To(Equal(volumeName), "Found device for 1st container with correct name")
				Expect(pod.Spec.Containers[0].VolumeDevices[1].Name).To(Equal(ephemeralVolumeName), "Found device for 1st container with correct name")

				Expect(pod.Spec.Containers[0].VolumeMounts).ToNot(BeEmpty(), "Found some mounts in manifest for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(6), "Found 6 mounts in manifest for 1st container")

				Expect(pod.Spec.Volumes).ToNot(BeEmpty(), "Found some volumes in manifest")
				Expect(len(pod.Spec.Volumes)).To(Equal(9), "Found 9 volumes in manifest")
				Expect(pod.Spec.Volumes[3].PersistentVolumeClaim).ToNot(BeNil(), "Found PVC volume")
				Expect(pod.Spec.Volumes[3].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "Found PVC volume with correct name")
			})
		})

		Context("with non existing pvc source", func() {
			It("should result in an error", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				namespace := "testns"
				pvcName := "pvcNotExisting"
				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				_, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).To(HaveOccurred(), "Render manifest results in an error")
				Expect(err).To(BeAssignableToTypeOf(PvcNotFoundError{}), "Render manifest results in an PvsNotFoundError")
			})
		})

		Context("with hotplug volumes", func() {
			It("should render without any hotplug volumes listed in volumeStatus or having `Hotpluggable` flag", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				permanentVolumeName := "permanent-vol"
				hotplugFromSpecName := "hotplug-from-spec"
				hotplugFromStatusName := "hotplug-from-status"
				namespace := "testns"
				volumeNames := []string{hotplugFromSpecName, hotplugFromStatusName, permanentVolumeName}
				volumes := make([]v1.Volume, len(volumeNames))
				for _, name := range volumeNames {
					pvc := kubev1.PersistentVolumeClaim{
						TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
						ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
					}
					err := pvcCache.Add(&pvc)
					Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")

					volumes = append(volumes, v1.Volume{
						Name: name,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: kubev1.PersistentVolumeClaimVolumeSource{
									ClaimName: name,
								},
								Hotpluggable: name == hotplugFromSpecName,
							},
						},
					})
				}

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: volumes,
					},
					Status: v1.VirtualMachineInstanceStatus{
						VolumeStatus: []v1.VolumeStatus{
							{
								Name:          hotplugFromStatusName,
								HotplugVolume: &v1.HotplugVolumeStatus{},
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Name).To(Equal("compute"))
				volumeMountNames := make([]string, len(pod.Spec.Containers[0].VolumeMounts))
				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					volumeMountNames = append(volumeMountNames, volumeMount.Name)
				}
				Expect(volumeMountNames).To(ContainElement(permanentVolumeName))
				Expect(volumeMountNames).ToNot(ContainElements(hotplugFromSpecName, hotplugFromStatusName))
			})
		})

		Context("with launcher's pull secret", func() {
			It("should contain launcher's secret in pod spec", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(1))
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-1"))
			})

		})

		Context("with ContainerDisk pull secrets", func() {
			volumes := []v1.Volume{
				{
					Name: "containerdisk1",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "my-image-1",
							ImagePullSecret: "pull-secret-2",
						},
					},
				},
				{
					Name: "containerdisk2",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "my-image-2",
						},
					},
				},
			}

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}},
			}

			It("should add secret to pod spec", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(2))

				// ContainerDisk secrets come first
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-2"))
				Expect(pod.Spec.ImagePullSecrets[1].Name).To(Equal("pull-secret-1"))
			})

			It("should deduplicate identical secrets", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				volumes[1].VolumeSource.ContainerDisk.ImagePullSecret = "pull-secret-2"

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(2))

				// ContainerDisk secrets come first
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-2"))
				Expect(pod.Spec.ImagePullSecrets[1].Name).To(Equal("pull-secret-1"))
			})

			It("should have compute as first container in the pod", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.Spec.Containers[0].Name).To(Equal("compute"))
				Expect(pod.Spec.Containers[1].Name).To(Equal("volumecontainerdisk1"))
			})
		})

		Context("with sriov interface", func() {

			It("should not run privileged", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				// For Power we are currently running in privileged mode or libvirt will fail to lock memory
				if svc.IsPPC64() {
					Skip("ppc64le is currently running is privileged mode, so skipping test")
				}
				pod, err := svc.RenderLaunchManifest(newVMIWithSriovInterface("testvmi", "1234"))
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(*pod.Spec.Containers[0].SecurityContext.Privileged).To(BeFalse())
			})

			It("should not mount pci related host directories", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(newVMIWithSriovInterface("testvmi", "1234"))
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))

				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					Expect(volumeMount.MountPath).ToNot(Equal("/sys/devices/"))
				}

				for _, volume := range pod.Spec.Volumes {
					if volume.HostPath != nil {
						Expect(volume.HostPath.Path).ToNot(Equal("/sys/devices/"))
					}
				}
			})
			It("should add 1G of memory overhead", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := newVMIWithSriovInterface("testvmi", "1234")
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("1G"),
					},
				}

				pod, err := svc.RenderLaunchManifest(vmi)
				arch := config.GetClusterCPUArch()
				Expect(err).ToNot(HaveOccurred())
				expectedMemory := resource.NewScaledQuantity(0, resource.Kilo)
				expectedMemory.Add(*GetMemoryOverhead(vmi, arch))
				expectedMemory.Add(*vmi.Spec.Domain.Resources.Requests.Memory())
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().Value()).To(Equal(expectedMemory.Value()))
			})
			It("should still add memory overhead for 1 core if cpu topology wasn't provided", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				requirements := v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("512Mi"),
						kubev1.ResourceCPU:    resource.MustParse("150m"),
					},
				}
				vmi := newVMIWithSriovInterface("testvmi1", "1234")
				vmi.Spec.Domain.Resources = requirements

				vmi1 := newVMIWithSriovInterface("testvmi2", "1134")
				vmi1.Spec.Domain.Resources = requirements
				vmi1.Spec.Domain.CPU = &v1.CPU{
					Model: "Conroe",
					Cores: 1,
				}

				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				pod1, err := svc.RenderLaunchManifest(vmi1)
				arch := config.GetClusterCPUArch()
				Expect(err).ToNot(HaveOccurred())
				expectedMemory := resource.NewScaledQuantity(0, resource.Kilo)
				expectedMemory.Add(*GetMemoryOverhead(vmi1, arch))
				expectedMemory.Add(*vmi.Spec.Domain.Resources.Requests.Memory())
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().Value()).To(Equal(expectedMemory.Value()))
				Expect(pod1.Spec.Containers[0].Resources.Requests.Memory().Value()).To(Equal(expectedMemory.Value()))
			})
		})
		Context("with slirp interface", func() {
			It("Should have empty port list in the pod manifest", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				slirpInterface := v1.InterfaceSlirp{}
				domain := v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(0))
			})
			It("Should create a port list in the pod manifest", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				slirpInterface := v1.InterfaceSlirp{}
				ports := []v1.Port{{Name: "http", Port: 80}, {Protocol: "UDP", Port: 80}, {Port: 90}, {Name: "other-http", Port: 80}}
				domain := v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", Ports: ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(4))
				Expect(pod.Spec.Containers[0].Ports[0].Name).To(Equal("http"))
				Expect(pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[0].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[1].Name).To(Equal(""))
				Expect(pod.Spec.Containers[0].Ports[1].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[1].Protocol).To(Equal(kubev1.Protocol("UDP")))
				Expect(pod.Spec.Containers[0].Ports[2].Name).To(Equal(""))
				Expect(pod.Spec.Containers[0].Ports[2].ContainerPort).To(Equal(int32(90)))
				Expect(pod.Spec.Containers[0].Ports[2].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[3].Name).To(Equal("other-http"))
				Expect(pod.Spec.Containers[0].Ports[3].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[3].Protocol).To(Equal(kubev1.Protocol("TCP")))
			})
			It("Should create a port list in the pod manifest with multiple interfaces", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				slirpInterface1 := v1.InterfaceSlirp{}
				slirpInterface2 := v1.InterfaceSlirp{}
				ports1 := []v1.Port{{Name: "http", Port: 80}}
				ports2 := []v1.Port{{Name: "other-http", Port: 80}}
				domain := v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}
				domain.Devices.Interfaces = []v1.Interface{
					{Name: "testnet",
						Ports:                  ports1,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface1}},
					{Name: "testnet",
						Ports:                  ports2,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface2}}}

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Ports[0].Name).To(Equal("http"))
				Expect(pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[0].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[1].Name).To(Equal("other-http"))
				Expect(pod.Spec.Containers[0].Ports[1].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[1].Protocol).To(Equal(kubev1.Protocol("TCP")))
			})
		})

		Context("with pod networking", func() {
			It("Should require tun device by default", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				tun, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeTrue())
				Expect(int(tun.Value())).To(Equal(1))

				caps := pod.Spec.Containers[0].SecurityContext.Capabilities

				Expect(caps.Drop).To(ContainElement(kubev1.Capability(CAP_NET_RAW)), "Expected compute container to drop NET_RAW capability")
			})

			It("Should require tun device if explicitly requested", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				domain := v1.DomainSpec{}
				autoAttach := true
				domain.Devices.AutoattachPodInterface = &autoAttach

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				tun, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeTrue())
				Expect(int(tun.Value())).To(Equal(1))

				caps := pod.Spec.Containers[0].SecurityContext.Capabilities

				Expect(caps.Drop).To(ContainElement(kubev1.Capability(CAP_NET_RAW)), "Expected compute container to drop NET_RAW capability")
			})

			It("Should not require tun device if explicitly rejected", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				domain := v1.DomainSpec{}
				autoAttach := false
				domain.Devices.AutoattachPodInterface = &autoAttach

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				_, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeFalse())

				caps := pod.Spec.Containers[0].SecurityContext.Capabilities

				Expect(caps.Drop).To(ContainElement(kubev1.Capability(CAP_NET_RAW)), "Expected compute container to drop NET_RAW capability")
			})
		})

		Context("with a downwardMetrics volume source", func() {

			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}
			})

			It("Should add an empytDir backed by Memory", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi.Spec.Volumes = []v1.Volume{
					{
						Name: "downardMetrics",
						VolumeSource: v1.VolumeSource{
							DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(8))
				Expect(pod.Spec.Volumes[3].EmptyDir).ToNot(BeNil())
				Expect(pod.Spec.Volumes[3].EmptyDir.Medium).To(Equal(kubev1.StorageMediumMemory))
				Expect(pod.Spec.Volumes[3].EmptyDir.SizeLimit.Equal(resource.MustParse("1Mi"))).To(BeTrue())
				Expect(pod.Spec.Containers[0].VolumeMounts[6].MountPath).To(Equal(k6tconfig.DownwardMetricDisksDir))
			})

			It("Should add 1Mi memory overhead", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				overhead := pod.Spec.Containers[0].Resources.Requests.Memory()
				vmi.Spec.Volumes = []v1.Volume{
					{
						Name: "downardMetrics",
						VolumeSource: v1.VolumeSource{
							DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
						},
					},
				}
				pod, err = svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				newOverhead := pod.Spec.Containers[0].Resources.Requests.Memory()
				overhead.Add(resource.MustParse("1Mi"))
				Expect(newOverhead.Equal(*overhead)).To(BeTrue())
			})
		})

		Context("with a configMap volume source", func() {
			It("Should add the ConfigMap to template", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				volumes := []v1.Volume{
					{
						Name: "configmap-volume",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: kubev1.LocalObjectReference{
									Name: "test-configmap",
								},
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(8))
				Expect(pod.Spec.Volumes[3].ConfigMap).ToNot(BeNil())
				Expect(pod.Spec.Volumes[3].ConfigMap.LocalObjectReference.Name).To(Equal("test-configmap"))
			})
		})

		Context("with a Sysprep volume source", func() {
			Context("with a ConfigMap", func() {
				It("Should add the Sysprep ConfigMap to template", func() {
					config, kvInformer, svc = configFactory(defaultArch)
					volumes := []v1.Volume{
						{
							Name: "sysprep-configmap-volume",
							VolumeSource: v1.VolumeSource{
								Sysprep: &v1.SysprepSource{
									ConfigMap: &kubev1.LocalObjectReference{
										Name: "test-sysprep-configmap",
									},
								},
							},
						},
					}
					vmi := v1.VirtualMachineInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "testvmi", Namespace: "default", UID: "1234",
						},
						Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
					}

					pod, err := svc.RenderLaunchManifest(&vmi)
					Expect(err).ToNot(HaveOccurred())

					Expect(pod.Spec.Volumes).ToNot(BeEmpty())
					Expect(len(pod.Spec.Volumes)).To(Equal(9))
					Expect(pod.Spec.Volumes[3].ConfigMap).ToNot(BeNil())
					Expect(pod.Spec.Volumes[3].ConfigMap.LocalObjectReference.Name).To(Equal("test-sysprep-configmap"))
				})
			})
			Context("with a Secret", func() {
				It("Should add the Sysprep SecretRef to template", func() {
					config, kvInformer, svc = configFactory(defaultArch)
					volumes := []v1.Volume{
						{
							Name: "sysprep-configmap-volume",
							VolumeSource: v1.VolumeSource{
								Sysprep: &v1.SysprepSource{
									Secret: &kubev1.LocalObjectReference{
										Name: "test-sysprep-secret",
									},
								},
							},
						},
					}
					vmi := v1.VirtualMachineInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "testvmi", Namespace: "default", UID: "1234",
						},
						Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
					}

					pod, err := svc.RenderLaunchManifest(&vmi)
					Expect(err).ToNot(HaveOccurred())

					Expect(pod.Spec.Volumes).ToNot(BeEmpty())
					Expect(len(pod.Spec.Volumes)).To(Equal(9))
					Expect(pod.Spec.Volumes[3].Secret).ToNot(BeNil())
					Expect(pod.Spec.Volumes[3].Secret.SecretName).To(Equal("test-sysprep-secret"))
				})
			})
		})

		Context("with a secret volume source", func() {
			It("should add the Secret to template", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				volumes := []v1.Volume{
					{
						Name: "secret-volume",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{
								SecretName: "test-secret",
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
						Devices: v1.Devices{
							DisableHotplug: true,
						},
					}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(8))
				Expect(pod.Spec.Volumes[3].Secret).ToNot(BeNil())
				Expect(pod.Spec.Volumes[3].Secret.SecretName).To(Equal("test-secret"))
			})
		})
		Context("with probes", func() {
			var vmi *v1.VirtualMachineInstance
			BeforeEach(func() {
				vmi = &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						ReadinessProbe: &v1.Probe{
							InitialDelaySeconds: 2,
							TimeoutSeconds:      3,
							PeriodSeconds:       4,
							SuccessThreshold:    5,
							FailureThreshold:    6,
							Handler: v1.Handler{
								TCPSocket: &kubev1.TCPSocketAction{
									Port: intstr.Parse("80"),
									Host: "123",
								},
								HTTPGet: &kubev1.HTTPGetAction{
									Path: "test",
								},
							},
						},
						LivenessProbe: &v1.Probe{
							InitialDelaySeconds: 12,
							TimeoutSeconds:      13,
							PeriodSeconds:       14,
							SuccessThreshold:    15,
							FailureThreshold:    16,
							Handler: v1.Handler{
								TCPSocket: &kubev1.TCPSocketAction{
									Port: intstr.Parse("82"),
									Host: "1234",
								},
								HTTPGet: &kubev1.HTTPGetAction{
									Path: "test34",
								},
							},
						},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
							},
						}},
				}
			})
			It("should copy all specified probes", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				livenessProbe := pod.Spec.Containers[0].LivenessProbe
				readinessProbe := pod.Spec.Containers[0].ReadinessProbe
				Expect(livenessProbe.Handler.TCPSocket).To(Equal(vmi.Spec.LivenessProbe.TCPSocket))
				Expect(readinessProbe.Handler.TCPSocket).To(Equal(vmi.Spec.ReadinessProbe.TCPSocket))

				Expect(livenessProbe.Handler.HTTPGet).To(Equal(vmi.Spec.LivenessProbe.HTTPGet))
				Expect(readinessProbe.Handler.HTTPGet).To(Equal(vmi.Spec.ReadinessProbe.HTTPGet))

				Expect(livenessProbe.PeriodSeconds).To(Equal(vmi.Spec.LivenessProbe.PeriodSeconds))
				Expect(livenessProbe.InitialDelaySeconds).To(Equal(vmi.Spec.LivenessProbe.InitialDelaySeconds + LibvirtStartupDelay))
				Expect(livenessProbe.TimeoutSeconds).To(Equal(vmi.Spec.LivenessProbe.TimeoutSeconds))
				Expect(livenessProbe.SuccessThreshold).To(Equal(vmi.Spec.LivenessProbe.SuccessThreshold))
				Expect(livenessProbe.FailureThreshold).To(Equal(vmi.Spec.LivenessProbe.FailureThreshold))

				Expect(readinessProbe.PeriodSeconds).To(Equal(vmi.Spec.ReadinessProbe.PeriodSeconds))
				Expect(readinessProbe.InitialDelaySeconds).To(Equal(vmi.Spec.ReadinessProbe.InitialDelaySeconds + LibvirtStartupDelay))
				Expect(readinessProbe.TimeoutSeconds).To(Equal(vmi.Spec.ReadinessProbe.TimeoutSeconds))
				Expect(readinessProbe.SuccessThreshold).To(Equal(vmi.Spec.ReadinessProbe.SuccessThreshold))
				Expect(readinessProbe.FailureThreshold).To(Equal(vmi.Spec.ReadinessProbe.FailureThreshold))
			})

			It("should not set a readiness probe on the pod, if no one was specified on the vmi", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi.Spec.ReadinessProbe = nil
				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].ReadinessProbe).To(BeNil())
			})
		})

		Context("with GPU device interface", func() {
			It("should not run privileged", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				// For Power we are currently running in privileged mode or libvirt will fail to lock memory
				if svc.IsPPC64() {
					Skip("ppc64le is currently running is privileged mode, so skipping test")
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
								GPUs: []v1.GPU{
									{
										Name:       "gpu1",
										DeviceName: "vendor.com/gpu_name",
									},
								},
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(*pod.Spec.Containers[0].SecurityContext.Privileged).To(BeFalse())
			})
			It("should not mount pci related host directories and should have gpu resource", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
								GPUs: []v1.GPU{
									{
										Name:       "gpu1",
										DeviceName: "vendor.com/gpu_name",
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pod.Spec.Containers)).To(Equal(1))

				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					Expect(volumeMount.MountPath).ToNot(Equal("/sys/devices/"))
				}

				for _, volume := range pod.Spec.Volumes {
					if volume.HostPath != nil {
						Expect(volume.HostPath.Path).ToNot(Equal("/sys/devices/"))
					}
				}

				resources := pod.Spec.Containers[0].Resources
				val, ok := resources.Requests["vendor.com/gpu_name"]
				Expect(ok).To(Equal(true))
				Expect(val).To(Equal(*resource.NewQuantity(1, resource.DecimalSI)))
			})
		})

		Context("with HostDevice device interface", func() {
			It("should not run privileged", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				// For Power we are currently running in privileged mode or libvirt will fail to lock memory
				if svc.IsPPC64() {
					Skip("ppc64le is currently running is privileged mode, so skipping test")
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
								HostDevices: []v1.HostDevice{
									{
										Name:       "hostdev1",
										DeviceName: "vendor.com/dev_name",
									},
								},
							},
						},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(*pod.Spec.Containers[0].SecurityContext.Privileged).To(BeFalse())
			})
			It("should not mount pci related host directories", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								DisableHotplug: true,
								HostDevices: []v1.HostDevice{
									{
										Name:       "hostdev1",
										DeviceName: "vendor.com/dev_name",
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pod.Spec.Containers)).To(Equal(1))

				for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
					Expect(volumeMount.MountPath).ToNot(Equal("/sys/devices/"))
				}

				for _, volume := range pod.Spec.Volumes {
					if volume.HostPath != nil {
						Expect(volume.HostPath.Path).ToNot(Equal("/sys/devices/"))
					}
				}

				resources := pod.Spec.Containers[0].Resources
				val, ok := resources.Requests["vendor.com/dev_name"]
				Expect(ok).To(Equal(true))
				Expect(val).To(Equal(*resource.NewQuantity(1, resource.DecimalSI)))
			})
		})

		Context("with specified priorityClass", func() {
			It("should add priorityClass", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "namespace",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						PriorityClassName: "test",
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.PriorityClassName).To(Equal("test"))
			})

		})

		Context("Ephemeral storage request", func() {

			table.DescribeTable("by verifying that ephemeral storage ", func(defineEphemeralStorageLimit bool) {
				vmi := api.NewMinimalVMI("fake-vmi")

				ephemeralStorageRequests := resource.MustParse("30M")
				ephemeralStorageLimit := resource.MustParse("70M")
				ephemeralStorageAddition := resource.MustParse(ephemeralStorageOverheadSize)

				if defineEphemeralStorageLimit {
					vmi.Spec.Domain.Resources = v1.ResourceRequirements{
						Requests: kubev1.ResourceList{
							kubev1.ResourceEphemeralStorage: ephemeralStorageRequests,
						},
						Limits: kubev1.ResourceList{
							kubev1.ResourceEphemeralStorage: ephemeralStorageLimit,
						},
					}
				} else {
					vmi.Spec.Domain.Resources = v1.ResourceRequirements{
						Requests: kubev1.ResourceList{
							kubev1.ResourceEphemeralStorage: ephemeralStorageRequests,
						},
					}
				}

				ephemeralStorageRequests.Add(ephemeralStorageAddition)
				ephemeralStorageLimit.Add(ephemeralStorageAddition)

				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())

				computeContainer := pod.Spec.Containers[0]
				Expect(computeContainer.Name).To(Equal("compute"))

				if defineEphemeralStorageLimit {
					Expect(computeContainer.Resources.Requests).To(HaveKeyWithValue(kubev1.ResourceEphemeralStorage, ephemeralStorageRequests))
					Expect(computeContainer.Resources.Limits).To(HaveKeyWithValue(kubev1.ResourceEphemeralStorage, ephemeralStorageLimit))
				} else {
					Expect(computeContainer.Resources.Requests).To(HaveKeyWithValue(kubev1.ResourceEphemeralStorage, ephemeralStorageRequests))
					Expect(computeContainer.Resources.Limits).To(Not(HaveKey(kubev1.ResourceEphemeralStorage)))
				}
			},
				table.Entry("request is increased to consist non-user ephemeral storage", false),
				table.Entry("request and limit is increased to consist non-user ephemeral storage", true),
			)

		})

		Context("with kernel boot", func() {
			hasContainerWithName := func(containers []kubev1.Container, name string) bool {
				for _, container := range containers {
					if strings.Contains(container.Name, name) {
						return true
					}
				}
				return false
			}

			It("should define containers and volumes properly", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := utils.GetVMIKernelBoot()
				vmi.ObjectMeta = metav1.ObjectMeta{
					Name: "testvmi-kernel-boot", Namespace: "default", UID: "1234",
				}

				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).To(BeNil())
				Expect(pod).ToNot(BeNil())

				containers := pod.Spec.Containers
				initContainers := pod.Spec.InitContainers

				Expect(hasContainerWithName(initContainers, "container-disk-binary")).To(BeTrue())
				Expect(hasContainerWithName(initContainers, "kernel-boot")).To(BeTrue())
				Expect(hasContainerWithName(containers, "kernel-boot")).To(BeTrue())
			})
		})

		Context("Using defaultRuntimeClass", func() {
			It("Should set a runtimeClassName on launcher pod, if configured", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				runtimeClassName := "customRuntime"
				kvConfig := kv.DeepCopy()
				kvConfig.Spec.Configuration.DefaultRuntimeClass = runtimeClassName
				testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "namespace",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(*pod.Spec.RuntimeClassName).To(BeEquivalentTo(runtimeClassName))
			})

			It("Should leave runtimeClassName unset on pod, if not configured", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "namespace",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.RuntimeClassName).To(BeNil())
			})
		})

		table.DescribeTable("should require NET_BIND_SERVICE", func(interfaceType string) {
			vmi := api.NewMinimalVMI("fake-vmi")
			switch interfaceType {
			case "bridge":
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			case "masquerade":
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
			case "slirp":
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultSlirpNetworkInterface()}
			}

			pod, err := svc.RenderLaunchManifest(vmi)
			Expect(err).ToNot(HaveOccurred())

			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					Expect(container.SecurityContext.Capabilities.Add).To(ContainElement(kubev1.Capability("NET_BIND_SERVICE")))
					return
				}
			}
			Expect(false).To(BeTrue())
		},
			table.Entry("when there is bridge interface", "bridge"),
			table.Entry("when there is masquerade interface", "masquerade"),
			table.Entry("when there is slirp interface", "slirp"),
		)

		It("should not require NET_BIND_SERVICE", func() {
			vmi := api.NewMinimalVMI("fake-vmi")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMacvtapNetworkInterface("test")}

			pod, err := svc.RenderLaunchManifest(vmi)
			Expect(err).ToNot(HaveOccurred())

			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					Expect(container.SecurityContext.Capabilities.Add).NotTo(ContainElement(kubev1.Capability("NET_BIND_SERVICE")))
					return
				}
			}
			Expect(false).To(BeTrue())
		})

		It("Should run as non-root except compute", func() {
			vmi := newMinimalWithContainerDisk("ranom")

			pod, err := svc.RenderLaunchManifest(vmi)
			Expect(err).NotTo(HaveOccurred())

			for _, container := range pod.Spec.InitContainers {
				Expect(*container.SecurityContext.RunAsNonRoot).To(BeTrue())
				Expect(*container.SecurityContext.RunAsUser).To(Equal(int64(107)))
			}

			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					continue
				}
				Expect(*container.SecurityContext.RunAsNonRoot).To(BeTrue())
				Expect(*container.SecurityContext.RunAsUser).To(Equal(int64(107)))
			}
		})

		Context("With a realtime workload", func() {
			It("should calculate the overhead memory including the requested memory", func() {
				config, kvInformer, svc = configFactory(defaultArch)
				vmi := newMinimalWithContainerDisk("testvmi")
				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("1G"),
						kubev1.ResourceCPU:    resource.MustParse("1"),
					},
					Limits: kubev1.ResourceList{
						kubev1.ResourceMemory: resource.MustParse("1G"),
						kubev1.ResourceCPU:    resource.MustParse("1"),
					},
				}
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 1,
					Sockets:               1,
					Threads:               1,
					DedicatedCPUPlacement: true,
					NUMA:                  &v1.NUMA{},
					IsolateEmulatorThread: true,
					Realtime:              &v1.Realtime{},
				}

				pod, err := svc.RenderLaunchManifest(vmi)
				arch := config.GetClusterCPUArch()
				Expect(err).ToNot(HaveOccurred())
				expectedMemory := resource.NewScaledQuantity(0, resource.Kilo)
				expectedMemory.Add(*GetMemoryOverhead(vmi, arch))
				expectedMemory.Add(*vmi.Spec.Domain.Resources.Requests.Memory())
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().Value()).To(Equal(expectedMemory.Value()))
			})
		})
	})

	Describe("ServiceAccountName", func() {

		It("Should add service account if present", func() {
			config, kvInformer, svc = configFactory(defaultArch)
			serviceAccountName := "testAccount"
			volumes := []v1.Volume{
				{
					Name: "serviceaccount-volume",
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: serviceAccountName,
						},
					},
				},
			}
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}},
			}

			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(Equal(serviceAccountName), "ServiceAccount matches")
			Expect(*pod.Spec.AutomountServiceAccountToken).To(BeTrue(), "Token automount is enabled")
		})

		It("Should not add service account if not present", func() {
			config, kvInformer, svc = configFactory(defaultArch)
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{
					Devices: v1.Devices{
						DisableHotplug: true,
					},
				}},
			}

			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(BeEmpty(), "ServiceAccount is empty")
			Expect(*pod.Spec.AutomountServiceAccountToken).To(BeFalse(), "Token automount is disabled")
		})

	})
})

var _ = Describe("getResourceNameForNetwork", func() {
	It("should return empty string when resource name is not specified", func() {
		network := &networkv1.NetworkAttachmentDefinition{}
		Expect(getResourceNameForNetwork(network)).To(Equal(""))
	})

	It("should return resource name if specified", func() {
		network := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					MULTUS_RESOURCE_NAME_ANNOTATION: "fake.com/fakeResource",
				},
			},
		}
		Expect(getResourceNameForNetwork(network)).To(Equal("fake.com/fakeResource"))
	})
})

var _ = Describe("getNamespaceAndNetworkName", func() {
	It("should return vmi namespace when namespace is implicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		namespace, networkName := getNamespaceAndNetworkName(vmi, "testnet")
		Expect(namespace).To(Equal("testns"))
		Expect(networkName).To(Equal("testnet"))
	})

	It("should return namespace from networkName when namespace is explicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		namespace, networkName := getNamespaceAndNetworkName(vmi, "otherns/testnet")
		Expect(namespace).To(Equal("otherns"))
		Expect(networkName).To(Equal("testnet"))
	})
})

var _ = Describe("requestResource", func() {
	It("should register resource in limits and requests", func() {
		resources := kubev1.ResourceRequirements{}
		resources.Requests = make(kubev1.ResourceList)
		resources.Limits = make(kubev1.ResourceList)

		resource := "intel.com/sriov"
		resourceName := kubev1.ResourceName(resource)

		for i := int64(1); i <= 5; i++ {
			requestResource(&resources, resource)

			val, ok := resources.Limits[resourceName]
			Expect(ok).To(BeTrue())

			valInt, isInt := val.AsInt64()
			Expect(isInt).To(BeTrue())
			Expect(valInt).To(Equal(i))

			val, ok = resources.Requests[resourceName]
			Expect(ok).To(BeTrue())

			valInt, isInt = val.AsInt64()
			Expect(isInt).To(BeTrue())
			Expect(valInt).To(Equal(i))
		}
	})
})

func newVMIWithSriovInterface(name, uid string) *v1.VirtualMachineInstance {
	sriovInterface := v1.Interface{
		Name: "sriov-nic",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			SRIOV: &v1.InterfaceSRIOV{},
		},
	}
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(uid),
		},
	}
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{sriovInterface}

	return vmi
}

func True() *bool {
	b := true
	return &b
}

func False() *bool {
	b := false
	return &b
}

func TestTemplate(t *testing.T) {
	testutils2.KubeVirtTestSuiteSetup(t)
}

func validateAndExtractQemuTimeoutArg(args []string) string {
	timeoutString := ""
	for i, arg := range args {
		if arg == "--qemu-timeout" {
			timeoutString = args[i+1]
			break
		}
	}

	Expect(timeoutString).ToNot(Equal(""))

	timeoutInt, err := strconv.Atoi(strings.TrimSuffix(timeoutString, "s"))
	Expect(err).To(BeNil())

	qemuTimeoutBaseSeconds := 240

	failMsg := ""
	if timeoutInt < qemuTimeoutBaseSeconds {
		failMsg = fmt.Sprintf("randomized qemu timeout [%d] is less that base range [%d]", timeoutInt, qemuTimeoutBaseSeconds)
	} else if timeoutInt > qemuTimeoutBaseSeconds+qemuTimeoutJitterRange {
		failMsg = fmt.Sprintf("randomized qemu timeout [%d] is greater than max range [%d]", timeoutInt, qemuTimeoutBaseSeconds+qemuTimeoutJitterRange)

	}
	Expect(failMsg).To(Equal(""))

	return timeoutString
}
