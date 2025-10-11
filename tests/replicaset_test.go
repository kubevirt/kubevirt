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

package tests_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/libvmi/replicaset"
	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autov1 "k8s.io/api/autoscaling/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/tests/libreplicaset"
	"kubevirt.io/kubevirt/tests/libvmifact"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachineInstanceReplicaSet", decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	doScale := func(name string, scale int32) {

		By(fmt.Sprintf("Scaling to %d", scale))
		rs, err := virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Patch(context.Background(), name, types.JSONPatchType, []byte(fmt.Sprintf("[{ \"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": %v }]", scale)), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the number of replicas")
		Eventually(func() int32 {
			rs, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(rs)).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(scale)))

		vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(libreplicaset.FilterNotDeletedVMIs(vmis)).To(HaveLen(int(scale)))
	}

	doScaleWithHPA := func(name string, min int32, max int32, expected int32) {

		// Status updates can conflict with our desire to change the spec
		By(fmt.Sprintf("Scaling to %d", min))
		hpa := &autov1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: autov1.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autov1.CrossVersionObjectReference{
					Name:       name,
					Kind:       v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Kind,
					APIVersion: v1.VirtualMachineInstanceReplicaSetGroupVersionKind.GroupVersion().String(),
				},
				MinReplicas: &min,
				MaxReplicas: max,
			},
		}
		_, err := virtClient.AutoscalingV1().HorizontalPodAutoscalers(testsuite.GetTestNamespace(nil)).Create(context.Background(), hpa, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		var s *autov1.Scale
		By("Checking the number of replicas")
		EventuallyWithOffset(1, func() int32 {
			s, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).GetScale(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return s.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(expected)))

		vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, libreplicaset.FilterNotDeletedVMIs(vmis)).To(HaveLen(int(min)))
		err = virtClient.AutoscalingV1().HorizontalPodAutoscalers(testsuite.GetTestNamespace(nil)).Delete(context.Background(), name, metav1.DeleteOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	newReplicaSetWithTemplate := func(template *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceReplicaSet {
		newRS := replicaset.New(template, 0)
		newRS, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(template)).Create(context.Background(), newRS, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return newRS
	}

	newReplicaSet := func() *v1.VirtualMachineInstanceReplicaSet {
		By("Create a new VirtualMachineInstance replica set")
		template := libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()))
		return newReplicaSetWithTemplate(template)
	}

	DescribeTable("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component]should scale", func(startScale int, stopScale int) {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, int32(startScale))
		doScale(newRS.ObjectMeta.Name, int32(stopScale))
		doScale(newRS.ObjectMeta.Name, int32(0))

	},
		Entry("[test_id:1405]to three, to two and then to zero replicas", 3, 2),
		Entry("[test_id:1406]to five, to six and then to zero replicas", 5, 6),
	)

	DescribeTable("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component]should scale with scale subresource", func(startScale int, stopScale int) {
		newRS := newReplicaSet()
		libreplicaset.DoScaleWithScaleSubresource(virtClient, newRS.ObjectMeta.Name, int32(startScale))
		libreplicaset.DoScaleWithScaleSubresource(virtClient, newRS.ObjectMeta.Name, int32(stopScale))
		libreplicaset.DoScaleWithScaleSubresource(virtClient, newRS.ObjectMeta.Name, int32(0))
	},
		Entry("[test_id:1407]to three, to two and then to zero replicas", 3, 2),
		Entry("[test_id:1408]to five, to six and then to zero replicas", 5, 6),
	)

	DescribeTable("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component]should scale with the horizontal pod autoscaler", func(startScale int, stopScale int) {
		template := libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		newRS := replicaset.New(template, 1)
		newRS, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(newRS)).Create(context.Background(), newRS, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		doScaleWithHPA(newRS.ObjectMeta.Name, int32(startScale), int32(startScale), int32(startScale))
		doScaleWithHPA(newRS.ObjectMeta.Name, int32(stopScale), int32(stopScale), int32(stopScale))
		doScaleWithHPA(newRS.ObjectMeta.Name, int32(1), int32(1), int32(1))

	},
		Entry("[test_id:1409]to three, to two and then to one replicas", 3, 2),
		Entry("[test_id:1410]to five, to six and then to one replicas", 5, 6),
	)

	It("[test_id:1411]should be rejected on POST if spec is invalid", func() {
		newRS := newReplicaSet()
		newRS.TypeMeta = metav1.TypeMeta{
			APIVersion: v1.StorageGroupVersion.String(),
			Kind:       "VirtualMachineInstanceReplicaSet",
		}

		jsonBytes, err := json.Marshal(newRS)
		Expect(err).ToNot(HaveOccurred())

		// change the name of a required field (like domain) so validation will fail
		jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

		result := virtClient.RestClient().Post().Resource("virtualmachineinstancereplicasets").Namespace(testsuite.GetTestNamespace(newRS)).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do(context.Background())

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

	})
	It("[test_id:1412]should reject POST if validation webhoook deems the spec is invalid", func() {
		newRS := newReplicaSet()
		newRS.TypeMeta = metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceReplicaSet",
		}

		// Add a disk that doesn't map to a volume.
		// This should get rejected which tells us the webhook validator is working.
		newRS.Spec.Template.Spec.Domain.Devices.Disks = append(newRS.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})

		result := virtClient.RestClient().Post().Resource("virtualmachineinstancereplicasets").Namespace(testsuite.GetTestNamespace(newRS)).Body(newRS).Do(context.Background())

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		reviewResponse := &metav1.Status{}
		body, _ := result.Raw()
		err = json.Unmarshal(body, reviewResponse)
		Expect(err).ToNot(HaveOccurred())

		Expect(reviewResponse.Details.Causes).To(HaveLen(1))
		Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[2].name"))
	})
	It("[test_id:1413]should update readyReplicas once VMIs are up", func() {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		By("checking the number of ready replicas in the returned yaml")
		Eventually(func() int {
			rs, err := virtClient.ReplicaSet(testsuite.GetTestNamespace(newRS)).Get(context.Background(), newRS.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return int(rs.Status.ReadyReplicas)
		}, 120*time.Second, 1*time.Second).Should(Equal(2))
	})

	It("[test_id:1414]should return the correct data when using server-side printing", func() {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		By("waiting until all VMIs are ready")
		Eventually(func() int {
			rs, err := virtClient.ReplicaSet(testsuite.GetTestNamespace(newRS)).Get(context.Background(), newRS.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return int(rs.Status.ReadyReplicas)
		}, 120*time.Second, 1*time.Second).Should(Equal(2))

		By("checking the output of server-side table printing")
		rawTable, err := virtClient.RestClient().Get().
			RequestURI(fmt.Sprintf("/apis/kubevirt.io/%s/namespaces/%s/virtualmachineinstancereplicasets/%s", v1.ApiLatestVersion, testsuite.GetTestNamespace(newRS), newRS.ObjectMeta.Name)).
			SetHeader("Accept", "application/json;as=Table;v=v1beta1;g=meta.k8s.io, application/json").
			DoRaw(context.Background())

		Expect(err).ToNot(HaveOccurred())
		table := &v1beta1.Table{}
		Expect(json.Unmarshal(rawTable, table)).To(Succeed())
		Expect(table.ColumnDefinitions[0].Name).To(Equal("Name"))
		Expect(table.ColumnDefinitions[0].Type).To(Equal("string"))
		Expect(table.ColumnDefinitions[1].Name).To(Equal("Desired"))
		Expect(table.ColumnDefinitions[1].Type).To(Equal("integer"))
		Expect(table.ColumnDefinitions[2].Name).To(Equal("Current"))
		Expect(table.ColumnDefinitions[2].Type).To(Equal("integer"))
		Expect(table.ColumnDefinitions[3].Name).To(Equal("Ready"))
		Expect(table.ColumnDefinitions[3].Type).To(Equal("integer"))
		Expect(table.ColumnDefinitions[4].Name).To(Equal("Age"))
		Expect(table.ColumnDefinitions[4].Type).To(Equal("date"))
		Expect(table.Rows[0].Cells[0].(string)).To(Equal(newRS.ObjectMeta.Name))
		Expect(table.Rows[0].Cells[1]).To(BeNumerically("==", 2))
		Expect(table.Rows[0].Cells[2]).To(BeNumerically("==", 2))
		Expect(table.Rows[0].Cells[3]).To(BeNumerically("==", 2))
	})

	It("[test_id:1415]should remove VMIs once they are marked for deletion", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)
		// Delete it
		By("Deleting the VirtualMachineInstance replica set")
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Delete(context.Background(), newRS.ObjectMeta.Name, metav1.DeleteOptions{})).To(Succeed())
		// Wait until VMIs are gone
		By("Waiting until all VMIs are gone")
		Eventually(func() int {
			vmis, err := virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vmis.Items)
		}, 120*time.Second, 1*time.Second).Should(BeZero())
	})

	It("[test_id:1416]should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)

		// Check for owner reference
		vmis, err := virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(vmis.Items).To(HaveLen(2))
		Expect(err).ToNot(HaveOccurred())
		for _, vmi := range vmis.Items {
			Expect(vmi.OwnerReferences).ToNot(BeEmpty())
		}

		// Delete it
		By("Deleting the VirtualMachineInstance replica set with the 'orphan' deletion strategy")
		orphanPolicy := metav1.DeletePropagationOrphan
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).
			Delete(context.Background(), newRS.ObjectMeta.Name, metav1.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
		// Wait until the replica set is deleted
		By("Waiting until the replica set got deleted")
		Eventually(func() error {
			_, err := virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Get(context.Background(), newRS.ObjectMeta.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Checking if two VMIs are orphaned and still exist")
		vmis, err = virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(vmis.Items).To(HaveLen(2))

		By("Checking a VirtualMachineInstance owner references")
		for _, vmi := range vmis.Items {
			Expect(vmi.OwnerReferences).To(BeEmpty())
		}
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_id:1417]should not scale when paused and scale when resume", func() {
		rs := newReplicaSet()
		// pause controller
		By("Pausing the replicaset")
		_, err := virtClient.ReplicaSet(rs.Namespace).Patch(context.Background(), rs.Name, types.JSONPatchType, []byte("[{ \"op\": \"add\", \"path\": \"/spec/paused\", \"value\": true }]"), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() *v1.VirtualMachineInstanceReplicaSet {
			rs, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(rs)).Get(context.Background(), rs.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs
		}, 10*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReplicaSetReplicaPaused))

		// set new replica count while still being paused
		By("Updating the number of replicas")
		patchData, err := patch.GenerateTestReplacePatch("/spec/replicas", rs.Spec.Replicas, pointer.P(2))
		Expect(err).ToNot(HaveOccurred())
		rs, err = virtClient.ReplicaSet(rs.Namespace).Patch(context.Background(), rs.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		// make sure that we don't scale up
		By("Checking that the replicaset do not scale while it is paused")
		Consistently(func() int32 {
			rs, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(rs)).Get(context.Background(), rs.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Make sure that no failure happened, so that ensure that we don't scale because we are paused
			Expect(rs.Status.Conditions).To(HaveLen(1))
			return rs.Status.Replicas
		}, 3*time.Second, 1*time.Second).Should(Equal(int32(0)))

		// resume controller
		By("Resuming the replicaset")
		_, err = virtClient.ReplicaSet(rs.Namespace).Patch(context.Background(), rs.Name, types.JSONPatchType, []byte("[{ \"op\": \"replace\", \"path\": \"/spec/paused\", \"value\": false }]"), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Paused condition should disappear
		By("Checking that the pause condition disappeared from the replicaset")
		Eventually(func() int {
			rs, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(rs)).Get(context.Background(), rs.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(rs.Status.Conditions)
		}, 10*time.Second, 1*time.Second).Should(Equal(0))

		// Replicas should be created
		By("Checking that the missing replicas are now created")
		Eventually(func() int32 {
			rs, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(rs)).Get(context.Background(), rs.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs.Status.Replicas
		}, 10*time.Second, 1*time.Second).Should(Equal(int32(2)))
	})

	It("[test_id:1418] should replace finished VMIs", func() {
		By("Creating new replica set")
		rs := newReplicaSet()
		doScale(rs.ObjectMeta.Name, int32(2))

		vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis.Items).ToNot(BeEmpty())

		vmi := &vmis.Items[0]
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
		Expect(err).ToNot(HaveOccurred())

		By("Deleting one of the RS VMS pods")
		err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(rs)).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking that the VM disappeared")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			return err
		}, 120*time.Second, time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Checking number of RS VM's to see that we got a replacement")
		vmis, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis.Items).Should(HaveLen(2))
	})

	It("should replace a VMI immediately when a virt-launcher pod gets deleted", func() {
		By("Creating new replica set")
		template := libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithTerminationGracePeriod(200),
		)
		rs := newReplicaSetWithTemplate(template)

		// ensure that the shutdown will take as long as possible

		doScale(rs.ObjectMeta.Name, int32(2))

		vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis.Items).ToNot(BeEmpty())

		By("Waiting until the VMIs are running")
		Eventually(func() int {
			vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(filterRunningVmis(vmis))
		}, 40*time.Second, time.Second).Should(Equal(2))

		vmi := &vmis.Items[0]
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
		Expect(err).ToNot(HaveOccurred())

		By("Deleting one of the RS VMS pods, which will take some time to really stop the VMI")
		err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(rs)).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking that then number of VMIs increases to three")
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() int {
			vmis, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vmis.Items)
		}, 20*time.Second, time.Second).Should(Equal(3))

		By("Checking that the shutting donw VMI is still running, reporting the pod deletion and being marked for deletion")
		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(rs)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		Expect(controller.NewVirtualMachineInstanceConditionManager().
			HasConditionWithStatusAndReason(
				vmi,
				v1.VirtualMachineInstanceConditionType(k8sv1.PodReady),
				k8sv1.ConditionFalse,
				v1.PodTerminatingReason,
			),
		).To(BeTrue())
		Expect(vmi.DeletionTimestamp).ToNot(BeNil())
	})

	Context("replicaset with topology spread constraints", decorators.WgS390x, func() {
		It("Replicas should be spread across nodes", func() {
			nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some schedulable nodes")
			numNodes := len(nodes.Items)
			if numNodes < 2 {
				Skip("Skipping spec if test environment has less than two schedulable nodes")
			}
			vmLabelKey := "test" + rand.String(5)
			vmLabelValue := "test" + rand.String(5)
			vmi := libvmifact.NewAlpine()
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       k8sv1.LabelHostname,
					WhenUnsatisfiable: "DoNotSchedule",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							vmLabelKey: vmLabelValue,
						},
					},
				},
			}

			By("Creating a VirtualMachineInstanceReplicaSet")
			replicas := int32(numNodes)
			// limit the number of replicas launched for this test
			if replicas > 10 {
				replicas = 10
			}
			rs := &v1.VirtualMachineInstanceReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: "replicaset" + rand.String(5)},
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{vmLabelKey: vmLabelValue},
					},
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{vmLabelKey: vmLabelValue},
							Name:   vmi.ObjectMeta.Name,
						},
						Spec: vmi.Spec,
					},
				},
			}
			createdRs, err := kubevirt.Client().ReplicaSet(vmi.Namespace).Create(context.Background(), rs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should create replicaset")

			By("Ensuring that all VMIs are ready")
			Eventually(func() int32 {
				rs, err := kubevirt.Client().ReplicaSet(vmi.Namespace).Get(context.Background(), createdRs.ObjectMeta.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return rs.Status.ReadyReplicas
			}, 120*time.Second, 1*time.Second).Should(Equal(replicas))

			By("Ensuring that VMI replicas are scheduled to separate nodes")
			vmiSet, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", vmLabelKey, vmLabelValue),
			})
			Expect(err).ToNot(HaveOccurred(), "Should find VMIs by label")
			Expect(vmiSet.Items).To(HaveLen(int(replicas)), "Should get expected number of VMIs")

			nodeNames := make(map[string]bool)
			for _, vmi := range vmiSet.Items {
				nodeName := vmi.Status.NodeName
				_, nodeHasReplica := nodeNames[nodeName]
				Expect(nodeHasReplica).To(BeFalse(), "Multiple replicas should not be scheduled to the same node")
				nodeNames[nodeName] = true
			}
		})
	})
})

func filterRunningVmis(vmis *v1.VirtualMachineInstanceList) (running []v1.VirtualMachineInstance) {
	for _, vmi := range vmis.Items {
		if vmi.DeletionTimestamp == nil && vmi.Status.Phase == v1.Running {
			running = append(running, vmi)
		}
	}
	return
}
