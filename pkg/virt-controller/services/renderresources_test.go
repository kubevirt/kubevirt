package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Resource pod spec renderer", func() {
	var rr *ResourceRenderer

	It("an empty resource renderer does not feature requests nor limits", func() {
		rr = NewResourceRenderer(nil, nil)
		Expect(rr.Requests()).To(BeEmpty())
		Expect(rr.Limits()).To(BeEmpty())
	})

	It("user provided CPU and memory requests are honored", func() {
		requests := kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("1m"),
			kubev1.ResourceMemory: resource.MustParse("64M"),
		}
		rr = NewResourceRenderer(nil, requests)
		Expect(rr.Limits()).To(BeEmpty())
		Expect(rr.Requests()).To(ConsistOf(resource.MustParse("1m"), resource.MustParse("64M")))
	})

	Context("WithEphemeral option", func() {
		It("adds an expected 50M memory overhead", func() {
			thirtyMegabytes := resource.MustParse("30M")
			seventyMegabytes := resource.MustParse("70M")
			ephemeralStorageRequests := kubev1.ResourceList{kubev1.ResourceEphemeralStorage: thirtyMegabytes}
			ephemeralStorageLimit := kubev1.ResourceList{kubev1.ResourceEphemeralStorage: seventyMegabytes}
			ephemeralStorageAddition := resource.MustParse(ephemeralStorageOverheadSize)

			rr = NewResourceRenderer(ephemeralStorageLimit, ephemeralStorageRequests, WithEphemeralStorageRequest())
			Expect(rr.Requests()).To(HaveKeyWithValue(
				kubev1.ResourceEphemeralStorage,
				addResources(thirtyMegabytes, ephemeralStorageAddition),
			))
			Expect(rr.Limits()).To(HaveKeyWithValue(
				kubev1.ResourceEphemeralStorage,
				addResources(seventyMegabytes, ephemeralStorageAddition),
			))
		})
	})

	Context("Default CPU configuration", func() {
		cpu := &v1.CPU{Cores: 5}
		It("Requests one CPU per core, when CPU allocation ratio is 1", func() {
			rr = NewResourceRenderer(nil, nil, WithoutDedicatedCPU(cpu, 1))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("5")))
			Expect(rr.Limits()).To(BeEmpty())
		})

		It("Requests 100m per core, when CPU allocation ratio is 10", func() {
			rr = NewResourceRenderer(nil, nil, WithoutDedicatedCPU(cpu, 10))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("500m")))
			Expect(rr.Limits()).To(BeEmpty())
		})

	})

	Context("WithMemoryOverhead option", func() {
		baseMemory := resource.MustParse("64M")
		memOverhead := resource.MustParse("128M")
		var userSpecifiedMemory kubev1.ResourceList

		BeforeEach(func() {
			userSpecifiedMemory = kubev1.ResourceList{kubev1.ResourceMemory: baseMemory}
		})

		It("the specified overhead is added to the user requested VM memory", func() {
			requestedVMRequirements := v1.ResourceRequirements{
				Requests: nil,
				Limits:   nil,
			}
			rr = NewResourceRenderer(
				userSpecifiedMemory,
				userSpecifiedMemory,
				WithMemoryOverhead(requestedVMRequirements, &memOverhead),
			)
			Expect(rr.Requests()).To(HaveKeyWithValue(
				kubev1.ResourceMemory,
				addResources(baseMemory, memOverhead)))
			Expect(rr.Limits()).To(HaveKeyWithValue(
				kubev1.ResourceMemory,
				addResources(baseMemory, memOverhead)))
		})

		When("the overcommit guest overhead option is specified", func() {
			It("the overhead is *only* added to the VM's limits, not requests", func() {
				rr = NewResourceRenderer(userSpecifiedMemory, userSpecifiedMemory, WithMemoryOverhead(v1.ResourceRequirements{
					Requests:                nil,
					Limits:                  nil,
					OvercommitGuestOverhead: true,
				}, &memOverhead))
				Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceMemory, baseMemory))
				Expect(rr.Limits()).To(HaveKeyWithValue(
					kubev1.ResourceMemory,
					addResources(baseMemory, memOverhead)))
			})
		})
	})

	Context("WithCPUPinning option", func() {
		userCPURequest := resource.MustParse("200m")
		userSpecifiedCPU := kubev1.ResourceList{kubev1.ResourceCPU: userCPURequest}

		It("the user requested CPU configs are *not* overriden", func() {
			rr = NewResourceRenderer(nil, userSpecifiedCPU, WithCPUPinning(&v1.CPU{Cores: 5}))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, userCPURequest))
		})

		It("carries over the CPU limits as requests when no CPUs are requested", func() {
			rr = NewResourceRenderer(userSpecifiedCPU, nil, WithCPUPinning(&v1.CPU{}))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, userCPURequest))
		})

		It("carries over the CPU requests as limits when no CPUs are requested", func() {
			rr = NewResourceRenderer(nil, userSpecifiedCPU, WithCPUPinning(&v1.CPU{}))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, userCPURequest))
		})

		It("carries over the requested memory as a *limit*", func() {
			memoryRequest := resource.MustParse("128M")
			userSpecifiedCPU := kubev1.ResourceList{
				kubev1.ResourceCPU:    userCPURequest,
				kubev1.ResourceMemory: memoryRequest,
			}
			rr = NewResourceRenderer(nil, userSpecifiedCPU, WithCPUPinning(&v1.CPU{Cores: 5}))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("200m")))
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceMemory, memoryRequest))
		})

		When("an isolated emulator thread is requested", func() {
			cpuIsolatedEmulatorThreadOverhead := resource.MustParse("1000m")
			userSpecifiedCPU := kubev1.ResourceList{kubev1.ResourceCPU: userCPURequest}

			It("requires an additional 1000m CPU, and an additional CPU is added to the limits", func() {
				rr = NewResourceRenderer(
					nil,
					userSpecifiedCPU,
					WithCPUPinning(&v1.CPU{
						Cores:                 5,
						IsolateEmulatorThread: true,
					}),
				)
				Expect(rr.Limits()).To(HaveKeyWithValue(
					kubev1.ResourceCPU,
					*resource.NewQuantity(6, resource.BinarySI),
				))
				Expect(rr.Requests()).To(HaveKeyWithValue(
					kubev1.ResourceCPU,
					addResources(userCPURequest, cpuIsolatedEmulatorThreadOverhead),
				))
			})
		})
	})
})

func addResources(firstQuantity resource.Quantity, resources ...resource.Quantity) resource.Quantity {
	for _, resourceQuantity := range resources {
		firstQuantity.Add(resourceQuantity)
	}
	return firstQuantity
}
