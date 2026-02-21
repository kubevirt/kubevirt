/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package storage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	backupv1alpha1 "kubevirt.io/api/backup/v1alpha1"

	"kubevirt.io/kubevirt/pkg/crdvalidation"
)

const (
	msgAPIGroupMustBeKubevirt   = "apiGroup must be kubevirt.io"
	msgSpecImmutableAfterCreate = "spec is immutable after creation"
)

var _ = Describe("VirtualMachineBackup Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Enum validations", func() {
		It("should reject invalid mode enum value", func() {
			invalidMode := backupv1alpha1.BackupMode("InvalidMode")
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
					Mode: &invalidMode,
				},
			}

			errs := validator.Validate("virtualmachinebackup", backup)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		It("should accept valid mode enum value - Push", func() {
			mode := backupv1alpha1.PushMode
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
					Mode:    &mode,
					PvcName: ptr.To("backup-pvc"),
				},
			}

			errs := validator.Validate("virtualmachinebackup", backup)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).To(BeEmpty())
		})
	})

	Context("CEL validations", func() {
		It("should pass when CEL rule is satisfied", func() {
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			errs := validator.Validate("virtualmachinebackup", backup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			// Check for apiGroup related CEL errors
			var apiGroupErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == msgAPIGroupMustBeKubevirt {
					apiGroupErrors = append(apiGroupErrors, e)
				}
			}
			Expect(apiGroupErrors).To(BeEmpty())
		})

		It("should fail when apiGroup is not kubevirt.io or backup.kubevirt.io", func() {
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("invalid.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			errs := validator.Validate("virtualmachinebackup", backup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			Expect(celErrs).ToNot(BeEmpty())
		})

		It("should fail when name is empty", func() {
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "",
					},
				},
			}

			errs := validator.Validate("virtualmachinebackup", backup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			var nameErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == "name is required" {
					nameErrors = append(nameErrors, e)
				}
			}
			Expect(nameErrors).ToNot(BeEmpty())
		})

		It("should fail when kind doesn't match apiGroup", func() {
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachineBackupTracker", // Wrong kind for kubevirt.io
						Name:     "my-vm",
					},
				},
			}

			errs := validator.Validate("virtualmachinebackup", backup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			Expect(celErrs).ToNot(BeEmpty())
		})

		//nolint:dupl
		It("should detect immutability violation on update", func() {
			oldBackup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			newBackup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "different-vm", // Changed!
					},
				},
			}

			errs := validator.ValidateUpdate("virtualmachinebackup", newBackup, oldBackup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			// Should have an immutability error
			var immutabilityErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == msgSpecImmutableAfterCreate {
					immutabilityErrors = append(immutabilityErrors, e)
				}
			}
			Expect(immutabilityErrors).ToNot(BeEmpty())
		})

		It("should allow update when spec is unchanged", func() {
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			errs := validator.ValidateUpdate("virtualmachinebackup", backup, backup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			// No immutability errors since spec is the same
			var immutabilityErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == msgSpecImmutableAfterCreate {
					immutabilityErrors = append(immutabilityErrors, e)
				}
			}
			Expect(immutabilityErrors).To(BeEmpty())
		})

		It("should skip transition rules on create (no oldSelf)", func() {
			backup := &backupv1alpha1.VirtualMachineBackup{
				Spec: backupv1alpha1.VirtualMachineBackupSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			// Validate without oldObj simulates a create operation
			errs := validator.Validate("virtualmachinebackup", backup)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			// Should NOT have immutability errors on create
			var immutabilityErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == msgSpecImmutableAfterCreate {
					immutabilityErrors = append(immutabilityErrors, e)
				}
			}
			Expect(immutabilityErrors).To(BeEmpty())
		})
	})
})

var _ = Describe("VirtualMachineBackupTracker Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("CEL validations", func() {
		//nolint:dupl
		It("should detect immutability violation on update", func() {
			oldTracker := &backupv1alpha1.VirtualMachineBackupTracker{
				Spec: backupv1alpha1.VirtualMachineBackupTrackerSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			newTracker := &backupv1alpha1.VirtualMachineBackupTracker{
				Spec: backupv1alpha1.VirtualMachineBackupTrackerSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "different-vm",
					},
				},
			}

			errs := validator.ValidateUpdate("virtualmachinebackuptracker", newTracker, oldTracker)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			var immutabilityErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == msgSpecImmutableAfterCreate {
					immutabilityErrors = append(immutabilityErrors, e)
				}
			}
			Expect(immutabilityErrors).ToNot(BeEmpty())
		})

		It("should fail when apiGroup is not kubevirt.io", func() {
			tracker := &backupv1alpha1.VirtualMachineBackupTracker{
				Spec: backupv1alpha1.VirtualMachineBackupTrackerSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("invalid.io"),
						Kind:     "VirtualMachine",
						Name:     "my-vm",
					},
				},
			}

			errs := validator.Validate("virtualmachinebackuptracker", tracker)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			var apiGroupErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == msgAPIGroupMustBeKubevirt {
					apiGroupErrors = append(apiGroupErrors, e)
				}
			}
			Expect(apiGroupErrors).ToNot(BeEmpty())
		})

		It("should fail when kind is not VirtualMachine", func() {
			tracker := &backupv1alpha1.VirtualMachineBackupTracker{
				Spec: backupv1alpha1.VirtualMachineBackupTrackerSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "Pod",
						Name:     "my-vm",
					},
				},
			}

			errs := validator.Validate("virtualmachinebackuptracker", tracker)
			celErrs := errs.ByType(crdvalidation.ErrorTypeCEL)
			var kindErrors crdvalidation.ValidationErrors
			for _, e := range celErrs {
				if e.Message == "kind must be VirtualMachine" {
					kindErrors = append(kindErrors, e)
				}
			}
			Expect(kindErrors).ToNot(BeEmpty())
		})
	})
})
