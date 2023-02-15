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

	Context("WithNetworkResources option", func() {
		It("does not request / set any limit when no network resources are required", func() {
			rr = NewResourceRenderer(
				nil,
				nil,
				WithNetworkResources(map[string]string{}),
			)
			Expect(rr.Limits()).To(BeEmpty())
			Expect(rr.Requests()).To(BeEmpty())
		})

		It("adds a request and sets limit for each multus network resource", func() {
			netToResourceMap := map[string]string{
				"net1": "res1",
				"net2": "res44",
			}
			rr = NewResourceRenderer(
				nil,
				nil,
				WithNetworkResources(netToResourceMap),
			)
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceName("res1"), *resource.NewScaledQuantity(1, 0)))
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceName("res44"), *resource.NewScaledQuantity(1, 0)))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceName("res1"), *resource.NewScaledQuantity(1, 0)))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceName("res44"), *resource.NewScaledQuantity(1, 0)))
		})
	})

	Context("WithHostDevices / WithGPU option", func() {
		It("host device requests / limits are absent when not requested", func() {
			rr = NewResourceRenderer(
				nil,
				nil,
				WithHostDevices([]v1.HostDevice{}),
			)
			Expect(rr.Limits()).To(BeEmpty())
			Expect(rr.Requests()).To(BeEmpty())
		})

		It("host device requests / limits are honored", func() {
			hostDevice := v1.HostDevice{
				Name:       "hd1",
				DeviceName: "discombobulator2000",
				Tag:        "not-so-megatag",
			}
			hostDevices := []v1.HostDevice{hostDevice}
			rr = NewResourceRenderer(
				nil,
				nil,
				WithHostDevices(hostDevices),
			)
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceName("discombobulator2000"), *resource.NewScaledQuantity(1, 0)))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceName("discombobulator2000"), *resource.NewScaledQuantity(1, 0)))
		})

		It("GPU requests / limits are absent when not requested", func() {
			rr = NewResourceRenderer(
				nil,
				nil,
				WithGPUs([]v1.GPU{}),
			)
			Expect(rr.Limits()).To(BeEmpty())
			Expect(rr.Requests()).To(BeEmpty())
		})

		It("GPU requests / limits are honored", func() {
			gp1 := v1.GPU{
				Name:       "gp1",
				DeviceName: "discombobulator2000",
				Tag:        "megatag",
			}
			requestedGPUs := []v1.GPU{gp1}
			rr = NewResourceRenderer(
				nil,
				nil,
				WithGPUs(requestedGPUs))
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceName("discombobulator2000"), *resource.NewScaledQuantity(1, 0)))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceName("discombobulator2000"), *resource.NewScaledQuantity(1, 0)))
		})
	})

	It("WithSEV option adds ", func() {
		sevResourceKey := kubev1.ResourceName("devices.kubevirt.io/sev")
		rr = NewResourceRenderer(nil, nil, WithSEV())
		Expect(rr.Requests()).To(Equal(kubev1.ResourceList{
			sevResourceKey: *resource.NewQuantity(1, resource.DecimalSI),
		}))
		Expect(rr.Limits()).To(Equal(kubev1.ResourceList{
			sevResourceKey: *resource.NewQuantity(1, resource.DecimalSI),
		}))
	})

	defaultRequest := func() kubev1.ResourceList {
		return kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("10m"),
			kubev1.ResourceMemory: resource.MustParse("2M"),
		}
	}

	zeroQuantity := func() *resource.Quantity {
		return resource.NewQuantity(0, resource.DecimalSI)
	}

	DescribeTable("Calculate ratios from VMI", func(reqMem, reqCpu, limMem, limCpu *resource.Quantity, expectedRequest, expectedLimits kubev1.ResourceList) {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{
						Requests: kubev1.ResourceList{},
						Limits:   kubev1.ResourceList{},
					},
				},
			},
		}
		if reqMem != nil {
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = *reqMem
		}
		if reqCpu != nil {
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceCPU] = *reqCpu
		}
		if limMem != nil {
			vmi.Spec.Domain.Resources.Limits[kubev1.ResourceMemory] = *limMem
		}
		if limCpu != nil {
			vmi.Spec.Domain.Resources.Limits[kubev1.ResourceCPU] = *limCpu
		}
		res := hotplugContainerResourceRequirementsForVMI(vmi)
		memRes := res.Requests[kubev1.ResourceMemory]
		memRes.RoundUp(resource.Mega)
		Expect(memRes.String()).ToNot(BeEmpty())
		expMemRes := expectedRequest[kubev1.ResourceMemory]
		Expect(expMemRes).To(Equal(memRes))

		cpuRes := res.Requests[kubev1.ResourceCPU]
		cpuRes.RoundUp(resource.Milli)
		Expect(cpuRes.String()).ToNot(BeEmpty())
		expCpuRes := expectedRequest[kubev1.ResourceCPU]
		Expect(expCpuRes).To(Equal(cpuRes))

		Expect(res.Limits).To(BeEquivalentTo(expectedLimits))
	},
		Entry("Nil everything", nil, nil, nil, nil, defaultRequest(), hotplugContainerLimits()),
		Entry("Zero memory request/limit, nil cpu request/limit", zeroQuantity(), nil, zeroQuantity(), nil, defaultRequest(), hotplugContainerLimits()),
		Entry("Zero everything", zeroQuantity(), zeroQuantity(), zeroQuantity(), zeroQuantity(), defaultRequest(), hotplugContainerLimits()),
		Entry("Nil memory request/limit, zero cpu request/limit", nil, zeroQuantity(), nil, zeroQuantity(), defaultRequest(), hotplugContainerLimits()),
		Entry("Memory request and limit same, nil cpu request/limit", resource.NewQuantity(10, resource.DecimalSI), nil, resource.NewQuantity(10, resource.DecimalSI), nil, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("10m"),
			kubev1.ResourceMemory: resource.MustParse("80M"),
		}, hotplugContainerLimits()),
		Entry("Cpu request and limit same, nil mem request/limit", nil, resource.NewQuantity(2, resource.DecimalSI), nil, resource.NewQuantity(2, resource.DecimalSI), kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("100m"),
			kubev1.ResourceMemory: resource.MustParse("2M"),
		}, hotplugContainerLimits()),
		Entry("Memory request and limit at ratio 2, nil cpu request/limit", resource.NewQuantity(10, resource.DecimalSI), nil, resource.NewQuantity(20, resource.DecimalSI), nil, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("10m"),
			kubev1.ResourceMemory: resource.MustParse("40M"),
		}, hotplugContainerLimits()),
		Entry("Cpu request and limit at ratio 2, nil mem request/limit", nil, resource.NewQuantity(2, resource.DecimalSI), nil, resource.NewQuantity(4, resource.DecimalSI), kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("50m"),
			kubev1.ResourceMemory: resource.MustParse("2M"),
		}, hotplugContainerLimits()),
		Entry("Memory request and limit at ratio 3, nil cpu request/limit", resource.NewQuantity(10, resource.DecimalSI), nil, resource.NewQuantity(30, resource.DecimalSI), nil, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("10m"),
			kubev1.ResourceMemory: resource.MustParse("27M"),
		}, hotplugContainerLimits()),
		Entry("Cpu request and limit at ratio 3, nil mem request/limit", nil, resource.NewQuantity(2, resource.DecimalSI), nil, resource.NewQuantity(6, resource.DecimalSI), kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("34m"),
			kubev1.ResourceMemory: resource.MustParse("2M"),
		}, hotplugContainerLimits()),
	)
})

func addResources(firstQuantity resource.Quantity, resources ...resource.Quantity) resource.Quantity {
	for _, resourceQuantity := range resources {
		firstQuantity.Add(resourceQuantity)
	}
	return firstQuantity
}
