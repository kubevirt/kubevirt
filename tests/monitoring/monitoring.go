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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

const (
	virtOperatorDeploymentName = "virt-operator"
)

var _ = Describe("[Serial][sig-monitoring]Prometheus Alerts", func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var scales map[string]*autoscalingv1.Scale

	backupScale := func(operatorName string) {
		Eventually(func() error {
			virtOperatorCurrentScale, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).GetScale(context.TODO(), operatorName, metav1.GetOptions{})
			if err == nil {
				scales[operatorName] = virtOperatorCurrentScale
			}
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	revertScale := func(operatorName string) {
		revert := scales[operatorName].DeepCopy()
		revert.ResourceVersion = ""
		Eventually(func() error {
			_, err = virtClient.
				AppsV1().
				Deployments(flags.KubeVirtInstallNamespace).
				UpdateScale(context.TODO(), operatorName, revert, metav1.UpdateOptions{})
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	updateScale := func(operatorName string, replicas int32) {
		scale := scales[operatorName].DeepCopy()
		scale.Spec.Replicas = replicas
		Eventually(func() error {
			_, err = virtClient.
				AppsV1().
				Deployments(flags.KubeVirtInstallNamespace).
				UpdateScale(context.TODO(), operatorName, scale, metav1.UpdateOptions{})
			return err
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	verifyAlertExist := func(alertName string) {
		Eventually(func() error {
			alerts, err := getAlerts(virtClient)
			if err != nil {
				return err
			}
			for _, alert := range alerts {
				if string(alert.Labels["alertname"]) == alertName {
					return nil
				}
			}
			return fmt.Errorf("alert doesn't exist: %v", alertName)
		}, 120*time.Second, 1*time.Second).Should(BeNil())

	}

	waitUntilThereIsNoAlert := func() {
		Eventually(func() error {
			alerts, err := getAlerts(virtClient)
			if err != nil {
				return err
			}
			if len(alerts) == 0 {
				return nil
			}
			return fmt.Errorf("some alerts exist: %v", alerts)
		}, 120*time.Second, 1*time.Second).Should(BeNil())

	}

	waitForMetricValue := func(client kubecli.KubevirtClient, metric string, expectedValue int64) {
		Eventually(func() int {
			v, err := getMetricValue(client, metric)
			if err != nil {
				return -1
			}
			i, err := strconv.Atoi(v)
			Expect(err).ToNot(HaveOccurred())
			return i
		}, 3*time.Minute, 1*time.Second).Should(BeNumerically("==", expectedValue))
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtClient).ToNot(BeNil())

		checks.SkipIfPrometheusRuleIsNotEnabled(virtClient)
		tests.BeforeTestCleanup()
	})

	Context("Up metrics", func() {
		BeforeEach(func() {
			scales = make(map[string]*autoscalingv1.Scale, 1)
			backupScale(virtOperatorDeploymentName)
		})

		AfterEach(func() {
			revertScale(virtOperatorDeploymentName)
			waitUntilThereIsNoAlert()
		})

		It("VirtOperatorDown should be triggered when virt-operator is down", func() {
			By("By scaling virt-operator to zero")
			updateScale(virtOperatorDeploymentName, int32(0))
			verifyAlertExist("VirtOperatorDown")
		})
	})

	Context("Alerts runbooks", func() {
		It("Should have available URLs", func() {
			alerts, err := getAlerts(virtClient)
			Expect(err).ToNot(HaveOccurred())
			for _, alert := range alerts {
				Expect(alert.Annotations).ToNot(BeNil())
				url, ok := alert.Annotations["runbook_url"]
				Expect(ok).To(BeTrue())
				resp, err := http.Head(string(url))
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).Should(Equal(http.StatusOK))
			}
		})
	})

	Context("VM snapshot metrics", func() {
		quantity, _ := resource.ParseQuantity("500Mi")

		createSimplePVCWithRestoreLabels := func(name string) {
			_, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Create(context.Background(), &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"restore.kubevirt.io/source-vm-name":      "simple-vm",
						"restore.kubevirt.io/source-vm-namespace": util.NamespaceTestDefault,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": quantity,
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		It("Number of disks restored and total restored bytes metric values should be correct", func() {
			totalMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source_total{vm_name='simple-vm',vm_namespace='%s'}", util.NamespaceTestDefault)
			bytesMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source_bytes{vm_name='simple-vm',vm_namespace='%s'}", util.NamespaceTestDefault)
			numPVCs := 2

			for i := 1; i < numPVCs+1; i++ {
				// Create dummy PVC that is labelled as "restored" from VM snapshot
				createSimplePVCWithRestoreLabels(fmt.Sprintf("vmsnapshot-restored-pvc-%d", i))
				// Metric values increases per restored disk
				waitForMetricValue(virtClient, totalMetric, int64(i))
				waitForMetricValue(virtClient, bytesMetric, quantity.Value()*int64(i))
			}
		})
	})
})
