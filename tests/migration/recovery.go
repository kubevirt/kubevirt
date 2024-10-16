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

package migration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	batchv1 "k8s.io/api/batch/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet/job"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// The following tests require the backend-storage storage class to be RWO (recovery doesn't apply to RWX).
// The flake-check lane sets up a default storage class that is RWX.
// For these tests to run on that lane, they would have to configure a RWO storage class in the CR,
//
//	and therefore be serial, which would make them a lot more expensive.
//
// TODO: maybe we should use a flag set by the flake-checker lane to make the tests serial and CR-altering
var _ = Describe("[sig-compute]Migration recovery", decorators.SigCompute, decorators.NoFlakeCheck, func() {

	DescribeTable("should successfully defer a migration", func(fakeSuccess bool) {
		virtClient, err := kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred())

		By("Creating a VMI with RWO backend-storage")
		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{
			Persistent: pointer.P(true),
		}
		vmi.Namespace = testsuite.GetTestNamespace(vmi)

		By("Creating a migration policy for that VMI")
		migrationPolicy := PreparePolicyAndVMIWithNSAndVMILabelsWithPreexistingPolicy(vmi, nil, 1, 0, nil)
		migrationPolicy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("1Mi"))
		CreateMigrationPolicy(virtClient, migrationPolicy)

		By("Starting the VMI as a VM")
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyManual))
		vm, err = virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm, k8smeta.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		vm = libvmops.StartVirtualMachine(vm)
		vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running})
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

		By("Stressing")
		runStressTest(vmi, stressDefaultVMSize, 42)

		By("Starting a slow migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigration(virtClient, migration)
		Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
			migration, err = virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(context.Background(), migration.Name, k8smeta.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			return migration.Status.Phase
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(v1.MigrationRunning))

		By("Killing the source pod")
		err = virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pod.Name, k8smeta.DeleteOptions{
			GracePeriodSeconds: pointer.P(int64(0)),
		})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the VMI to be gone or failed")
		Eventually(func() string {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, k8smeta.GetOptions{})
			if err != nil {
				return err.Error()
			}
			return string(vmi.Status.Phase)
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Or(
			ContainSubstring("the server could not find the requested resource"),
			Equal(string(v1.Failed)),
		))

		By("Expecting the migration object, source and target PVCs to still exist")
		migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(migration.Status.MigrationState).NotTo(BeNil())
		Expect(migration.Status.Phase).To(Equal(v1.MigrationInterrupted))
		sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
		targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName
		Expect(sourcePVC).NotTo(BeEmpty())
		Expect(targetPVC).NotTo(BeEmpty())
		Expect(sourcePVC).NotTo(Equal(targetPVC), "This test can't run on RWX storage")
		_, err = virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), sourcePVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		_, err = virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), targetPVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		keptPVC := sourcePVC
		nukedPVC := targetPVC
		if fakeSuccess {
			By("Simulating a migration success by manually adding /meta/migrated to the source PVC")
			fakeMigrationSuccessInPVC(virtClient, sourcePVC, migration.Namespace)

			keptPVC = targetPVC
			nukedPVC = sourcePVC
		}

		By("Starting the VM again")
		vm = libvmops.StartVirtualMachine(vm)

		By("Expecting the right PVC to be removed")
		Eventually(func() error {
			_, err = virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), nukedPVC, k8smeta.GetOptions{})
			return err
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Expecting the right PVC to be preserved")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), keptPVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvc.Labels).To(HaveKeyWithValue("persistent-state-for", vmi.Name))
	},
		Entry("failure", false),
		Entry("success", true),
	)
})

func fakeMigrationSuccessInPVC(virtClient kubecli.KubevirtClient, pvcName, namespace string) {
	var err error

	By("Creating a job")
	fakeSuccessJob := &batchv1.Job{
		ObjectMeta: k8smeta.ObjectMeta{
			GenerateName: "migration-success-faker-",
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds:   pointer.P(int64(90)),
			BackoffLimit:            pointer.P(int32(1)),
			TTLSecondsAfterFinished: pointer.P(int32(90)),
			Template: k8score.PodTemplateSpec{
				ObjectMeta: k8smeta.ObjectMeta{
					GenerateName: "backend-storage-recover-",
				},
				Spec: k8score.PodSpec{
					RestartPolicy: k8score.RestartPolicyNever,
					SecurityContext: &k8score.PodSecurityContext{
						RunAsNonRoot: pointer.P(true),
						RunAsUser:    pointer.P(int64(util.NonRootUID)),
						RunAsGroup:   pointer.P(int64(util.NonRootUID)),
						FSGroup:      pointer.P(int64(util.NonRootUID)),
						SeccompProfile: &k8score.SeccompProfile{
							Type: k8score.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []k8score.Container{{
						Name: "container",
						SecurityContext: &k8score.SecurityContext{
							AllowPrivilegeEscalation: pointer.P(false),
							Capabilities:             &k8score.Capabilities{Drop: []k8score.Capability{"ALL"}},
						},
						Image:   libregistry.GetUtilityImageFromRegistry("vm-killer"), // Any image will do, we just need `touch`
						Command: []string{"touch"},
						Args:    []string{"/meta/migrated"},
						VolumeMounts: []k8score.VolumeMount{{
							Name:      "backend-storage",
							MountPath: "/meta",
							SubPath:   "meta",
						}},
					}},
					Volumes: []k8score.Volume{{
						Name: "backend-storage",
						VolumeSource: k8score.VolumeSource{
							PersistentVolumeClaim: &k8score.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					}},
				},
			},
		},
	}

	fakeSuccessJob, err = virtClient.BatchV1().Jobs(namespace).Create(context.Background(), fakeSuccessJob, k8smeta.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	By("Waiting for the job to succeed")
	err = job.WaitForJobToSucceed(fakeSuccessJob, time.Minute)
	Expect(err).NotTo(HaveOccurred())

	By("Removing the job")
	// Job is auto-removed after 90 seconds, might already be gone, deleting anyway to free PVC
	_ = virtClient.BatchV1().Jobs(namespace).Delete(context.Background(), fakeSuccessJob.Name, k8smeta.DeleteOptions{
		PropagationPolicy: pointer.P(k8smeta.DeletePropagationBackground),
	})
}
