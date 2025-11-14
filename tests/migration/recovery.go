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
	"k8s.io/client-go/kubernetes"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet/job"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]Migration recovery", decorators.SigCompute, decorators.RequiresRWOFsVMStateStorageClass, func() {
	DescribeTable("should successfully defer a migration", func(fakeSuccess, flakeCheck bool) {
		virtClient, err := kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred())

		if flakeCheck {
			kv := getCurrentKvConfig(virtClient)
			var exists bool
			kv.VMStateStorageClass, exists = libstorage.GetRWOFileSystemStorageClass()
			Expect(exists).To(BeTrue())
			config.UpdateKubeVirtConfigValueAndWait(kv)
		}

		By("Creating a VM with RWO backend-storage")
		vmi := libvmifact.NewFedora(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
			libvmi.WithTPM(true),
		)
		vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyManual))
		vm, err = virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm, k8smeta.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		if fakeSuccess {
			By("Creating the backend-storage PVC ourselves to be able to alter it before the VM starts")
			pvc := createPVCFor(k8s.Client(), vm)
			By("Simulating a migration success by manually adding /meta/migrated to the source PVC")
			fakeMigrationSuccessInPVC(k8s.Client(), pvc.Name, vm.Namespace)
		}

		By("Creating a migration policy for that VMI")
		migrationPolicy := PreparePolicyAndVMIWithNSAndVMILabelsWithPreexistingPolicy(vmi, nil, 1, 0, nil)
		migrationPolicy.Spec.BandwidthPerMigration = pointer.P(resource.MustParse("1Ki"))
		CreateMigrationPolicy(virtClient, migrationPolicy)

		By("Starting the VM")
		vm = libvmops.StartVirtualMachine(vm)
		vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running})
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

		By("Stressing")
		runStressTest(vmi, stressDefaultVMSize)

		By("Starting a slow migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigration(virtClient, migration)
		Eventually(func() (*v1.VirtualMachineInstanceMigration, error) {
			return virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(context.Background(), migration.Name, k8smeta.GetOptions{})
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))

		// Update the migration variable
		migration, err = virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(context.Background(), migration.Name, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Killing the source pod")
		err = k8s.Client().CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pod.Name, k8smeta.DeleteOptions{
			GracePeriodSeconds: pointer.P(int64(0)),
		})
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the VMI to be gone or failed")
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(time.Minute).WithPolling(time.Second).Should(Or(
			matcher.BeGone(),
			matcher.BeInPhase(v1.Failed),
		))

		sourcePVC := migration.Status.MigrationState.SourcePersistentStatePVCName
		targetPVC := migration.Status.MigrationState.TargetPersistentStatePVCName

		keptPVC := sourcePVC
		nukedPVC := targetPVC
		if fakeSuccess {
			keptPVC = targetPVC
			nukedPVC = sourcePVC
		}

		By("Starting the VM again")
		vm = libvmops.StartVirtualMachine(vm)

		By("Expecting the right PVC to be removed")
		Eventually(func() error {
			_, err = k8s.Client().CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), nukedPVC, k8smeta.GetOptions{})
			return err
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Expecting the right PVC to be preserved")
		pvc, err := k8s.Client().CoreV1().PersistentVolumeClaims(migration.Namespace).Get(context.Background(), keptPVC, k8smeta.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvc.Labels).To(HaveKeyWithValue("persistent-state-for", vmi.Name))
	},
		Entry("failure", decorators.NoFlakeCheck, false, false),
		Entry("success", decorators.NoFlakeCheck, true, false),
		Entry("failure [Serial]", decorators.FlakeCheck, Serial, false, true),
		Entry("success [Serial]", decorators.FlakeCheck, Serial, true, true),
	)
})

func createPVCFor(k8sClient kubernetes.Interface, vm *v1.VirtualMachine) *k8score.PersistentVolumeClaim {
	storageClass, exists := libstorage.GetRWOFileSystemStorageClass()
	Expect(exists).To(BeTrue())
	mode := k8score.PersistentVolumeFilesystem
	accessMode := k8score.ReadWriteOnce
	ownerReferences := []k8smeta.OwnerReference{
		*k8smeta.NewControllerRef(vm, v1.VirtualMachineGroupVersionKind),
	}
	pvc := &k8score.PersistentVolumeClaim{
		ObjectMeta: k8smeta.ObjectMeta{
			GenerateName:    backendstorage.PVCPrefix + "-" + vm.Name + "-",
			OwnerReferences: ownerReferences,
			Labels:          map[string]string{backendstorage.PVCPrefix: vm.Name},
		},
		Spec: k8score.PersistentVolumeClaimSpec{
			AccessModes: []k8score.PersistentVolumeAccessMode{accessMode},
			Resources: k8score.VolumeResourceRequirements{
				Requests: k8score.ResourceList{k8score.ResourceStorage: resource.MustParse(backendstorage.PVCSize)},
			},
			StorageClassName: &storageClass,
			VolumeMode:       &mode,
		},
	}

	pvc, err := k8sClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvc, k8smeta.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	return pvc
}

func fakeMigrationSuccessInPVC(k8sClient kubernetes.Interface, pvcName, namespace string) {
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

	fakeSuccessJob, err = k8sClient.BatchV1().Jobs(namespace).Create(context.Background(), fakeSuccessJob, k8smeta.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	By("Waiting for the job to succeed")
	err = job.WaitForJobToSucceed(fakeSuccessJob, time.Minute)
	Expect(err).NotTo(HaveOccurred())

	By("Removing the job")
	// Job is auto-removed after 90 seconds, might already be gone, deleting anyway to free PVC
	_ = k8sClient.BatchV1().Jobs(namespace).Delete(context.Background(), fakeSuccessJob.Name, k8smeta.DeleteOptions{
		PropagationPolicy: pointer.P(k8smeta.DeletePropagationBackground),
	})
}
