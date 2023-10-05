package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/testsuite"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	grepCmd                  = "%s | grep \"%s\"\n"
	grepCmdWithCount         = "%s | grep \"%s\"| wc -l\n"
	qemuGa                   = ".*qemu-ga.*%s.*"
	vmSnapshotContent        = "vmsnapshot-content"
	snapshotDeadlineExceeded = "snapshot deadline exceeded"
	notReady                 = "Not ready"
	operationComplete        = "Operation complete"
)

var _ = SIGDescribe("VirtualMachineSnapshot Tests", func() {

	var (
		err        error
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
		snapshot   *snapshotv1.VirtualMachineSnapshot
		webhook    *admissionregistrationv1.ValidatingWebhookConfiguration
	)

	groupName := "kubevirt.io"

	newSnapshot := func() *snapshotv1.VirtualMachineSnapshot {
		return &snapshotv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snapshot-" + vm.Name,
				Namespace: vm.Namespace,
			},
			Spec: snapshotv1.VirtualMachineSnapshotSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: &groupName,
					Kind:     "VirtualMachine",
					Name:     vm.Name,
				},
			},
		}
	}

	waitSnapshotReady := func() {
		Eventually(func() *snapshotv1.VirtualMachineSnapshotStatus {
			snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return snapshot.Status
		}, 180*time.Second, time.Second).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"ReadyToUse": gstruct.PointTo(BeTrue()),
		})))
	}

	waitSnapshotSucceeded := func(snapshotName string) *snapshotv1.VirtualMachineSnapshot {
		var snapshot *snapshotv1.VirtualMachineSnapshot
		Eventually(func() *snapshotv1.VirtualMachineSnapshotStatus {
			snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshotName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return snapshot.Status
		}, 180*time.Second, 2*time.Second).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Conditions": ContainElements(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(snapshotv1.ConditionReady),
					"Status": Equal(corev1.ConditionTrue),
					"Reason": Equal(operationComplete)}),
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(snapshotv1.ConditionProgressing),
					"Status": Equal(corev1.ConditionFalse),
					"Reason": Equal(operationComplete)}),
			),
		})))

		return snapshot
	}

	deleteSnapshot := func() {
		err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Delete(context.Background(), snapshot.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
		snapshot = nil
	}

	deleteWebhook := func() {
		err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), webhook.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
		webhook = nil
	}

	deletePVC := func(pvc *corev1.PersistentVolumeClaim) {
		err := virtClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			err = nil
		}
		Expect(err).ToNot(HaveOccurred())
		pvc = nil
	}

	waitDataVolumePopulated := func(namespace, name string) {
		libstorage.EventuallyDVWith(namespace, name, 180, matcher.HaveSucceeded())
		// THIS SHOULD NOT BE NECESSARY - but in DV/Populator integration
		Eventually(func() string {
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pvc.Spec.VolumeName
		}, 180*time.Second, time.Second).ShouldNot(BeEmpty())
	}

	createDenyVolumeSnapshotCreateWebhook := func() {
		fp := admissionregistrationv1.Fail
		sideEffectNone := admissionregistrationv1.SideEffectClassNone
		whPath := "/foobar"
		whName := "dummy-webhook-deny-volume-snapshot-create.kubevirt.io"
		wh := &admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "temp-webhook-deny-volume-snapshot-create-" + rand.String(5),
			},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{
				{
					Name:                    whName,
					AdmissionReviewVersions: []string{"v1", "v1beta1"},
					FailurePolicy:           &fp,
					SideEffects:             &sideEffectNone,
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{vsv1.GroupName},
							APIVersions: []string{vsv1.SchemeGroupVersion.Version},
							Resources:   []string{"volumesnapshots"},
						},
					}},
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: testsuite.GetTestNamespace(nil),
							Name:      "nonexistant",
							Path:      &whPath,
						},
					},
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"snapshot.kubevirt.io/source-vm-name": vm.Name,
						},
					},
				},
			},
		}
		wh, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), wh, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		webhook = wh
	}

	checkOnlineSnapshotExpectedContentSource := func(vm *v1.VirtualMachine, contentName string, expectVolumeBackups bool) {
		content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Template.Spec.Volumes = content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes
		vm.Spec.Template.Spec.Domain.Devices.Disks = content.Spec.Source.VirtualMachine.Spec.Template.Spec.Domain.Devices.Disks

		Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
		Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
		Expect(content.Spec.Source.VirtualMachine.UID).ToNot(BeEmpty())
		if expectVolumeBackups {
			Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))
		} else {
			Expect(content.Spec.VolumeBackups).Should(BeEmpty())
		}
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	AfterEach(func() {
		if snapshot != nil {
			deleteSnapshot()
		}
		if webhook != nil {
			deleteWebhook()
		}
	})

	Context("With simple VM", func() {
		BeforeEach(func() {
			var err error
			vm = tests.NewRandomVirtualMachine(libvmi.NewCirros(), false)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
		})

		createAndVerifyVMSnapshot := func(vm *v1.VirtualMachine) {
			snapshot = newSnapshot()

			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitSnapshotReady()

			Expect(snapshot.Status.SourceUID).ToNot(BeNil())
			Expect(*snapshot.Status.SourceUID).To(Equal(vm.UID))

			contentName := *snapshot.Status.VirtualMachineSnapshotContentName
			if vm.Spec.Running != nil && *vm.Spec.Running {
				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))
				checkOnlineSnapshotExpectedContentSource(vm, contentName, false)
			} else {
				Expect(snapshot.Status.Indications).To(BeEmpty())
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.Source.VirtualMachine.UID).ToNot(BeEmpty())
				Expect(content.Spec.VolumeBackups).To(BeEmpty())
			}
		}

		It("[test_id:4609]should successfully create a snapshot", func() {
			createAndVerifyVMSnapshot(vm)
		})

		It("[test_id:4610]create a snapshot when VM is running should succeed", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(*vm.Spec.Running).Should(BeTrue())

			createAndVerifyVMSnapshot(vm)
		})

		It("should create a snapshot when VM runStrategy is Manual", func() {
			patch := []byte("[{ \"op\": \"remove\", \"path\": \"/spec/running\"}, { \"op\": \"add\", \"path\": \"/spec/runStrategy\", \"value\": \"Manual\"}]")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.RunStrategy).ToNot(BeNil())
			Expect(*vm.Spec.RunStrategy).Should(Equal(v1.RunStrategyManual))

			createAndVerifyVMSnapshot(vm)
		})

		It("VM should contain snapshot status for all volumes", func() {
			patch := []byte("[{ \"op\": \"replace\", \"path\": \"/spec/running\", \"value\": true }]")
			vm, err := virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				vm2, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("VM Statuses: %+v", vm2.Status))
				return len(vm2.Status.VolumeSnapshotStatuses) == 2 &&
					vm2.Status.VolumeSnapshotStatuses[0].Name == "disk0" &&
					vm2.Status.VolumeSnapshotStatuses[1].Name == "disk1"
			}, 180*time.Second, time.Second).Should(BeTrue())
		})
	})

	Context("[storage-req]", decorators.StorageReq, func() {
		var (
			snapshotStorageClass string
		)

		BeforeEach(func() {
			sc, err := libstorage.GetSnapshotStorageClass(virtClient)
			Expect(err).ToNot(HaveOccurred())

			if sc == "" {
				Skip("Skiping test, no VolumeSnapshot support")
			}

			snapshotStorageClass = sc
		})

		Context("With online vm snapshot", func() {
			const VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION = "pre.hook.backup.velero.io/container"
			const VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION = "pre.hook.backup.velero.io/command"
			const VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION = "post.hook.backup.velero.io/container"
			const VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION = "post.hook.backup.velero.io/command"

			createAndStartVM := func(vm *v1.VirtualMachine) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
				var vmi *v1.VirtualMachineInstance
				t := true
				vm.Spec.Running = &t
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 360*time.Second, time.Second).Should(BeTrue())

				return vm, vmi
			}

			checkVMFreeze := func(snapshot *snapshotv1.VirtualMachineSnapshot, vmi *v1.VirtualMachineInstance, hasGuestAgent, shouldFreeze bool) {
				var expectedIndications []snapshotv1.Indication
				if hasGuestAgent {
					expectedIndications = []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotGuestAgentIndication}
				} else {
					expectedIndications = []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotNoGuestAgentIndication}
				}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				conditionsLength := 2
				Expect(snapshot.Status.Conditions).To(HaveLen(conditionsLength))
				Expect(snapshot).To(matcher.HaveConditionMissingOrFalse(snapshotv1.ConditionProgressing))
				Expect(snapshot).To(matcher.HaveConditionTrue(snapshotv1.ConditionReady))

				Expect(console.LoginToFedora(vmi)).To(Succeed())
				journalctlCheck := "journalctl --file /var/log/journal/*/system.journal"
				expectedFreezeOutput := "executing fsfreeze hook with arg 'freeze'"
				expectedThawOutput := "executing fsfreeze hook with arg 'thaw'"
				if hasGuestAgent {
					if shouldFreeze {
						Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf(grepCmd, journalctlCheck, expectedFreezeOutput)},
							&expect.BExp{R: fmt.Sprintf(qemuGa, expectedFreezeOutput)},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: fmt.Sprintf(grepCmd, journalctlCheck, expectedThawOutput)},
							&expect.BExp{R: fmt.Sprintf(qemuGa, expectedThawOutput)},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
							&expect.BSnd{S: fmt.Sprintf(grepCmdWithCount, journalctlCheck, expectedThawOutput)},
							&expect.BExp{R: console.RetValue("1")},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("0")},
						}, 30)).To(Succeed())
					} else {
						Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
							&expect.BSnd{S: fmt.Sprintf(grepCmd, journalctlCheck, expectedFreezeOutput)},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: tests.EchoLastReturnValue},
							&expect.BExp{R: console.RetValue("1")},
						}, 30)).To(Succeed())
					}
				}
			}

			checkContentSourceAndMemory := func(vm *v1.VirtualMachine, contentName string, expectedMemory resource.Quantity) {
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				contentSourceSpec := content.Spec.Source.VirtualMachine.Spec
				snapshotSourceMemory := contentSourceSpec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				Expect(snapshotSourceMemory).To(Equal(expectedMemory))
				checkOnlineSnapshotExpectedContentSource(vm, contentName, true)
			}

			callVeleroHook := func(vmi *v1.VirtualMachineInstance, annoContainer, annoCommand string) (string, string, error) {
				pod := tests.GetPodByVirtualMachineInstance(vmi)

				command := pod.Annotations[annoCommand]
				command = strings.Trim(command, "[]")
				commandSlice := []string{}
				for _, c := range strings.Split(command, ",") {
					commandSlice = append(commandSlice, strings.Trim(c, "\" "))
				}
				virtClient := kubevirt.Client()
				return exec.ExecuteCommandOnPodWithResults(virtClient, pod, pod.Annotations[annoContainer], commandSlice)
			}

			It("[test_id:6767]with volumes and guest agent available", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(nil)
				vm = tests.NewRandomVirtualMachine(vmi, false)
				dvName := "dv-" + vm.Name
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: dvName,
						},
						Spec: cdiv1.DataVolumeSpec{
							Source: &cdiv1.DataVolumeSource{
								Blank: &cdiv1.DataVolumeBlankImage{},
							},
							PVC: &corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": quantity,
									},
								},
								StorageClassName: &snapshotStorageClass,
							},
						},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blank",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				})
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "blank",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv-" + vm.Name,
						},
					},
				})

				vm, vmi = createAndStartVM(vm)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				initialMemory := vmi.Spec.Domain.Resources.Requests[corev1.ResourceMemory]
				newMemory := resource.MustParse("1Gi")
				Expect(newMemory).ToNot(Equal(initialMemory))

				//update vm to make sure vm revision is saved in the snapshot
				By("Updating the VM template spec")
				patchData, err := patch.GenerateTestReplacePatch(
					"/spec/template/spec/domain/resources/requests/"+string(corev1.ResourceMemory),
					initialMemory,
					newMemory,
				)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchData, &metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
				shouldFreeze := true
				checkVMFreeze(snapshot, vmi, true, shouldFreeze)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				checkContentSourceAndMemory(vm.DeepCopy(), contentName, initialMemory)
			})

			It("[test_id:6768]with volumes and no guest agent available", func() {
				quantity, err := resource.ParseQuantity("1Gi")
				Expect(err).ToNot(HaveOccurred())
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(nil)
				vm = tests.NewRandomVirtualMachine(vmi, false)
				dvName := "dv-" + vm.Name
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: dvName,
						},
						Spec: cdiv1.DataVolumeSpec{
							Source: &cdiv1.DataVolumeSource{
								Blank: &cdiv1.DataVolumeBlankImage{},
							},
							PVC: &corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": quantity,
									},
								},
								StorageClassName: &snapshotStorageClass,
							},
						},
					},
				}
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blank",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				})
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "blank",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv-" + vm.Name,
						},
					},
				})

				vm, vmi = createAndStartVM(vm)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
				shouldFreeze := false
				checkVMFreeze(snapshot, vmi, false, shouldFreeze)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				checkOnlineSnapshotExpectedContentSource(vm.DeepCopy(), contentName, true)
			})

			It("[test_id:6769]without volumes with guest agent available", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(nil)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
				shouldFreeze := false
				checkVMFreeze(snapshot, vmi, true, shouldFreeze)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				checkOnlineSnapshotExpectedContentSource(vm.DeepCopy(), contentName, false)
			})

			It("[test_id:6837]delete snapshot after freeze, expect vm unfreeze", func() {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVMI.Status.FSFreezeStatus == "frozen"
				}, time.Minute, 2*time.Second).Should(BeTrue())

				deleteSnapshot()
				Eventually(func() bool {
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVMI.Status.FSFreezeStatus == ""
				}, time.Minute, 2*time.Second).Should(BeTrue())
			})

			It("[test_id:6949]should unfreeze vm if snapshot fails when deadline exceeded", func() {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()
				snapshot.Spec.FailureDeadline = &metav1.Duration{Duration: 40 * time.Second}

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return snapshot.Status != nil &&
						snapshot.Status.Phase == snapshotv1.InProgress &&
						updatedVMI.Status.FSFreezeStatus == "frozen"
				}, 30*time.Second, 2*time.Second).Should(BeTrue())

				contentName := fmt.Sprintf("%s-%s", vmSnapshotContent, snapshot.UID)
				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					updatedVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					_, contentErr := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
					return snapshot.Status != nil &&
						len(snapshot.Status.Conditions) == 3 &&
						snapshot.Status.Conditions[0].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[0].Reason, snapshotDeadlineExceeded) &&
						snapshot.Status.Conditions[1].Status == corev1.ConditionFalse &&
						strings.Contains(snapshot.Status.Conditions[1].Reason, notReady) &&
						snapshot.Status.Conditions[2].Status == corev1.ConditionTrue &&
						snapshot.Status.Conditions[2].Type == snapshotv1.ConditionFailure &&
						strings.Contains(snapshot.Status.Conditions[2].Reason, snapshotDeadlineExceeded) &&
						snapshot.Status.Phase == snapshotv1.Failed &&
						updatedVMI.Status.FSFreezeStatus == "" &&
						errors.IsNotFound(contentErr)
				}, time.Minute, 2*time.Second).Should(BeTrue())
			})

			It("[test_id:7472]should succeed online snapshot with hot plug disk", func() {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce))
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Add persistent hotplug disk")
				persistVolName := AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, false)
				By("Add temporary hotplug disk")
				tempVolName := AddVolumeAndVerify(virtClient, snapshotStorageClass, vm, true)
				By("Create Snapshot")
				snapshot = newSnapshot()
				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				snapshot = waitSnapshotSucceeded(snapshot.Name)
				expectedIndications := []snapshotv1.Indication{snapshotv1.VMSnapshotOnlineSnapshotIndication, snapshotv1.VMSnapshotGuestAgentIndication}
				Expect(snapshot.Status.Indications).To(Equal(expectedIndications))

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				contentVMTemplate := content.Spec.Source.VirtualMachine.Spec.Template
				Expect(contentVMTemplate.Spec.Volumes).Should(HaveLen(len(updatedVM.Spec.Template.Spec.Volumes)))
				foundHotPlug := false
				foundTempHotPlug := false
				for _, volume := range contentVMTemplate.Spec.Volumes {
					if volume.Name == persistVolName {
						foundHotPlug = true
					} else if volume.Name == tempVolName {
						foundTempHotPlug = true
					}
				}
				Expect(foundHotPlug).To(BeTrue())
				Expect(foundTempHotPlug).To(BeFalse())

				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(updatedVM.Spec.Template.Spec.Volumes)))
				Expect(snapshot.Status.SnapshotVolumes.IncludedVolumes).Should(HaveLen(len(content.Spec.VolumeBackups)))
				Expect(snapshot.Status.SnapshotVolumes.ExcludedVolumes).Should(BeEmpty())
				for _, vol := range updatedVM.Spec.Template.Spec.Volumes {
					if vol.DataVolume == nil {
						continue
					}
					found := false
					for _, vb := range content.Spec.VolumeBackups {
						if vol.DataVolume.Name == vb.PersistentVolumeClaim.Name {
							found = true
							Expect(vol.Name).To(Equal(vb.VolumeName))

							pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), vol.DataVolume.Name, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(pvc.Spec).To(Equal(vb.PersistentVolumeClaim.Spec))

							Expect(vb.VolumeSnapshotName).ToNot(BeNil())
							vs, err := virtClient.
								KubernetesSnapshotClient().
								SnapshotV1().
								VolumeSnapshots(vm.Namespace).
								Get(context.Background(), *vb.VolumeSnapshotName, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(*vs.Spec.Source.PersistentVolumeClaimName).Should(Equal(vol.DataVolume.Name))
							Expect(vs.Status.Error).To(BeNil())
							Expect(*vs.Status.ReadyToUse).To(BeTrue())
						}
					}
					Expect(found).To(BeTrue())
				}
			})

			It("Calling Velero hooks should freeze/unfreeze VM", func() {
				By("Creating VM")
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(nil)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Logging into Fedora")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Calling Velero pre-backup hook")
				_, _, err := callVeleroHook(vmi, VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).ToNot(HaveOccurred())

				By("Veryfing the VM was frozen")
				journalctlCheck := "journalctl --file /var/log/journal/*/system.journal"
				expectedFreezeOutput := "executing fsfreeze hook with arg 'freeze'"
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf(grepCmd, journalctlCheck, expectedFreezeOutput)},
					&expect.BExp{R: fmt.Sprintf(qemuGa, expectedFreezeOutput)},
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 30)).To(Succeed())
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.FSFreezeStatus == "frozen"
				}, 180*time.Second, time.Second).Should(BeTrue())

				By("Calling Velero post-backup hook")
				_, _, err = callVeleroHook(vmi, VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).ToNot(HaveOccurred())

				By("Veryfing the VM was thawed")
				expectedThawOutput := "executing fsfreeze hook with arg 'thaw'"
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf(grepCmd, journalctlCheck, expectedThawOutput)},
					&expect.BExp{R: fmt.Sprintf(qemuGa, expectedThawOutput)},
					&expect.BSnd{S: tests.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 30)).To(Succeed())
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.FSFreezeStatus == ""
				}, 180*time.Second, time.Second).Should(BeTrue())
			})

			It("[test_id:9647]Calling Velero hooks should not error if no guest agent", func() {
				const noGuestAgentString = "No guest agent, exiting"
				By("Creating VM")
				var vmi *v1.VirtualMachineInstance
				running := false
				vm = tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce,
				)
				vm.Spec.Running = &running

				vm, vmi = createAndStartVM(vm)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)

				By("Calling Velero pre-backup hook")
				_, stderr, err := callVeleroHook(vmi, VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).ToNot(HaveOccurred())
				Expect(stderr).Should(ContainSubstring(noGuestAgentString))

				By("Calling Velero post-backup hook")
				_, stderr, err = callVeleroHook(vmi, VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).ToNot(HaveOccurred())
				Expect(stderr).Should(ContainSubstring(noGuestAgentString))
			})

			It("Calling Velero hooks should error if VM is Paused", func() {
				By("Creating VM")
				vmi := tests.NewRandomFedoraVMI()
				vmi.Namespace = testsuite.GetTestNamespace(nil)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				vm, vmi = createAndStartVM(vm)
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(300),
				)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Logging into Fedora")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Pausing the VirtualMachineInstance")
				err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

				By("Calling Velero pre-backup hook")
				_, stderr, err := callVeleroHook(vmi, VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION, VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION)
				Expect(err).To(HaveOccurred())
				Expect(stderr).Should(ContainSubstring("Paused VM"))
			})

			Context("with memory dump", func() {
				var memoryDumpPVC *corev1.PersistentVolumeClaim
				const memoryDumpPVCName = "fs-pvc"

				BeforeEach(func() {
					memoryDumpPVC = libstorage.NewPVC(memoryDumpPVCName, "1.5Gi", snapshotStorageClass)
					volumeMode := corev1.PersistentVolumeFilesystem
					memoryDumpPVC.Spec.VolumeMode = &volumeMode
					var err error
					memoryDumpPVC, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Create(context.Background(), memoryDumpPVC, metav1.CreateOptions{})
					if err != nil {
						Skip(fmt.Sprintf("Skiping test, no filesystem pvc available, err: %s", err))
					}
				})

				AfterEach(func() {
					if memoryDumpPVC != nil {
						deletePVC(memoryDumpPVC)
					}
				})

				getMemoryDump := func(vmName, namespace, claimName string) {
					Eventually(func() error {
						memoryDumpRequest := &v1.VirtualMachineMemoryDumpRequest{
							ClaimName: claimName,
						}

						return virtClient.VirtualMachine(namespace).MemoryDump(context.Background(), vmName, memoryDumpRequest)
					}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				}

				waitMemoryDumpCompletion := func(vm *v1.VirtualMachine) {
					Eventually(func() bool {
						updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if updatedVM.Status.MemoryDumpRequest == nil ||
							updatedVM.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted {
							return false
						}

						return true
					}, 60*time.Second, time.Second).Should(BeTrue())
				}

				It("[test_id:8922]should include memory dump in vm snapshot", func() {
					var vmi *v1.VirtualMachineInstance
					vm, vmi = createAndStartVM(tests.NewRandomVMWithDataVolumeWithRegistryImport(
						cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling),
						testsuite.GetTestNamespace(nil),
						snapshotStorageClass,
						corev1.ReadWriteOnce))
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					By("Get VM memory dump")
					getMemoryDump(vm.Name, vm.Namespace, memoryDumpPVCName)
					waitMemoryDumpCompletion(vm)

					By("Create Snapshot")
					snapshot = newSnapshot()
					_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					snapshot = waitSnapshotSucceeded(snapshot.Name)

					updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(updatedVM.Status.MemoryDumpRequest).ToNot(BeNil())
					contentName := *snapshot.Status.VirtualMachineSnapshotContentName
					content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					contentVMTemplate := content.Spec.Source.VirtualMachine.Spec.Template
					Expect(contentVMTemplate.Spec.Volumes).Should(HaveLen(len(updatedVM.Spec.Template.Spec.Volumes)))
					foundMemoryDump := false
					for _, volume := range contentVMTemplate.Spec.Volumes {
						if volume.Name == memoryDumpPVCName {
							foundMemoryDump = true
						}
					}
					Expect(foundMemoryDump).To(BeTrue())

					Expect(content.Spec.VolumeBackups).Should(HaveLen(len(updatedVM.Spec.Template.Spec.Volumes)))
					for _, vol := range updatedVM.Spec.Template.Spec.Volumes {
						if vol.MemoryDump == nil {
							continue
						}
						found := false
						for _, vb := range content.Spec.VolumeBackups {
							if vol.MemoryDump.ClaimName == vb.PersistentVolumeClaim.Name {
								found = true
								Expect(vol.Name).To(Equal(vb.VolumeName))

								pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), vol.MemoryDump.ClaimName, metav1.GetOptions{})
								Expect(err).ToNot(HaveOccurred())
								Expect(pvc.Spec).To(Equal(vb.PersistentVolumeClaim.Spec))

								Expect(vb.VolumeSnapshotName).ToNot(BeNil())
								vs, err := virtClient.
									KubernetesSnapshotClient().
									SnapshotV1().
									VolumeSnapshots(vm.Namespace).
									Get(context.Background(), *vb.VolumeSnapshotName, metav1.GetOptions{})
								Expect(err).ToNot(HaveOccurred())
								Expect(*vs.Spec.Source.PersistentVolumeClaimName).Should(Equal(vol.MemoryDump.ClaimName))
								Expect(vs.Status.Error).To(BeNil())
								Expect(*vs.Status.ReadyToUse).To(BeTrue())
							}
						}
						Expect(found).To(BeTrue())
					}
				})
			})
		})

		Context("With more complicated VM", func() {
			BeforeEach(func() {
				running := false
				vm = tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce,
				)
				vm.Spec.Running = &running

				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					waitDataVolumePopulated(vm.Namespace, dvt.Name)
				}
			})

			It("should successfully recreate status", func() {
				snapshot = newSnapshot()

				ss, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				ss, err = virtClient.VirtualMachineSnapshot(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				origStatus := ss.Status
				ss.Status = nil
				ss, err = virtClient.VirtualMachineSnapshot(ss.Namespace).Update(context.Background(), ss, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(ss.Status).To(BeNil())

				Eventually(func() bool {
					ss, err = virtClient.VirtualMachineSnapshot(ss.Namespace).Get(context.Background(), ss.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if ss.Status == nil {
						return false
					}
					if *ss.Status.SourceUID != *origStatus.SourceUID ||
						*ss.Status.VirtualMachineSnapshotContentName != *origStatus.VirtualMachineSnapshotContentName ||
						*ss.Status.CreationTime != *origStatus.CreationTime ||
						ss.Status.Phase != origStatus.Phase ||
						*ss.Status.ReadyToUse != *origStatus.ReadyToUse {
						return false
					}
					if len(ss.Status.Conditions) != len(origStatus.Conditions) {
						return false
					}
					for i, c := range ss.Status.Conditions {
						oc := origStatus.Conditions[i]
						if c.Type != oc.Type ||
							c.Status != oc.Status {
							return false
						}
					}
					if len(ss.Status.Indications) != len(origStatus.Indications) {
						return false
					}
					for i := range ss.Status.Indications {
						if ss.Status.Indications[i] != origStatus.Indications[i] {
							return false
						}
					}
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())
			})

			It("VM should contain snapshot status for all volumes", func() {
				volumes := len(vm.Spec.Template.Spec.Volumes)
				Eventually(func() int {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return len(vm.Status.VolumeSnapshotStatuses)
				}, 180*time.Second, time.Second).Should(Equal(volumes))

				Eventually(func() bool {
					vm2, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					By(fmt.Sprintf("VM Statuses: %+v", vm2.Status))
					return len(vm2.Status.VolumeSnapshotStatuses) == 1 &&
						vm2.Status.VolumeSnapshotStatuses[0].Enabled
				}, 180*time.Second, time.Second).Should(BeTrue())
			})

			It("should error if VolumeSnapshot deleted", func() {
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				cn := snapshot.Status.VirtualMachineSnapshotContentName
				Expect(cn).ToNot(BeNil())
				vmSnapshotContent, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *cn, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmSnapshotContent.Spec.VolumeBackups).To(HaveLen(1))
				vb := vmSnapshotContent.Spec.VolumeBackups[0]
				Expect(vb.VolumeSnapshotName).ToNot(BeNil())

				err = virtClient.KubernetesSnapshotClient().
					SnapshotV1().
					VolumeSnapshots(vm.Namespace).
					Delete(context.Background(), *vb.VolumeSnapshotName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return *snapshot.Status.ReadyToUse
				}, 180*time.Second, time.Second).Should(BeFalse())

				errStr := fmt.Sprintf("VolumeSnapshots (%s) missing", *vb.VolumeSnapshotName)
				Expect(snapshot.Status.Error).ToNot(BeNil())
				Expect(snapshot.Status.Error.Message).To(HaveValue(Equal(errStr)))
			})

			It("should not error if VolumeSnapshot has error", func() {
				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				cn := snapshot.Status.VirtualMachineSnapshotContentName
				Expect(cn).ToNot(BeNil())
				vmSnapshotContent, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *cn, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmSnapshotContent.Spec.VolumeBackups).To(HaveLen(1))
				vb := vmSnapshotContent.Spec.VolumeBackups[0]
				Expect(vb.VolumeSnapshotName).ToNot(BeNil())

				m := "bad stuff"
				Eventually(func() bool {
					vs, err := virtClient.KubernetesSnapshotClient().
						SnapshotV1().
						VolumeSnapshots(vm.Namespace).
						Get(context.Background(), *vb.VolumeSnapshotName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vsc := vs.DeepCopy()
					t := metav1.Now()
					vsc.Status.Error = &vsv1.VolumeSnapshotError{
						Time:    &t,
						Message: &m,
					}

					_, err = virtClient.KubernetesSnapshotClient().
						SnapshotV1().
						VolumeSnapshots(vs.Namespace).
						UpdateStatus(context.Background(), vsc, metav1.UpdateOptions{})
					if errors.IsConflict(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return true
				}, 180*time.Second, time.Second).Should(BeTrue())

				Eventually(func() bool {
					vmSnapshotContent, err = virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), *cn, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					vss := vmSnapshotContent.Status.VolumeSnapshotStatus[0]
					if vss.Error != nil {
						Expect(*vss.Error.Message).To(Equal(m))
						Expect(vmSnapshotContent.Status.Error).To(BeNil())
						Expect(*vmSnapshotContent.Status.ReadyToUse).To(BeTrue())
						return true
					}
					return false
				}, 180*time.Second, time.Second).Should(BeTrue())

				snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(snapshot.Status.Error).To(BeNil())
				Expect(*snapshot.Status.ReadyToUse).To(BeTrue())
			})

			It("[test_id:6952]snapshot change phase to in progress and succeeded and then should not fail", func() {
				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()
				snapshot.Spec.FailureDeadline = &metav1.Duration{Duration: time.Minute}

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() *snapshotv1.VirtualMachineSnapshotStatus {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return snapshot.Status
				}, 30*time.Second, 2*time.Second).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Conditions": ContainElements(
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionReady),
							"Status": Equal(corev1.ConditionFalse)}),
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionProgressing),
							"Status": Equal(corev1.ConditionTrue),
							"Reason": Equal("Source locked and operation in progress")}),
					),
					"Phase": Equal(snapshotv1.InProgress),
				})))

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedVM.Status.SnapshotInProgress).To(Equal(snapshot.Name))

				Expect(snapshot.Status.CreationTime).To(BeNil())

				contentName := fmt.Sprintf("%s-%s", vmSnapshotContent, snapshot.UID)
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(content.Status).To(BeNil())

				deleteWebhook()

				snapshot = waitSnapshotSucceeded(snapshot.Name)

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				content, err = virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))

				// Sleep to pass the time of the deadline
				time.Sleep(time.Second)
				// If snapshot succeeded it should not change to failure when deadline exceeded
				Expect(snapshot.Status.Phase).To(Equal(snapshotv1.Succeeded))
			})

			It("[test_id:6838]snapshot should fail when deadline exceeded due to volume snapshots failure", func() {
				createDenyVolumeSnapshotCreateWebhook()
				snapshot = newSnapshot()
				snapshot.Spec.FailureDeadline = &metav1.Duration{Duration: 40 * time.Second}

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				contentName := fmt.Sprintf("%s-%s", vmSnapshotContent, snapshot.UID)
				Eventually(func() *snapshotv1.VirtualMachineSnapshotStatus {
					snapshot, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					_, contentErr := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
					if !errors.IsNotFound(contentErr) {
						_, _ = fmt.Fprintf(GinkgoWriter, "Content error is not 'not found' %v", contentErr)
						return nil
					}
					return snapshot.Status
				}, time.Minute, 2*time.Second).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Conditions": ContainElements(
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionReady),
							"Status": Equal(corev1.ConditionFalse)}),
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionProgressing),
							"Status": Equal(corev1.ConditionFalse),
							"Reason": Equal(snapshotDeadlineExceeded)}),
						gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Type":   Equal(snapshotv1.ConditionFailure),
							"Status": Equal(corev1.ConditionTrue),
							"Reason": Equal(snapshotDeadlineExceeded)}),
					),
					"Phase": Equal(snapshotv1.Failed),
				})))
				Eventually(matcher.ThisVM(vm), 30*time.Second, 2*time.Second).Should(
					And(
						WithTransform(func(vm *v1.VirtualMachine) *string {
							return vm.Status.SnapshotInProgress
						}, BeNil()),
						WithTransform(func(vm *v1.VirtualMachine) []string {
							return vm.Finalizers
						}, BeEquivalentTo([]string{v1.VirtualMachineControllerFinalizer}))),
					"SnapshotInProgress should be empty")

				Expect(snapshot.Status.CreationTime).To(BeNil())
			})
		})

		Context("[Serial]With more complicated VM with/out GC of succeeded DV", Serial, func() {
			var originalTTL *int32

			BeforeEach(func() {
				cdi := libstorage.GetCDI(virtClient)
				originalTTL = cdi.Spec.Config.DataVolumeTTLSeconds
			})

			AfterEach(func() {
				libstorage.SetDataVolumeGC(virtClient, originalTTL)
			})

			DescribeTable("should successfully create a snapshot", func(ttl *int32) {
				libstorage.SetDataVolumeGC(virtClient, ttl)

				running := false
				vm = tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce,
				)
				vm.Spec.Running = &running

				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					waitDataVolumePopulated(vm.Namespace, dvt.Name)
				}

				snapshot = newSnapshot()

				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()

				Expect(snapshot.Status.CreationTime).ToNot(BeNil())
				contentName := *snapshot.Status.VirtualMachineSnapshotContentName
				content, err := virtClient.VirtualMachineSnapshotContent(vm.Namespace).Get(context.Background(), contentName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(*content.Spec.VirtualMachineSnapshotName).To(Equal(snapshot.Name))
				Expect(content.Spec.Source.VirtualMachine.Spec).To(Equal(vm.Spec))
				Expect(content.Spec.VolumeBackups).Should(HaveLen(len(vm.Spec.DataVolumeTemplates)))

				for _, vol := range vm.Spec.Template.Spec.Volumes {
					if vol.DataVolume == nil {
						continue
					}
					found := false
					for _, vb := range content.Spec.VolumeBackups {
						if vol.DataVolume.Name == vb.PersistentVolumeClaim.Name {
							found = true
							Expect(vol.Name).To(Equal(vb.VolumeName))

							pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), vol.DataVolume.Name, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(pvc.Spec).To(Equal(vb.PersistentVolumeClaim.Spec))

							Expect(vb.VolumeSnapshotName).ToNot(BeNil())
							vs, err := virtClient.
								KubernetesSnapshotClient().
								SnapshotV1().
								VolumeSnapshots(vm.Namespace).
								Get(context.Background(), *vb.VolumeSnapshotName, metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							Expect(*vs.Spec.Source.PersistentVolumeClaimName).Should(Equal(vol.DataVolume.Name))
							Expect(vs.Labels["snapshot.kubevirt.io/source-vm-name"]).Should(Equal(vm.Name))
							Expect(vs.Status.Error).To(BeNil())
							Expect(*vs.Status.ReadyToUse).To(BeTrue())
						}
					}
					Expect(found).To(BeTrue())
				}
			},
				Entry("[test_id:4611] without DV garbage collection", pointer.Int32(-1)),
				Entry("[test_id:8668] with DV garbage collection", pointer.Int32(0)),
			)
		})

		Context("with independent DataVolume", func() {
			var dv *cdiv1.DataVolume

			AfterEach(func() {
				libstorage.DeleteDataVolume(&dv)
			})

			DescribeTable("should accurately report DataVolume provisioning", func(vmif func(string) *v1.VirtualMachineInstance) {
				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(snapshotStorageClass)),
				)

				vmi := vmif(dataVolume.Name)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return len(vm.Status.VolumeSnapshotStatuses) == 1 &&
						!vm.Status.VolumeSnapshotStatuses[0].Enabled
				}, 180*time.Second, 1*time.Second).Should(BeTrue())

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return len(vm.Status.VolumeSnapshotStatuses) == 1 &&
						vm.Status.VolumeSnapshotStatuses[0].Enabled
				}, 180*time.Second, 1*time.Second).Should(BeTrue())
			},
				Entry("with DataVolume volume", tests.NewRandomVMIWithDataVolume),
				Entry("with PVC volume", tests.NewRandomVMIWithPVC),
			)

			It("[test_id:9705]Should show included and excluded volumes in the snapshot", func() {
				noSnapshotSC := libstorage.GetNoVolumeSnapshotStorageClass("local")
				if noSnapshotSC == "" {
					Skip("Skipping test, no storage class without snapshot support")
				}
				By("Creating DV with snapshot supported storage class")
				includedDataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(snapshotStorageClass)),
				)
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), includedDataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				waitDataVolumePopulated(dv.Namespace, dv.Name)

				By("Creating DV with no snapshot supported storage class")
				excludedDataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(noSnapshotSC)),
				)
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), excludedDataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi := tests.NewRandomVMI()
				vmi = tests.AddPVCDisk(vmi, "snapshotablevolume", v1.DiskBusVirtio, includedDataVolume.Name)
				vmi = tests.AddPVCDisk(vmi, "notsnapshotablevolume", v1.DiskBusVirtio, excludedDataVolume.Name)
				vm = tests.NewRandomVirtualMachine(vmi, false)

				_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				volumeSnapshotStatusAsExpected := func(volumeSnapshotStatuses []v1.VolumeSnapshotStatus) bool {
					return len(volumeSnapshotStatuses) == 2 &&
						volumeSnapshotStatuses[0].Enabled &&
						!volumeSnapshotStatuses[1].Enabled
				}
				Eventually(func() []v1.VolumeSnapshotStatus {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.VolumeSnapshotStatuses
				}, 180*time.Second, 3*time.Second).WithOffset(1).Should(Satisfy(volumeSnapshotStatusAsExpected))

				By("Create Snapshot")
				snapshot = newSnapshot()
				_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				snapshot = waitSnapshotSucceeded(snapshot.Name)
				Expect(snapshot.Status.SnapshotVolumes.IncludedVolumes).Should(HaveLen(1))
				Expect(snapshot.Status.SnapshotVolumes.IncludedVolumes[0]).Should(Equal("snapshotablevolume"))
				Expect(snapshot.Status.SnapshotVolumes.ExcludedVolumes).Should(HaveLen(1))
				Expect(snapshot.Status.SnapshotVolumes.ExcludedVolumes[0]).Should(Equal("notsnapshotablevolume"))
			})
		})

		Context("With VM using instancetype and preferences", func() {

			var instancetype *instancetypev1beta1.VirtualMachineInstancetype

			BeforeEach(func() {
				instancetype = &instancetypev1beta1.VirtualMachineInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "vm-instancetype-",
						Namespace:    testsuite.GetTestNamespace(nil),
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: 1,
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}
				instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(nil)).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm = tests.NewRandomVMWithDataVolumeWithRegistryImport(
					cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine),
					testsuite.GetTestNamespace(nil),
					snapshotStorageClass,
					corev1.ReadWriteOnce,
				)
				vm.Spec.Template.Spec.Domain.Resources = v1.ResourceRequirements{}
				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: instancetype.Name,
					Kind: "VirtualMachineInstanceType",
				}
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				for _, dvt := range vm.Spec.DataVolumeTemplates {
					waitDataVolumePopulated(vm.Namespace, dvt.Name)
				}
			})

			DescribeTable("Bug #8435 - should create a snapshot successfully", func(toRunSourceVM bool) {
				if toRunSourceVM {
					By("Starting the VM and expecting it to run")
					vm = tests.StartVMAndExpectRunning(virtClient, vm)
				}

				snapshot = newSnapshot()
				snapshot, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				waitSnapshotReady()
			},
				Entry("with running source VM", true),
				Entry("with stopped source VM", false),
			)
		})
	})
})

