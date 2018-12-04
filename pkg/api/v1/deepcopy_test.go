package v1

import (
	"reflect"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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
			&ContainerDiskSource{},
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
			&VirtualMachineInstance{},
			&VirtualMachineInstanceList{},
			&VirtualMachineInstanceSpec{},
			&VirtualMachineInstanceStatus{},
			&VirtualMachineInstanceCondition{},
			&VMISelector{},
			&VirtualMachineInstanceReplicaSet{},
			&VirtualMachineInstanceReplicaSetList{},
			&VirtualMachineInstanceReplicaSetSpec{},
			&VirtualMachineInstanceReplicaSetStatus{},
			&VirtualMachineInstanceReplicaSetCondition{},
			&VirtualMachineInstanceTemplateSpec{},
			&VirtualMachine{},
			&VirtualMachineList{},
			&VirtualMachineSpec{},
			&VirtualMachineCondition{},
			&VirtualMachineStatus{},
			&VirtualMachineInstancePreset{},
			&VirtualMachineInstancePresetList{},
			&VirtualMachineInstancePresetSpec{},
			&Probe{},
			&Handler{},
			&Hugepages{},
			&Interface{},
			&Memory{},
			&Machine{},
			&InterfaceBridge{},
			&InterfaceSlirp{},
		}
	})

	table.DescribeTable("should work for fuzzed structs with a probability for nils of", func(nilProbability float64) {
		for _, s := range structs {
			fuzz.New().NilChance(nilProbability).Fuzz(s)
			Expect(reflect.ValueOf(s).MethodByName("DeepCopy").Call(nil)[0].Interface()).To(Equal(s))
			if reflect.ValueOf(s).MethodByName("DeepCopyObject").IsValid() {
				Expect(reflect.ValueOf(s).MethodByName("DeepCopyObject").Call(nil)[0].Interface()).To(Equal(s))
			}
			new := reflect.New(reflect.TypeOf(s).Elem())
			reflect.ValueOf(s).MethodByName("DeepCopyInto").Call([]reflect.Value{new})
			Expect(new.Interface()).To(Equal(s))
		}
	},
		table.Entry("0%", float64(0)),
		table.Entry("10%", float64(0.1)),
		table.Entry("50%", float64(0.5)),
		table.Entry("70%", float64(0.7)),
		table.Entry("100%", float64(1)),
	)
})
