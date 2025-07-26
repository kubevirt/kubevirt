package vmi_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller/vmi"
)

var _ = Describe("Conditions", func() {
	var collector vmi.ConditionsCollector

	BeforeEach(func() {
		collector = vmi.ConditionsCollector{}
	})

	It("should describe metrics", func() {
		metrics := collector.Describe()
		Expect(metrics).To(HaveLen(1))
	})

	Context("with VMI", func() {
		It("should collect no metrics when VMI has no conditions", func() {
			vmi := &k6tv1.VirtualMachineInstance{}
			results := collector.Collect(vmi)
			Expect(results).To(BeEmpty())
		})

		It("should collect metrics for VMI conditions", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				Status: k6tv1.VirtualMachineInstanceStatus{
					Conditions: []k6tv1.VirtualMachineInstanceCondition{
						{
							Type:    k6tv1.VirtualMachineInstanceSynchronized,
							Status:  k8sv1.ConditionFalse,
							Reason:  "Synchronizing with the Domain failed.",
							Message: "server error. command SyncVMI failed: 'LibvirtError(Code=1, Domain=10, ...)'",
						},
						{
							Type:    k6tv1.VirtualMachineInstanceDataVolumesReady,
							Status:  k8sv1.ConditionTrue,
							Reason:  "AllDVsReady",
							Message: "All of the VMI's DVs are bound and not running",
						},
					},
				},
			}

			results := collector.Collect(vmi)
			Expect(results).To(HaveLen(2))

			Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmi_conditions"))
			Expect(results[0].Labels).To(Equal([]string{
				vmi.Namespace,
				vmi.Name,
				string(k6tv1.VirtualMachineInstanceSynchronized),
				"Synchronizing with the Domain failed.",
				"server error. command SyncVMI failed: 'LibvirtError(Code=1, Domain=10, ...)'",
			}))
			Expect(results[0].Value).To(Equal(0.0))

			Expect(results[1].Metric.GetOpts().Name).To(Equal("kubevirt_vmi_conditions"))
			Expect(results[1].Labels).To(Equal([]string{
				vmi.Namespace,
				vmi.Name,
				string(k6tv1.VirtualMachineInstanceDataVolumesReady),
				"AllDVsReady",
				"All of the VMI's DVs are bound and not running",
			}))
			Expect(results[1].Value).To(Equal(1.0))
		})
	})

})