func AddVolumeAndVerify(virtClient kubecli.KubevirtClient, storageClass string, vm *v1.VirtualMachine, addVMIOnly bool) string {
	dv := libdv.NewDataVolume(
		libdv.WithBlankImageSource(),
		libdv.WithPVC(libdv.PVCWithStorageClass(storageClass), libdv.PVCWithVolumeSize(cd.BlankVolumeSize)),
	)

	var err error
	dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	libstorage.EventuallyDV(dv, 240, matcher.HaveSucceeded())
	volumeSource := &v1.HotplugVolumeSource{
		DataVolume: &v1.DataVolumeSource{
			Name: dv.Name,
		},
	}
	addVolumeName := "test-volume-" + rand.String(12)
	addVolumeOptions := &v1.AddVolumeOptions{
		Name: addVolumeName,
		Disk: &v1.Disk{
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.DiskBusSCSI,
				},
			},
			Serial: addVolumeName,
		},
		VolumeSource: volumeSource,
	}

	if addVMIOnly {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(vm.Namespace).AddVolume(context.Background(), vm.Name, addVolumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	} else {
		Eventually(func() error {
			return virtClient.VirtualMachine(vm.Namespace).AddVolume(context.Background(), vm.Name, addVolumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		verifyVolumeAndDiskVMAdded(virtClient, vm, addVolumeName)
	}

	vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	verifyVolumeAndDiskVMIAdded(virtClient, vmi, addVolumeName)

	return addVolumeName
}
