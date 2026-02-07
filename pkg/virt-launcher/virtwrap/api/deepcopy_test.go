package api

import (
	"reflect"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/randfill"
)

var _ = ginkgo.Describe("Generated deepcopy functions", func() {

	var structs []interface{}
	ginkgo.BeforeEach(func() {

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

	ginkgo.It("should work for fuzzed structs", func() {
		for _, s := range structs {
			randfill.New().NilChance(0).Fill(s)
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
