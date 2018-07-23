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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"flag"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Bridge", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	tests.BeforeAll(func() {
		// create a deamon set for the network bridge device plugin
		// taken from here: https://github.com/kubevirt/kubernetes-device-plugins/blob/master/manifests/bridge-ds.yml
		runNetworkBridgeDevicePlugin := func() {
			const repo = "quay.io/kubevirt"
			const tag = "latest"
			const name = "device-plugin-network-bridge"

			ds := appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: name,
					Labels: map[string]string{
						v1.AppLabel: "test",
						"name":      name,
					},
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"name": name},
					},
					Template: k8sv1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"name": name},
						},
						Spec: k8sv1.PodSpec{
							HostPID:     true,
							HostNetwork: true,
							Containers: []k8sv1.Container{
								{
									Name:  name,
									Image: fmt.Sprintf("%s/%s:%s", repo, name, tag),
									SecurityContext: &k8sv1.SecurityContext{
										Privileged: tests.NewBool(true),
									},
									// instead of using a config map, the values of the env variable
									// are set statically
									Env: []k8sv1.EnvVar{{Name: "BRIDGES", Value: "red,blue"}},
									VolumeMounts: []k8sv1.VolumeMount{
										{Name: "var-run", MountPath: "/var/run"},
										{Name: "device-plugin", MountPath: "/var/lib/kubelet/device-plugins"},
									},
								},
							},
							Volumes: []k8sv1.Volume{
								{
									Name: "var-run", VolumeSource: k8sv1.VolumeSource{
										HostPath: &k8sv1.HostPathVolumeSource{Path: "/var/run"}},
								},
								{
									Name: "device-plugin", VolumeSource: k8sv1.VolumeSource{
										HostPath: &k8sv1.HostPathVolumeSource{Path: "/var/lib/kubelet/device-plugins"}},
								},
							},
						},
					},
				},
			}

			_, err = virtClient.AppsV1().DaemonSets(tests.NamespaceTestDefault).Create(&ds)
			Expect(err).ToNot(HaveOccurred())
		}

		waitForPodToFinish := func(pod *k8sv1.Pod) k8sv1.PodPhase {
			Eventually(func() k8sv1.PodPhase {
				j, err := virtClient.Core().Pods(tests.NamespaceTestDefault).Get(pod.ObjectMeta.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return j.Status.Phase
			}, 30*time.Second, 1*time.Second).Should(Or(Equal(k8sv1.PodSucceeded), Equal(k8sv1.PodFailed)))
			j, err := virtClient.Core().Pods(tests.NamespaceTestDefault).Get(pod.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return j.Status.Phase
		}

		addBridgeToHost := func(name string) {
			// create bridge on the node
			parameters := []string{"link", "add", name, "type", "bridge"}
			job := tests.RenderIPRouteJob(fmt.Sprintf("ip-add-%s", name), parameters)
			job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
			Expect(err).ToNot(HaveOccurred())
			waitForPodToFinish(job)
			// dont check results, as this may fail because bridge is already there
			// if there was any issue with creating the bridges the following "set" command would indicate the failure

			parameters = []string{"link", "set", "dev", name, "up"}
			job = tests.RenderIPRouteJob(fmt.Sprintf("ip-set-%s", name), parameters)
			job, err = virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
			Expect(err).ToNot(HaveOccurred())
			phase := waitForPodToFinish(job)
			Expect(phase).To(Equal(k8sv1.PodSucceeded))
		}

		// add red and blue bridges to host
		addBridgeToHost("red")
		addBridgeToHost("blue")
		// start the network bridge device plugin
		runNetworkBridgeDevicePlugin()
	})

	Context("Exposing interface to the VM via bridge device plugin", func() {
		var vmi *v1.VirtualMachineInstance
		tests.BeforeAll(func() {
			vmi = tests.NewRandomVMIWithResourceNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
				"#!/bin/bash\necho 'hello'\n",
				"red",
				"bridge.network.kubevirt.io/red")

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
		})

		It("Should create 2 interfaces on the VM", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "ip link show eth1 &> /dev/null && echo ok\n"},
				&expect.BExp{R: "ok"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM should be able to connect to the outside world over the default interface", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			defer expecter.Close()
			Expect(err).ToNot(HaveOccurred())

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "curl -o /dev/null -s -w \"%{http_code}\\n\" -k https://google.com\n"},
				&expect.BExp{R: "301"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Exposing multiple interface to the VM via bridge device plugin", func() {
		var vmi *v1.VirtualMachineInstance
		tests.BeforeAll(func() {
			vmi = tests.NewRandomVMIWithResourceNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
				"#!/bin/bash\necho 'hello'\n",
				"red",
				"bridge.network.kubevirt.io/red")

			// add the "blue" interface and network
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces,
				v1.Interface{Name: "blue",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}})
			vmi.Spec.Networks = append(vmi.Spec.Networks,
				v1.Network{Name: "blue",
					NetworkSource: v1.NetworkSource{Resource: &v1.ResourceNetwork{ResourceName: "bridge.network.kubevirt.io/blue"}}})

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
		})

		It("Should create 3 interfaces on the VM", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "ip link show eth2 &> /dev/null && echo ok\n"},
				&expect.BExp{R: "ok"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Let 2 VMs communicate over a private L2 network", func() {
		var vmi1 *v1.VirtualMachineInstance
		var vmi2 *v1.VirtualMachineInstance
		const IP1 = "192.168.1.1"
		const IP2 = "192.168.1.2"

		tests.BeforeAll(func() {
			createVMWithNetworkandIP := func(networkName string, cidr string) (vmi *v1.VirtualMachineInstance) {
				vmi = tests.NewRandomVMIWithResourceNetworkEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros),
					"#!/bin/bash\necho 'hello'\n",
					networkName,
					fmt.Sprintf("bridge.network.kubevirt.io/%s", networkName))

				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)

				// add IP addresses on the interfaces
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()
				out, err := expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: "\\$ "},
					&expect.BSnd{S: fmt.Sprintf("ip addr add %s dev eth1 && echo ok\n", cidr)},
					&expect.BExp{R: "ok"},
				}, 180*time.Second)
				log.Log.Infof("%v", out)
				Expect(err).ToNot(HaveOccurred())
				return
			}

			vmi1 = createVMWithNetworkandIP("red", IP1+"/24")
			vmi2 = createVMWithNetworkandIP("red", IP2+"/24")
		})

		It("VM1 should be able to ping VM2", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi1)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ping %s -I eth1 -q -c 2 > /dev/null && echo ok\n", IP2)},
				&expect.BExp{R: "ok"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VM2 should be able to ping VM1", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi2)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ping %s -I eth1 -q -c 2 > /dev/null && echo ok\n", IP1)},
				&expect.BExp{R: "ok"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Ping should fail over the default interface", func() {
			expecter, err := tests.LoggedInCirrosExpecter(vmi2)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
			out, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("ping %s -q -c 2 > /dev/null; echo $?\n", IP1)},
				&expect.BExp{R: "1"},
			}, 180*time.Second)
			log.Log.Infof("%v", out)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
