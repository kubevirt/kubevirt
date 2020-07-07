package api

import (
	"reflect"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Generated deepcopy functions", func() {

	var structs []interface{}
	BeforeEach(func() {

		structs = []interface{}{
			&Domain{},
			&DomainStatus{},
			&DomainList{},
			&DomainSpec{},
			&Features{},
			&FeatureHyperv{},
			&FeatureSpinlocks{},
			&FeatureVendorID{},
			&FeatureEnabled{},
			&FeatureState{},
			&Metadata{},
			&KubeVirtMetadata{},
			&GracePeriodMetadata{},
			&Commandline{},
			&Env{},
			&Resource{},
			&Memory{},
			&Devices{},
			&Disk{},
			&DiskAuth{},
			&DiskSecret{},
			&ReadOnly{},
			&DiskSource{},
			&DiskTarget{},
			&DiskDriver{},
			&DiskSourceHost{},
			&Serial{},
			&SerialTarget{},
			&Console{},
			&ConsoleTarget{},
			&Interface{},
			&LinkState{},
			&BandWidth{},
			&BootOrder{},
			&MAC{},
			&FilterRef{},
			&InterfaceSource{},
			&Model{},
			&InterfaceTarget{},
			&Alias{},
			&OS{},
			&OSType{},
			&SMBios{},
			&NVRam{},
			&Boot{},
			&BootMenu{},
			&BIOS{},
			&Loader{},
			&SysInfo{},
			&Entry{},
			&Clock{},
			&Timer{},
			&Channel{},
			&ChannelTarget{},
			&ChannelSource{},
			&Video{},
			&VideoModel{},
			&Graphics{},
			&GraphicsListen{},
			&Address{},
			&MemBalloon{},
			&Rng{},
			&RngBackend{},
			&RngRate{},
			&Watchdog{},
			&SecretUsage{},
			&SecretSpec{},
			&CPU{},
			&CPUTopology{},
			&VCPU{},
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
