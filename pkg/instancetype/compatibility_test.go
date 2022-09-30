package instancetype

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1alpha1"
)

var _ = Describe("instancetype compatibility", func() {
	Context("reading old ControllerRevision", func() {
		It("should decode v1alpha1 instancetype from ControllerRevision", func() {
			instancetypeSpec := v1alpha1.VirtualMachineInstancetypeSpec{
				CPU: v1alpha1.CPUInstancetype{
					Guest: 4,
				},
			}

			specBytes, err := json.Marshal(&instancetypeSpec)
			Expect(err).ToNot(HaveOccurred())

			revision := v1alpha1.VirtualMachineInstancetypeSpecRevision{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Spec:       specBytes,
			}

			revisionBytes, err := json.Marshal(revision)
			Expect(err).ToNot(HaveOccurred())

			decoded, err := decodeOldInstancetypeRevisionObject(revisionBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).ToNot(BeNil())
			Expect(decoded.Spec.CPU).To(Equal(instancetypeSpec.CPU))
		})

		It("should decode v1alpha1 preference from ControllerRevision", func() {
			preferredTopology := v1alpha1.PreferCores
			preferenceSpec := v1alpha1.VirtualMachinePreferenceSpec{
				CPU: &v1alpha1.CPUPreferences{
					PreferredCPUTopology: preferredTopology,
				},
			}

			specBytes, err := json.Marshal(&preferenceSpec)
			Expect(err).ToNot(HaveOccurred())

			revision := v1alpha1.VirtualMachinePreferenceSpecRevision{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Spec:       specBytes,
			}

			revisionBytes, err := json.Marshal(revision)
			Expect(err).ToNot(HaveOccurred())

			decoded, err := decodeOldPreferenceRevisionObject(revisionBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).ToNot(BeNil())
			Expect(decoded.Spec).To(Equal(preferenceSpec))
		})
	})

	Context("instancetype conversion", func() {
		It("should convert instancetype from v1alpha1 to v1alpha2", func() {
			instancetypeOld := &v1alpha1.VirtualMachineInstancetype{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "VirtualMachineInstancetype",
				},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       getOldInstanceTypeSpec(),
			}

			_, err := convertInstancetype(instancetypeOld)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should convert preference from v1alpha1 to v1alpha2", func() {
			preferenceOld := &v1alpha1.VirtualMachinePreference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "VirtualMachinePreference",
				},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       getOldPreferenceSpec(),
			}

			_, err := convertPreference(preferenceOld)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getOldInstanceTypeSpec() v1alpha1.VirtualMachineInstancetypeSpec {
	ioThreadPolicy := v1.IOThreadsPolicyAuto
	return v1alpha1.VirtualMachineInstancetypeSpec{
		CPU: v1alpha1.CPUInstancetype{
			Guest:                 1,
			Model:                 "test-model",
			DedicatedCPUPlacement: true,
			NUMA: &v1.NUMA{
				GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
			},
			IsolateEmulatorThread: true,
			Realtime: &v1.Realtime{
				Mask: "0-3",
			},
		},
		Memory: v1alpha1.MemoryInstancetype{
			Guest:     resource.MustParse("10G"),
			Hugepages: &v1.Hugepages{PageSize: "1Gi"},
		},
		GPUs: []v1.GPU{{
			Name:       "test-gpu",
			DeviceName: "test-device",
			VirtualGPUOptions: &v1.VGPUOptions{
				Display: &v1.VGPUDisplayOptions{
					Enabled: pointer.Bool(true),
					RamFB: &v1.FeatureState{
						Enabled: pointer.Bool(true),
					},
				},
			},
			Tag: "tag",
		}},
		HostDevices: []v1.HostDevice{{
			Name:       "name",
			DeviceName: "dev-name",
			Tag:        "tag",
		}},
		IOThreadsPolicy: &ioThreadPolicy,
		LaunchSecurity: &v1.LaunchSecurity{
			SEV: &v1.SEV{},
		},
	}
}

func getOldPreferenceSpec() v1alpha1.VirtualMachinePreferenceSpec {
	var timezone v1.ClockOffsetTimezone = "timezone"
	var retries uint32 = 1
	return v1alpha1.VirtualMachinePreferenceSpec{
		Clock: &v1alpha1.ClockPreferences{
			PreferredClockOffset: &v1.ClockOffset{
				UTC: &v1.ClockOffsetUTC{
					OffsetSeconds: pointer.Int(1),
				},
				Timezone: &timezone,
			},
			PreferredTimer: &v1.Timer{
				HPET: &v1.HPETTimer{
					TickPolicy: v1.HPETTickPolicyDelay,
					Enabled:    pointer.Bool(true),
				},
				KVM: &v1.KVMTimer{
					Enabled: pointer.Bool(true),
				},
				PIT: &v1.PITTimer{
					TickPolicy: v1.PITTickPolicyDelay,
					Enabled:    pointer.Bool(true),
				},
				RTC: &v1.RTCTimer{
					TickPolicy: v1.RTCTickPolicyDelay,
					Enabled:    pointer.Bool(true),
					Track:      v1.TrackGuest,
				},
				Hyperv: &v1.HypervTimer{
					Enabled: pointer.Bool(true),
				},
			},
		},
		CPU: &v1alpha1.CPUPreferences{
			PreferredCPUTopology: v1alpha1.PreferCores,
		},
		Devices: &v1alpha1.DevicePreferences{
			PreferredAutoattachGraphicsDevice: pointer.Bool(true),
			PreferredAutoattachMemBalloon:     pointer.Bool(true),
			PreferredAutoattachPodInterface:   pointer.Bool(true),
			PreferredAutoattachSerialConsole:  pointer.Bool(true),
			PreferredAutoattachInputDevice:    pointer.Bool(true),
			PreferredDisableHotplug:           pointer.Bool(true),
			PreferredVirtualGPUOptions: &v1.VGPUOptions{
				Display: &v1.VGPUDisplayOptions{
					Enabled: pointer.Bool(true),
					RamFB:   &v1.FeatureState{Enabled: pointer.Bool(true)},
				},
			},
			PreferredSoundModel:            "sound-model",
			PreferredUseVirtioTransitional: pointer.Bool(true),
			PreferredInputBus:              v1.InputBusVirtio,
			PreferredInputType:             v1.InputTypeTablet,
			PreferredDiskBus:               v1.DiskBusVirtio,
			PreferredLunBus:                v1.DiskBusVirtio,
			PreferredCdromBus:              v1.DiskBusVirtio,
			PreferredDiskDedicatedIoThread: pointer.Bool(true),
			PreferredDiskCache:             v1.CacheNone,
			PreferredDiskIO:                v1.IOThreads,
			PreferredDiskBlockSize: &v1.BlockSize{
				Custom: &v1.CustomBlockSize{
					Logical:  1,
					Physical: 1,
				},
				MatchVolume: &v1.FeatureState{Enabled: pointer.Bool(true)},
			},
			PreferredInterfaceModel:             "interface",
			PreferredRng:                        &v1.Rng{},
			PreferredBlockMultiQueue:            pointer.Bool(true),
			PreferredNetworkInterfaceMultiQueue: pointer.Bool(true),
			PreferredTPM:                        &v1.TPMDevice{},
		},
		Features: &v1alpha1.FeaturePreferences{
			PreferredAcpi: &v1.FeatureState{Enabled: pointer.Bool(true)},
			PreferredApic: &v1.FeatureAPIC{
				Enabled:        pointer.Bool(true),
				EndOfInterrupt: true,
			},
			PreferredHyperv: &v1.FeatureHyperv{
				Relaxed: &v1.FeatureState{Enabled: pointer.Bool(true)},
				VAPIC:   &v1.FeatureState{Enabled: pointer.Bool(true)},
				Spinlocks: &v1.FeatureSpinlocks{
					Enabled: pointer.Bool(true),
					Retries: &retries,
				},
				VPIndex: &v1.FeatureState{Enabled: pointer.Bool(true)},
				Runtime: &v1.FeatureState{Enabled: pointer.Bool(true)},
				SyNIC:   &v1.FeatureState{Enabled: pointer.Bool(true)},
				SyNICTimer: &v1.SyNICTimer{
					Enabled: pointer.Bool(true),
					Direct:  &v1.FeatureState{Enabled: pointer.Bool(true)},
				},
				Reset: &v1.FeatureState{Enabled: pointer.Bool(true)},
				VendorID: &v1.FeatureVendorID{
					Enabled:  pointer.Bool(true),
					VendorID: "vendor-id",
				},
				Frequencies:     &v1.FeatureState{Enabled: pointer.Bool(true)},
				Reenlightenment: &v1.FeatureState{Enabled: pointer.Bool(true)},
				TLBFlush:        &v1.FeatureState{Enabled: pointer.Bool(true)},
				IPI:             &v1.FeatureState{Enabled: pointer.Bool(true)},
				EVMCS:           &v1.FeatureState{Enabled: pointer.Bool(true)},
			},
			PreferredKvm:        &v1.FeatureKVM{Hidden: true},
			PreferredPvspinlock: &v1.FeatureState{Enabled: pointer.Bool(true)},
			PreferredSmm:        &v1.FeatureState{Enabled: pointer.Bool(true)},
		},
		Firmware: &v1alpha1.FirmwarePreferences{
			PreferredUseBios:       pointer.Bool(true),
			PreferredUseBiosSerial: pointer.Bool(true),
			PreferredUseEfi:        pointer.Bool(true),
			PreferredUseSecureBoot: pointer.Bool(true),
		},
		Machine: &v1alpha1.MachinePreferences{
			PreferredMachineType: "machine-type",
		},
	}
}
