package v1

import (
	"reflect"

	"github.com/google/gofuzz"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Generated deepcopy functions", func() {

	var structs []interface{}
	BeforeEach(func() {

		structs = []interface{}{
			&CloudInitNoCloudSource{},
			&DomainSpec{},
			&ResourceRequirements{},
			&Firmware{},
			&Devices{},
			&Disk{},
			&DiskDevice{},
			&DiskTarget{},
			&LunTarget{},
			&FloppyTarget{},
			&CDRomTarget{},
			&Volume{},
			&VolumeSource{},
			&RegistryDiskSource{},
			&Interface{},
			&InterfaceDevice{},
			&E1000Interface{},
			&VirtIOInterface{},
			&RTL8139Interface{},
			&InterfaceAttrs{},
			&InterfaceSource{},
			&PodNetworkSource{},
			&ClockOffset{},
			&ClockOffsetUTC{},
			&Clock{},
			&Timer{},
			&RTCTimerAttrs{},
			&TimerAttrs{},
			&KVMTimerAttrs{},
			&HypervTimerAttrs{},
			&Features{},
			&FeatureState{},
			&FeatureAPIC{},
			&FeatureSpinlocks{},
			&FeatureVendorID{},
			&FeatureHyperv{},
			&Watchdog{},
			&WatchdogDevice{},
			&I6300ESBWatchdog{},
			&VirtualMachine{},
			&VirtualMachineList{},
			&VirtualMachineSpec{},
			&Affinity{},
			&VirtualMachineStatus{},
			&VirtualMachineGraphics{},
			&VirtualMachineCondition{},
			&Spice{},
			&SpiceInfo{},
			&Migration{},
			&MigrationSpec{},
			&VMSelector{},
			&Migration{},
			&MigrationStatus{},
			&MigrationList{},
			&MigrationHostInfo{},
			&VirtualMachineReplicaSet{},
			&VirtualMachineReplicaSetList{},
			&VMReplicaSetSpec{},
			&VMReplicaSetStatus{},
			&VMReplicaSetCondition{},
			&VMTemplateSpec{},
		}
	})

	It("should work for fuzzed structs", func() {
		for _, s := range structs {
			fuzz.New().NilChance(0).Fuzz(s)
			Expect(reflect.ValueOf(s).MethodByName("DeepCopy").Call(nil)[0].Interface()).To(Equal(s))
			if reflect.ValueOf(s).MethodByName("DeepCopyObject").IsValid() {
				Expect(reflect.ValueOf(s).MethodByName("DeepCopyObject").Call(nil)[0].Interface()).To(Equal(s))
			}
			new := reflect.New(reflect.TypeOf(s).Elem())
			reflect.ValueOf(s).MethodByName("DeepCopyInto").Call([]reflect.Value{new})
			Expect(new.Interface()).To(Equal(s))
		}

	})
})
