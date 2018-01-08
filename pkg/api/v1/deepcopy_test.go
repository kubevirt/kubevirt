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
			&ClockOffset{},
			&ClockOffsetUTC{},
			&Clock{},
			&Timer{},
			&RTCTimer{},
			&HPETTimer{},
			&PITTimer{},
			&KVMTimer{},
			&HypervTimer{},
			&Features{},
			&FeatureState{},
			&FeatureAPIC{},
			&FeatureSpinlocks{},
			&FeatureVendorID{},
			&FeatureHyperv{},
			&CPU{},
			&Watchdog{},
			&WatchdogDevice{},
			&I6300ESBWatchdog{},
			&VirtualMachine{},
			&VirtualMachineList{},
			&VirtualMachineSpec{},
			&Affinity{},
			&VirtualMachineStatus{},
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
