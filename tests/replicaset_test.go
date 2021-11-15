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

package tests_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	autov1 "k8s.io/api/autoscaling/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/tests/libreplicaset"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[Serial][rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachineInstanceReplicaSet", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	doScale := func(name string, scale int32) {

		By(fmt.Sprintf("Scaling to %d", scale))
		rs, err := virtClient.ReplicaSet(util.NamespaceTestDefault).Patch(name, types.JSONPatchType, []byte(fmt.Sprintf("[{ \"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": %v }]", scale)))
		Expect(err).ToNot(HaveOccurred())

		By("Checking the number of replicas")
		Eventually(func() int32 {
			rs, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Get(name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(scale)))

		vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tests.NotDeleted(vmis)).To(HaveLen(int(scale)))
	}

	doScaleWithHPA := func(name string, min int32, max int32, expected int32) {

		// Status updates can conflict with our desire to change the spec
		By(fmt.Sprintf("Scaling to %d", min))
		hpa := &autov1.HorizontalPodAutoscaler{
			ObjectMeta: v12.ObjectMeta{
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
		_, err := virtClient.AutoscalingV1().HorizontalPodAutoscalers(util.NamespaceTestDefault).Create(context.Background(), hpa, v12.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		var s *autov1.Scale
		By("Checking the number of replicas")
		EventuallyWithOffset(1, func() int32 {
			s, err = virtClient.ReplicaSet(util.NamespaceTestDefault).GetScale(name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return s.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(expected)))

		vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, tests.NotDeleted(vmis)).To(HaveLen(int(min)))
		err = virtClient.AutoscalingV1().HorizontalPodAutoscalers(util.NamespaceTestDefault).Delete(context.Background(), name, v12.DeleteOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	newReplicaSetWithTemplate := func(template *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceReplicaSet {
		newRS := tests.NewRandomReplicaSetFromVMI(template, int32(0))
		newRS, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Create(newRS)
		Expect(err).ToNot(HaveOccurred())
		return newRS
	}

	newReplicaSet := func() *v1.VirtualMachineInstanceReplicaSet {
		By("Create a new VirtualMachineInstance replica set")
		template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		return newReplicaSetWithTemplate(template)
	}

	table.DescribeTable("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component]should scale", func(startScale int, stopScale int) {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, int32(startScale))
		doScale(newRS.ObjectMeta.Name, int32(stopScale))
		doScale(newRS.ObjectMeta.Name, int32(0))

	},
		table.Entry("[test_id:1405]to three, to two and then to zero replicas", 3, 2),
		table.Entry("[test_id:1406]to five, to six and then to zero replicas", 5, 6),
	)

	table.DescribeTable("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component]should scale with scale subresource", func(startScale int, stopScale int) {
		newRS := newReplicaSet()
		libreplicaset.DoScaleWithScaleSubresource(virtClient, newRS.ObjectMeta.Name, int32(startScale))
		libreplicaset.DoScaleWithScaleSubresource(virtClient, newRS.ObjectMeta.Name, int32(stopScale))
		libreplicaset.DoScaleWithScaleSubresource(virtClient, newRS.ObjectMeta.Name, int32(0))
	},
		table.Entry("[test_id:1407]to three, to two and then to zero replicas", 3, 2),
		table.Entry("[test_id:1408]to five, to six and then to zero replicas", 5, 6),
	)

	table.DescribeTable("[rfe_id:588][crit:medium][vendor:cnv-qe@redhat.com][level:component]should scale with the horizontal pod autoscaler", func(startScale int, stopScale int) {
		tests.SkipIfVersionBelow("HPA only works with CRs with multiple versions starting from 1.13", "1.13")
		template := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
		newRS := tests.NewRandomReplicaSetFromVMI(template, int32(1))
		newRS, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Create(newRS)
		Expect(err).ToNot(HaveOccurred())
		doScaleWithHPA(newRS.ObjectMeta.Name, int32(startScale), int32(startScale), int32(startScale))
		doScaleWithHPA(newRS.ObjectMeta.Name, int32(stopScale), int32(stopScale), int32(stopScale))
		doScaleWithHPA(newRS.ObjectMeta.Name, int32(1), int32(1), int32(1))

	},
		table.Entry("[test_id:1409]to three, to two and then to one replicas", 3, 2),
		table.Entry("[test_id:1410]to five, to six and then to one replicas", 5, 6),
	)

	It("[test_id:1411]should be rejected on POST if spec is invalid", func() {
		newRS := newReplicaSet()
		newRS.TypeMeta = v12.TypeMeta{
			APIVersion: v1.StorageGroupVersion.String(),
			Kind:       "VirtualMachineInstanceReplicaSet",
		}

		jsonBytes, err := json.Marshal(newRS)
		Expect(err).To(BeNil())

		// change the name of a required field (like domain) so validation will fail
		jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

		result := virtClient.RestClient().Post().Resource("virtualmachineinstancereplicasets").Namespace(util.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do(context.Background())

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

	})
	It("[test_id:1412]should reject POST if validation webhoook deems the spec is invalid", func() {
		newRS := newReplicaSet()
		newRS.TypeMeta = v12.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceReplicaSet",
		}

		// Add a disk that doesn't map to a volume.
		// This should get rejected which tells us the webhook validator is working.
		newRS.Spec.Template.Spec.Domain.Devices.Disks = append(newRS.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})

		result := virtClient.RestClient().Post().Resource("virtualmachineinstancereplicasets").Namespace(util.NamespaceTestDefault).Body(newRS).Do(context.Background())

		// Verify validation failed.
		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		reviewResponse := &v12.Status{}
		body, _ := result.Raw()
		err = json.Unmarshal(body, reviewResponse)
		Expect(err).To(BeNil())

		Expect(len(reviewResponse.Details.Causes)).To(Equal(1))
		Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[2].name"))
	})
	It("[test_id:1413]should update readyReplicas once VMIs are up", func() {
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		By("checking the number of ready replicas in the returned yaml")
		Eventually(func() int {
			rs, err := virtClient.ReplicaSet(util.NamespaceTestDefault).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return int(rs.Status.ReadyReplicas)
		}, 120*time.Second, 1*time.Second).Should(Equal(2))
	})

	It("[test_id:1414]should return the correct data when using server-side printing", func() {
		tests.SkipIfVersionBelow("server-side printing is only enabled by default from 1.11 on", "1.11")
		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		By("waiting until all VMIs are ready")
		Eventually(func() int {
			rs, err := virtClient.ReplicaSet(util.NamespaceTestDefault).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return int(rs.Status.ReadyReplicas)
		}, 120*time.Second, 1*time.Second).Should(Equal(2))

		By("checking the output of server-side table printing")
		rawTable, err := virtClient.RestClient().Get().
			RequestURI(fmt.Sprintf("/apis/kubevirt.io/%s/namespaces/%s/virtualmachineinstancereplicasets/%s", v1.ApiLatestVersion, util.NamespaceTestDefault, newRS.ObjectMeta.Name)).
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
		Expect(int(table.Rows[0].Cells[1].(float64))).To(Equal(2))
		Expect(int(table.Rows[0].Cells[2].(float64))).To(Equal(2))
		Expect(int(table.Rows[0].Cells[3].(float64))).To(Equal(2))
	})

	It("[test_id:1415]should remove VMIs once they are marked for deletion", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)
		// Delete it
		By("Deleting the VirtualMachineInstance replica set")
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Delete(newRS.ObjectMeta.Name, &v12.DeleteOptions{})).To(Succeed())
		// Wait until VMIs are gone
		By("Waiting until all VMIs are gone")
		Eventually(func() int {
			vmis, err := virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vmis.Items)
		}, 120*time.Second, 1*time.Second).Should(BeZero())
	})

	It("[test_id:1416]should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {
		newRS := newReplicaSet()
		// Create a replicaset with two replicas
		doScale(newRS.ObjectMeta.Name, 2)

		// Check for owner reference
		vmis, err := virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(&v12.ListOptions{})
		Expect(vmis.Items).To(HaveLen(2))
		Expect(err).ToNot(HaveOccurred())
		for _, vmi := range vmis.Items {
			Expect(vmi.OwnerReferences).ToNot(BeEmpty())
		}

		// Delete it
		By("Deleting the VirtualMachineInstance replica set with the 'orphan' deletion strategy")
		orphanPolicy := v12.DeletePropagationOrphan
		Expect(virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).
			Delete(newRS.ObjectMeta.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
		// Wait until the replica set is deleted
		By("Waiting until the replica set got deleted")
		Eventually(func() bool {
			_, err := virtClient.ReplicaSet(newRS.ObjectMeta.Namespace).Get(newRS.ObjectMeta.Name, v12.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			return false
		}, 60*time.Second, 1*time.Second).Should(BeTrue())

		By("Checking if two VMIs are orphaned and still exist")
		vmis, err = virtClient.VirtualMachineInstance(newRS.ObjectMeta.Namespace).List(&v12.ListOptions{})
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
		_, err := virtClient.ReplicaSet(rs.Namespace).Patch(rs.Name, types.JSONPatchType, []byte("[{ \"op\": \"add\", \"path\": \"/spec/paused\", \"value\": true }]"))
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() v1.VirtualMachineInstanceReplicaSetConditionType {
			rs, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(rs.Status.Conditions) > 0 {
				return rs.Status.Conditions[0].Type
			}
			return ""
		}, 10*time.Second, 1*time.Second).Should(Equal(v1.VirtualMachineInstanceReplicaSetReplicaPaused))

		// set new replica count while still being paused
		By("Updating the number of replicas")
		rs.Spec.Replicas = tests.NewInt32(2)
		_, err = virtClient.ReplicaSet(rs.ObjectMeta.Namespace).Update(rs)
		Expect(err).ToNot(HaveOccurred())

		// make sure that we don't scale up
		By("Checking that the replicaset do not scale while it is paused")
		Consistently(func() int32 {
			rs, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Make sure that no failure happened, so that ensure that we don't scale because we are paused
			Expect(rs.Status.Conditions).To(HaveLen(1))
			return rs.Status.Replicas
		}, 3*time.Second, 1*time.Second).Should(Equal(int32(0)))

		// resume controller
		By("Resuming the replicaset")
		_, err = virtClient.ReplicaSet(rs.Namespace).Patch(rs.Name, types.JSONPatchType, []byte("[{ \"op\": \"replace\", \"path\": \"/spec/paused\", \"value\": false }]"))
		Expect(err).ToNot(HaveOccurred())

		// Paused condition should disappear
		By("Checking that the pause condition disappeared from the replicaset")
		Eventually(func() int {
			rs, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(rs.Status.Conditions)
		}, 10*time.Second, 1*time.Second).Should(Equal(0))

		// Replicas should be created
		By("Checking that the missing replicas are now created")
		Eventually(func() int32 {
			rs, err = virtClient.ReplicaSet(util.NamespaceTestDefault).Get(rs.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return rs.Status.Replicas
		}, 10*time.Second, 1*time.Second).Should(Equal(int32(2)))
	})

	It("[test_id:1418]should replace finished VMIs", func() {
		By("Creating new replica set")
		rs := newReplicaSet()
		doScale(rs.ObjectMeta.Name, int32(2))

		vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis.Items).ToNot(BeEmpty())

		vmi := vmis.Items[0]
		pods, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).List(context.Background(), tests.UnfinishedVMIPodSelector(&vmi))
		Expect(err).ToNot(HaveOccurred())
		Expect(len(pods.Items)).To(Equal(1))
		pod := pods.Items[0]

		By("Deleting one of the RS VMS pods")
		err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), pod.Name, v12.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking that the VM disappeared")
		Eventually(func() bool {
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &v12.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			return false
		}, 120*time.Second, time.Second).Should(BeTrue())

		By("Checking number of RS VM's to see that we got a replacement")
		vmis, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vmis.Items)).Should(Equal(2))
	})

	It("should replace a VMI immediately when a virt-launcher pod gets deleted", func() {
		By("Creating new replica set")
		template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		var gracePeriod int64 = 200
		template.Spec.TerminationGracePeriodSeconds = &gracePeriod
		rs := newReplicaSetWithTemplate(template)

		// ensure that the shutdown will take as long as possible

		doScale(rs.ObjectMeta.Name, int32(2))

		vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmis.Items).ToNot(BeEmpty())

		By("Waiting until the VMIs are running")
		Eventually(func() int {
			vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(tests.Running(vmis))
		}, 40*time.Second, time.Second).Should(Equal(2))

		vmi := &vmis.Items[0]
		pods, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).List(context.Background(), tests.UnfinishedVMIPodSelector(vmi))
		Expect(err).ToNot(HaveOccurred())
		Expect(len(pods.Items)).To(Equal(1))
		pod := pods.Items[0]

		By("Deleting one of the RS VMS pods, which will take some time to really stop the VMI")
		err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), pod.Name, v12.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking that then number of VMIs increases to three")
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() int {
			vmis, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vmis.Items)
		}, 20*time.Second, time.Second).Should(Equal(3))

		By("Checking that the shutting donw VMI is still running, reporting the pod deletion and being marked for deletion")
		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		Expect(controller.NewVirtualMachineInstanceConditionManager().
			HasConditionWithStatusAndReason(
				vmi,
				v1.VirtualMachineInstanceConditionType(v13.PodReady),
				v13.ConditionFalse,
				v1.PodTerminatingReason,
			),
		).To(BeTrue())
		Expect(vmi.DeletionTimestamp).ToNot(BeNil())
	})

	It("[test_id:4121]should create and verify kubectl/oc output for vm replicaset", func() {
		k8sClient := tests.GetK8sCmdClient()
		tests.SkipIfNoCmd(k8sClient)

		newRS := newReplicaSet()
		doScale(newRS.ObjectMeta.Name, 2)

		result, _, _ := tests.RunCommand(k8sClient, "get", "virtualmachineinstancereplicaset")
		Expect(result).ToNot(BeNil())
		resultFields := strings.Fields(result)
		expectedHeader := []string{"NAME", "DESIRED", "CURRENT", "READY", "AGE"}
		columnHeaders := resultFields[:len(expectedHeader)]
		// Verify the generated header is same as expected
		Expect(columnHeaders).To(Equal(expectedHeader))
		// Name will be there in all the cases, so verify name
		Expect(resultFields[len(expectedHeader)]).To(Equal(newRS.Name))
	})
})
