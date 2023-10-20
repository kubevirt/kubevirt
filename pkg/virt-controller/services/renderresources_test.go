package services

import (
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
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
			rr = NewResourceRenderer(nil, nil, WithoutDedicatedCPU(cpu, 1, false))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("5")))
			Expect(rr.Limits()).To(BeEmpty())
		})

		It("Requests 100m per core, when CPU allocation ratio is 10", func() {
			rr = NewResourceRenderer(nil, nil, WithoutDedicatedCPU(cpu, 10, false))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("500m")))
			Expect(rr.Limits()).To(BeEmpty())
		})
		It("Limits to one CPU per core, when CPU allocation ratio is 1 and CPU limits are enabled", func() {
			rr = NewResourceRenderer(nil, nil, WithoutDedicatedCPU(cpu, 1, true))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("5")))
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("5")))
		})

		It("Limits to one CPU per core, when CPU allocation ratio is 10 and CPU limits are enabled", func() {
			rr = NewResourceRenderer(nil, nil, WithoutDedicatedCPU(cpu, 10, true))
			Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("500m")))
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceCPU, resource.MustParse("5")))
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
				WithMemoryOverhead(requestedVMRequirements, memOverhead),
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
				}, memOverhead))
				Expect(rr.Requests()).To(HaveKeyWithValue(kubev1.ResourceMemory, baseMemory))
				Expect(rr.Limits()).To(HaveKeyWithValue(
					kubev1.ResourceMemory,
					addResources(baseMemory, memOverhead)))
			})
		})
	})

	Context("WithAutoMemoryLimits option", func() {
		const customRatioNamespace = "custom-memory-ratio-ns"
		const customRatioValue = 3.2
		var (
			baseMemory          resource.Quantity
			userSpecifiedMemory kubev1.ResourceList
			namespaceStore      cache.Store
		)

		BeforeEach(func() {
			baseMemory = resource.MustParse("64M")
			namespaceStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

			userSpecifiedMemory = kubev1.ResourceList{kubev1.ResourceMemory: baseMemory}
			namespaceWithCustomMemoryRatio := kubev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind: "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: customRatioNamespace,
					Labels: map[string]string{
						v1.AutoMemoryLimitsRatioLabel: fmt.Sprintf("%f", customRatioValue),
					},
				},
			}
			err := namespaceStore.Add(&namespaceWithCustomMemoryRatio)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("should set limits accordingly with the ratio if vmi.limits are not set", func(namespace string, expectedRatioUsed float64) {
			rr = NewResourceRenderer(nil, userSpecifiedMemory, WithAutoMemoryLimits(namespace, namespaceStore))
			value := int64(float64(baseMemory.Value()) * expectedRatioUsed)
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceMemory, addResources(*resource.NewQuantity(value, baseMemory.Format))))
		},
			Entry("with default limit overhead ratio", "default", DefaultMemoryLimitOverheadRatio),
			Entry("with custom limit overhead ratio", customRatioNamespace, customRatioValue),
		)

		It("should not override limits if vmi.limits are set", func() {
			rr = NewResourceRenderer(userSpecifiedMemory, userSpecifiedMemory, WithAutoMemoryLimits(customRatioNamespace, namespaceStore))
			Expect(rr.Limits()).To(HaveKeyWithValue(kubev1.ResourceMemory, addResources(baseMemory)))
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
			userSpecifiedCPURequest := kubev1.ResourceList{kubev1.ResourceCPU: userCPURequest}

			DescribeTable("requires an additional 1000m CPU, and an additional CPU is added to the limits", func(defineUserSpecifiedCPULimit bool) {

				var userSpecifiedCPULimit kubev1.ResourceList

				if defineUserSpecifiedCPULimit {
					userSpecifiedCPULimit = kubev1.ResourceList{kubev1.ResourceCPU: userCPURequest}
				}
				rr = NewResourceRenderer(
					userSpecifiedCPULimit,
					userSpecifiedCPURequest,
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
			},
				Entry("only CPU requests set by the user", false),
				Entry("request and limits set by the user", true),
			)
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
			kubev1.ResourceCPU:    resource.MustParse("100m"),
			kubev1.ResourceMemory: resource.MustParse("80M"),
		}
	}

	DescribeTable("Calculate ratios from VMI", func(req, lim, expectedReq, expectedLim kubev1.ResourceList) {
		kvConfig := &v1.KubeVirtConfiguration{
			SupportContainerResources: []v1.SupportContainerResources{
				{
					Type: v1.HotplugAttachment,
					Resources: kubev1.ResourceRequirements{
						Requests: kubev1.ResourceList{},
						Limits:   kubev1.ResourceList{},
					},
				},
			},
		}
		kvConfig.SupportContainerResources[0].Resources.Requests = req
		kvConfig.SupportContainerResources[0].Resources.Limits = lim
		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kvConfig)
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{},
		}
		res := hotplugContainerResourceRequirementsForVMI(vmi, clusterConfig)
		Expect(res.Requests).To(BeEquivalentTo(expectedReq))
		Expect(res.Limits).To(BeEquivalentTo(expectedLim))
	},
		Entry("empty request/limit", kubev1.ResourceList{}, kubev1.ResourceList{}, defaultRequest(), defaultRequest()),
		Entry("empty request, set limit", kubev1.ResourceList{}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, defaultRequest(), kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}),
		Entry("set request, empty limit", kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}, kubev1.ResourceList{}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}, defaultRequest()),
		Entry("set request, set limit", kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}),
		Entry("partial set request cpu, set limit", kubev1.ResourceList{
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("100m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}),
		Entry("partial set request mem, set limit", kubev1.ResourceList{
			kubev1.ResourceCPU: resource.MustParse("25m"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("80M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}),
		Entry("set request, partial set limit cpu", kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("100m"),
			kubev1.ResourceMemory: resource.MustParse("1024M"),
		}),
		Entry("set request, partial set limit memory", kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU: resource.MustParse("140m"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("25m"),
			kubev1.ResourceMemory: resource.MustParse("32M"),
		}, kubev1.ResourceList{
			kubev1.ResourceCPU:    resource.MustParse("140m"),
			kubev1.ResourceMemory: resource.MustParse("80M"),
		}),
	)
})

var _ = Describe("GetMemoryOverhead calculation", func() {
	// VirtLauncherMonitorOverhead + VirtLauncherOverhead + VirtlogdOverhead + VirtqemudOverhead + QemuOverhead + IothreadsOverhead
	const staticOverheadString = "218Mi"
	var (
		vmi                     *v1.VirtualMachineInstance
		staticOverhead          *resource.Quantity
		baseOverhead            *resource.Quantity
		coresOverhead           *resource.Quantity
		videoRAMOverhead        *resource.Quantity
		cpuArchOverhead         *resource.Quantity
		vfioOverhead            *resource.Quantity
		downwardmetricsOverhead *resource.Quantity
		sevOverhead             *resource.Quantity
		tpmOverhead             *resource.Quantity
		passtOverhead           *resource.Quantity
	)

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "test-vmi"},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{
						Requests: kubev1.ResourceList{
							kubev1.ResourceMemory: resource.MustParse("1Gi"),
						},
						Limits: kubev1.ResourceList{},
					},
				},
			},
		}
		staticOverhead = pointer.P(resource.MustParse(staticOverheadString))
		// MemoryReq / 512bit
		baseOverhead = pointer.P(resource.MustParse("2Mi"))
		coresOverhead = pointer.P(resource.MustParse("8Mi"))
		videoRAMOverhead = pointer.P(resource.MustParse("16Mi"))
		cpuArchOverhead = pointer.P(resource.MustParse("128Mi"))
		vfioOverhead = pointer.P(resource.MustParse("1Gi"))
		downwardmetricsOverhead = pointer.P(resource.MustParse("1Mi"))
		sevOverhead = pointer.P(resource.MustParse("256Mi"))
		tpmOverhead = pointer.P(resource.MustParse("53Mi"))
		passtOverhead = pointer.P(resource.MustParse("800Mi"))
	})

	When("the vmi is not requesting any specific device or cpu or whatever", func() {
		It("should return base overhead+static+8Mi", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			// 8Mi*1core(default)
			expected.Add(*coresOverhead)
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests the specific cpu", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:   2,
				Threads: 2,
				Sockets: 2,
			}
		})

		It("should adjust overhead based on the cores/threads/sockets", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			// (2cores* 2threads *2sockets)
			value := coresOverhead.Value() * 8
			expected.Add(*resource.NewQuantity(value, coresOverhead.Format))
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests cpu resource", func() {
		DescribeTable("should adjust overhead", func(requests, limits string, coresMultiplier int) {
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceCPU] = resource.MustParse(requests)
			if limits != "" {
				vmi.Spec.Domain.Resources.Limits[kubev1.ResourceCPU] = resource.MustParse(limits)
			}

			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			value := coresOverhead.Value() * int64(coresMultiplier)
			expected.Add(*resource.NewQuantity(value, coresOverhead.Format))
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("based on the limits if both requests and limits are provided", "3", "5", 5),
			Entry("based on the requests if only requests are provided", "3", "", 3),
		)

	})

	When("the vmi does not require auto attach graphics device", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.P(false)
		})

		It("should not add videoRAMOverhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*coresOverhead)
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the cpu arch is arm64", func() {
		It("should add arm64 overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*cpuArchOverhead)
			overhead := GetMemoryOverhead(vmi, "arm64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests a VFIO device", func() {
		DescribeTable("should add vfio overhead", func(devices v1.Devices) {
			vmi.Spec.Domain.Devices = devices
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*vfioOverhead)
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with hostDEV", v1.Devices{HostDevices: []v1.HostDevice{{Name: "test"}}}),
			Entry("with GPU", v1.Devices{GPUs: []v1.GPU{{Name: "test"}}}),
			Entry("with SRIOV", v1.Devices{Interfaces: []v1.Interface{{Name: "test", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}}}),
		)
	})

	When("the vmi has a downward metrics volume", func() {
		BeforeEach(func() {
			vmi.Spec.Volumes = []v1.Volume{{VolumeSource: v1.VolumeSource{DownwardMetrics: &v1.DownwardMetricsVolumeSource{}}}}
		})
		It("should add downwardMetrics overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*downwardmetricsOverhead)
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi has probes fields", func() {
		DescribeTable("should add probes overhead", func(livenessProbe, readinessProbe *v1.Probe, probeOverhead resource.Quantity) {
			vmi.Spec.LivenessProbe = livenessProbe
			vmi.Spec.ReadinessProbe = readinessProbe
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(probeOverhead)

			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with livenessProbe only", &v1.Probe{Handler: v1.Handler{Exec: &kubev1.ExecAction{}}}, nil, resource.MustParse("110Mi")),
			Entry("with readinessProbe only", nil, &v1.Probe{Handler: v1.Handler{Exec: &kubev1.ExecAction{}}}, resource.MustParse("110Mi")),
			Entry("with both readinessProbe adn livenessProbe", &v1.Probe{Handler: v1.Handler{Exec: &kubev1.ExecAction{}}}, &v1.Probe{Handler: v1.Handler{Exec: &kubev1.ExecAction{}}}, resource.MustParse("120Mi")),
		)
	})

	When("the vmi requests AMD SEV", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{},
			}
		})

		It("should add SEV overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*sevOverhead)
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests TPM device", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.Devices = v1.Devices{
				TPM: &v1.TPMDevice{},
			}
		})

		It("should add SEV overhead", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(*tpmOverhead)
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the vmi requests interfaces with Passt binding", func() {
		BeforeEach(func() {
			vmi.Spec.Domain.Devices = v1.Devices{
				Interfaces: []v1.Interface{
					{Name: "passt1", InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}}},
					{Name: "passt2", InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}}},
					{Name: "passt3", InterfaceBindingMethod: v1.InterfaceBindingMethod{Passt: &v1.InterfacePasst{}}},
					{Name: "nonpasst", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				},
			}
		})

		It("should add passt overhead for each interface", func() {
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			value := passtOverhead.Value() * 3
			expected.Add(*resource.NewQuantity(value, passtOverhead.Format))
			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		})
	})

	When("the additionalOverheadRatio is provided", func() {
		DescribeTable("should adjust the overhead using the given ratio", func(additionalOverheadRatio string, expectParseError bool) {
			base := resource.NewScaledQuantity(0, resource.Kilo)
			base.Add(*baseOverhead)
			base.Add(*staticOverhead)
			base.Add(*videoRAMOverhead)
			base.Add(*coresOverhead)
			var expected resource.Quantity
			if expectParseError {
				expected = *base
			} else {
				ratio, _ := strconv.ParseFloat(additionalOverheadRatio, 64)
				expected = multiplyMemory(*base, ratio)
			}

			overhead := GetMemoryOverhead(vmi, "amd64", pointer.P(additionalOverheadRatio))
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with the given value if the given value is a float", "3.2", false),
			Entry("with no value if the given value is not a float", "no_float", true),
		)
	})

	When("the vmi is requesting dedicated CPU or wants to have QOSGuaranteed", func() {
		DescribeTable("should add 100Mi of overhead", func(requestDedicatedCPU, wantsQOSGuaranteed bool) {
			vmi.Spec.Domain.CPU = &v1.CPU{Cores: 1}
			if requestDedicatedCPU {
				vmi.Spec.Domain.CPU.DedicatedCPUPlacement = true
			}
			if wantsQOSGuaranteed {
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1Gi")
				vmi.Spec.Domain.Resources.Limits[kubev1.ResourceMemory] = resource.MustParse("1Gi")
				vmi.Spec.Domain.Resources.Requests[kubev1.ResourceCPU] = resource.MustParse("4")
				vmi.Spec.Domain.Resources.Limits[kubev1.ResourceCPU] = resource.MustParse("4")
			}
			expected := resource.NewScaledQuantity(0, resource.Kilo)
			expected.Add(*baseOverhead)
			expected.Add(*staticOverhead)
			expected.Add(*videoRAMOverhead)
			expected.Add(*coresOverhead)
			expected.Add(resource.MustParse("100Mi"))

			overhead := GetMemoryOverhead(vmi, "amd64", nil)
			Expect(overhead.Value()).To(BeEquivalentTo(expected.Value()))
		},
			Entry("with DedicatedCPU", true, false),
			Entry("when wants QOSGuaranteed", false, true),
		)
	})

})

func addResources(firstQuantity resource.Quantity, resources ...resource.Quantity) resource.Quantity {
	for _, resourceQuantity := range resources {
		firstQuantity.Add(resourceQuantity)
	}
	return firstQuantity
}
