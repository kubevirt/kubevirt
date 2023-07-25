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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package vmstats

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("VM Stats Collector", func() {
	Context("VM status collector", func() {
		var ch chan prometheus.Metric
		var scrapper *prometheusScraper

		createVM := func(status k6tv1.VirtualMachinePrintableStatus, vmLastTransitionsTime time.Time) *k6tv1.VirtualMachine {
			return &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-vm"},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"vm.kubevirt.io/os":       "linux",
								"vm.kubevirt.io/workload": "desktop",
								"vm.kubevirt.io/flavor":   "small",
							},
						},
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Volumes: []k6tv1.Volume{
								{
									Name: "test-volume",
									VolumeSource: k6tv1.VolumeSource{
										PersistentVolumeClaim: &k6tv1.PersistentVolumeClaimVolumeSource{
											PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
												ClaimName: "test-pvc",
											},
										},
									},
								},
							},
						},
					},
				},
				Status: k6tv1.VirtualMachineStatus{
					PrintableStatus: status,
					Conditions: []k6tv1.VirtualMachineCondition{
						{
							Type:               k6tv1.VirtualMachineFailure,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime.Add(-20 * time.Second)),
						},
						{
							Type:               k6tv1.VirtualMachineReady,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime),
						},
						{
							Type:               k6tv1.VirtualMachinePaused,
							Status:             "any",
							Reason:             "any",
							LastTransitionTime: metav1.NewTime(vmLastTransitionsTime.Add(-40 * time.Second)),
						},
					},
				},
			}
		}

		BeforeEach(func() {
			ch = make(chan prometheus.Metric, 5)
			scrapper = &prometheusScraper{ch: ch}
		})

		DescribeTable("Add VM status metrics", func(status k6tv1.VirtualMachinePrintableStatus, metric string) {
			t := time.Now()

			scrapper.updateVMStatusMetrics(createVM(status, t))
			close(ch)

			containsStateMetric := false

			for m := range ch {
				dto := &io_prometheus_client.Metric{}
				m.Write(dto)

				if strings.Contains(m.Desc().String(), metric) {
					containsStateMetric = true
					Expect(*dto.Counter.Value).To(Equal(float64(t.Unix())))
				} else {
					Expect(*dto.Counter.Value).To(BeZero())
				}
			}

			Expect(containsStateMetric).To(BeTrue())
		},
			Entry("Starting VM", k6tv1.VirtualMachineStatusProvisioning, startingMetric),
			Entry("Running VM", k6tv1.VirtualMachineStatusRunning, runningMetric),
			Entry("Migrating VM", k6tv1.VirtualMachineStatusMigrating, migratingMetric),
			Entry("Non running VM", k6tv1.VirtualMachineStatusStopped, nonRunningMetric),
			Entry("Errored VM", k6tv1.VirtualMachineStatusCrashLoopBackOff, errorMetric),
		)

		It("should correctly create kubevirt_vm_pvc_info", func() {
			t := time.Now()
			vm := createVM(k6tv1.VirtualMachineStatusRunning, t)
			vmc := &VMCollector{}
			blockMode := corev1.PersistentVolumeBlock

			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName,
					Namespace: vm.Namespace,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					VolumeName: "test-pv",
					VolumeMode: &(blockMode),
				},
			}
			vmc.pvcInformer, _ = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
			vmc.pvcInformer.GetIndexer().Add(pvc)

			pv := &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv",
				},
				Spec: corev1.PersistentVolumeSpec{
					PersistentVolumeSource: corev1.PersistentVolumeSource{
						CSI: &corev1.CSIPersistentVolumeSource{
							Driver: "test-driver",
							VolumeAttributes: map[string]string{
								"clusterID":     "cluster0",
								"mounter":       "rbd",
								"imageFeatures": "layering",
								"mapOptions":    "krbd:rxbounce",
							},
						},
					},
				},
			}
			vmc.pvInformer, _ = testutils.NewFakeInformerFor(&corev1.PersistentVolume{})
			vmc.pvInformer.GetIndexer().Add(pv)

			scrapper.updatePVInfoMetrics(vmc, createVM(k6tv1.VirtualMachineStatusRunning, t))
			close(ch)

			m := <-ch
			dto := &io_prometheus_client.Metric{}
			m.Write(dto)

			Expect(m.Desc().String()).To(ContainSubstring("kubevirt_vm_persistentvolume_info"))
			Expect(*dto.Gauge.Value).To(Equal(float64(1)))

			for _, pair := range dto.Label {
				switch *pair.Name {
				case "name":
					Expect(*pair.Value).To(Equal("test-vm"))
				case "namespace":
					Expect(*pair.Value).To(Equal("test-ns"))
				case "volumename":
					Expect(*pair.Value).To(Equal("test-pv"))
				case "volumeAttributes":
					Expect(*pair.Value).To(ContainSubstring("clusterID=cluster0;"))
					Expect(*pair.Value).To(ContainSubstring("mounter=rbd;"))
					Expect(*pair.Value).To(ContainSubstring("imageFeatures=layering;"))
					Expect(*pair.Value).To(ContainSubstring("mapOptions=krbd:rxbounce;"))
				}
			}
		})
	})
})
