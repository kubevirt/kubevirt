package api

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/randfill"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Generated deepcopy functions", func() {

	var structs []interface{}
	BeforeEach(func() {

		structs = []interface{}{
			&v1.CloudInitNoCloudSource{},
			&v1.DomainSpec{},
			&v1.ResourceRequirements{},
			&v1.Firmware{},
			&v1.Devices{},
			&v1.Disk{},
			&v1.DiskDevice{},
			&v1.DiskTarget{},
			&v1.LunTarget{},
			&v1.CDRomTarget{},
			&v1.Volume{},
			&v1.VolumeSource{},
			&v1.ContainerDiskSource{},
			&v1.ClockOffset{},
			&v1.ClockOffsetUTC{},
			&v1.Clock{},
			&v1.Timer{},
			&v1.RTCTimer{},
			&v1.HPETTimer{},
			&v1.PITTimer{},
			&v1.KVMTimer{},
			&v1.HypervTimer{},
			&v1.Features{},
			&v1.FeatureState{},
			&v1.FeatureAPIC{},
			&v1.FeatureSpinlocks{},
			&v1.FeatureVendorID{},
			&v1.FeatureHyperv{},
			&v1.CPU{},
			&v1.Watchdog{},
			&v1.WatchdogDevice{},
			&v1.I6300ESBWatchdog{},
			&v1.VirtualMachineInstance{},
			&v1.VirtualMachineInstanceList{},
			&v1.VirtualMachineInstanceSpec{},
			&v1.VirtualMachineInstanceStatus{},
			&v1.VirtualMachineInstanceCondition{},
			&v1.VMISelector{},
			&v1.VirtualMachineInstanceReplicaSet{},
			&v1.VirtualMachineInstanceReplicaSetList{},
			&v1.VirtualMachineInstanceReplicaSetSpec{},
			&v1.VirtualMachineInstanceReplicaSetStatus{},
			&v1.VirtualMachineInstanceReplicaSetCondition{},
			&v1.VirtualMachineInstanceTemplateSpec{},
			&v1.VirtualMachine{},
			&v1.VirtualMachineList{},
			&v1.VirtualMachineSpec{},
			&v1.VirtualMachineCondition{},
			&v1.VirtualMachineStatus{},
			&v1.VirtualMachineInstancePreset{},
			&v1.VirtualMachineInstancePresetList{},
			&v1.VirtualMachineInstancePresetSpec{},
			&v1.Probe{},
			&v1.Handler{},
			&v1.Hugepages{},
			&v1.Interface{},
			&v1.Memory{},
			&v1.Machine{},
			&v1.InterfaceBridge{},
			&v1.DeprecatedInterfaceSlirp{},
		}
	})

	DescribeTable("should work for fuzzed structs with a probability for nils of", func(nilProbability float64) {
		for _, s := range structs {
			randfill.New().NilChance(nilProbability).Fill(s)
			Expect(reflect.ValueOf(s).MethodByName("DeepCopy").Call(nil)[0].Interface()).To(Equal(s))
			if reflect.ValueOf(s).MethodByName("DeepCopyObject").IsValid() {
				Expect(reflect.ValueOf(s).MethodByName("DeepCopyObject").Call(nil)[0].Interface()).To(Equal(s))
			}
			new := reflect.New(reflect.TypeOf(s).Elem())
			reflect.ValueOf(s).MethodByName("DeepCopyInto").Call([]reflect.Value{new})
			Expect(new.Interface()).To(Equal(s))
		}
	},
		Entry("0%", float64(0)),
		Entry("10%", float64(0.1)),
		Entry("50%", float64(0.5)),
		Entry("70%", float64(0.7)),
		Entry("100%", float64(1)),
	)
})
