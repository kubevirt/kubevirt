package virthandler

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Set pthread scheduling type and priority", func() {
	Context("when parsing the thread command for CPU ID", func() {
		DescribeTable("extracts the CPU ID", func(comm []byte, cpuID string, parseOK bool) {
			v, ok := isVCPU(comm)
			Expect(ok).To(Equal(parseOK))
			Expect(v).To(Equal(cpuID))
		},
			Entry("Correctly extracts the CPU ID from a single digit vcpu pthread", []byte("CPU 3/KVM\n"), "3", true),
			Entry("Correctly extracts the CPU ID from a double digit vcpu pthread", []byte("CPU 10/KVM\n"), "10", true),
			Entry("Fails to parse the comm value", []byte("vCPU 1/KVM\n"), "", false),
			Entry("Fails to parse a negative vCPU value", []byte("CPU -1/KVM\n"), "", false),
			Entry("Fails to parse a vCPU value that does not end with a carrier return", []byte("CPU 1/KVM"), "", false))
	})
})

var _ = Describe("Determines if a vcpu is set for realtime", func() {
	Context("when checking for values", func() {
		DescribeTable("extracts the vcpu configuration", func(parsedMask map[string]maskType, vcpuID string, status maskType) {
			res := isRealtimeVCPU(parsedMask, vcpuID)
			Expect(res).To(BeEquivalentTo(status))
		},
			Entry("correctly returns a match as enabled", map[string]maskType{"0": enabled, "1": disabled}, "0", enabled),
			Entry("correctly returns a match as disabled", map[string]maskType{"0": disabled, "1": disabled}, "1", disabled),
			Entry("correctly returns a match when the map is empty", map[string]maskType{}, "1", enabled),
			Entry("fails to find the vcpu in the map", map[string]maskType{"0": disabled, "1": disabled}, "2", disabled),
		)
	})
})

var _ = Describe("Parse the vCPU mask", func() {
	Context("when checking for single digit, range and negative digit", func() {
		DescribeTable("extracts the vCPU range", func(mask string, parsed map[string]maskType, err error) {
			m, e := parseCPUMask(mask)
			if err != nil {
				Expect(e).To(BeEquivalentTo(err))
			} else {
				Expect(e).NotTo(HaveOccurred())
			}
			Expect(m).To(BeEquivalentTo(parsed))
		},

			// Empty mask
			Entry("Empty mask", "", nil, fmt.Errorf("emtpy mask ``")),
			Entry("Empty mask with spaces", "  ", nil, fmt.Errorf("emtpy mask `  `")),
			Entry("Invalid expression", "a-b,33_", nil, fmt.Errorf("invalid mask value 'a-b' in 'a-b,33_'")),
			Entry("Invalid mask value", "3,3a", nil, fmt.Errorf("invalid mask value '3a' in '3,3a'")),
			// With range
			Entry("Correctly extracts a single range", "0-2", map[string]maskType{"0": enabled, "1": enabled, "2": enabled}, nil),
			Entry("Correctly extracts a multiple non overlapping ranges", "0-2,5-6", map[string]maskType{"0": enabled, "1": enabled, "2": enabled, "5": enabled, "6": enabled}, nil),
			Entry("Correctly extracts a multiple overlapping ranges", "0-2,2-3", map[string]maskType{"0": enabled, "1": enabled, "2": enabled, "3": enabled}, nil),
			Entry("Correctly extracts a single range with double digit vcpu IDs", "10-12", map[string]maskType{"10": enabled, "11": enabled, "12": enabled}, nil),
			Entry("Correctly extracts a multiple range with double digit vcpu IDs", "10-12,20,21", map[string]maskType{"10": enabled, "11": enabled, "12": enabled, "20": enabled, "21": enabled}, nil),
			Entry("Correctly extracts a multiple range with double digit vcpu IDs spaced in between", "10-12 ,  20,21", map[string]maskType{"10": enabled, "11": enabled, "12": enabled, "20": enabled, "21": enabled}, nil),
			Entry("Correctly extracts a single range prefixed and postfixed with spaces", "  0-2  ", map[string]maskType{"0": enabled, "1": enabled, "2": enabled}, nil),
			Entry("Correctly extracts a single range with a range of 1 vcpu only", "0-0", map[string]maskType{"0": enabled}, nil),
			Entry("Fails to parse as the start range is smaller than the ending vcpu index id", "1-0", nil, fmt.Errorf("invalid mask range `1-0`")),
			Entry("Fails to match", "1-", nil, fmt.Errorf("invalid mask value '1-' in '1-'")),
			// Single vcpus
			Entry("Correctly extracts a single vCPU index id", "0", map[string]maskType{"0": enabled}, nil),
			Entry("Correctly extracts a single vCPU index id and ignores a negative index", "0,-1", nil, fmt.Errorf("invalid mask value '-1' in '0,-1'")),
			Entry("Correctly extracts a multiple single vCPU index ids", "0,1,2", map[string]maskType{"0": enabled, "1": enabled, "2": enabled}, nil),
			Entry("Correctly extracts a single double digit vcpu index id", "10", map[string]maskType{"10": enabled}, nil),
			Entry("Fails to parse a negative vcpu ID", "-1", nil, fmt.Errorf("invalid mask value '-1' in '-1'")),
			// Negate vcpus
			Entry("Correctly extracts a single negate vCPU", "^3", map[string]maskType{"3": disabled}, nil),
			Entry("Fails to extract a negate vCPU with a negative index", "^3,^-1", nil, fmt.Errorf("invalid mask value '^-1' in '^3,^-1'")),
			Entry("Correctly extracts a multiple negate vCPUs", "^0,^1", map[string]maskType{"0": disabled, "1": disabled}, nil),
			Entry("Correctly extracts a multiple overlapping negate vCPUs", "^0,^1,^0", map[string]maskType{"0": disabled, "1": disabled}, nil),
			Entry("Correctly extracts a double digit negate vCPU", "^13", map[string]maskType{"13": disabled}, nil),
			Entry("Correctly extracts a multiple double digit negate vCPU ids", "^13,^15", map[string]maskType{"13": disabled, "15": disabled}, nil),
			Entry("Fails to parse a negative negate vcpu ID", "^-1", nil, fmt.Errorf("invalid mask value '^-1' in '^-1'")),
			// Combination of range and single Ids
			Entry("Correctly extracts a single range and individual ID without overlaping index but in sequence", "0-2,3", map[string]maskType{"0": enabled, "1": enabled, "2": enabled, "3": enabled}, nil),
			Entry("Correctly extracts a single range and individual ID with overlaping index", "0-2,1", map[string]maskType{"0": enabled, "1": enabled, "2": enabled}, nil),
			Entry("Correctly extracts a single range and individual ID without overlaping but out of sequence", "5,1-2,6", map[string]maskType{"1": enabled, "2": enabled, "5": enabled, "6": enabled}, nil),
			Entry("Correctly extracts a single range and individual ID without overlaping but out of sequence with spaces", " 6 , 1-2 , 5 ", map[string]maskType{"1": enabled, "2": enabled, "5": enabled, "6": enabled}, nil),
			// Combination of single Ids and negate index
			Entry("Correctly extracts a single vCPU index with a negated id", "0,^1", map[string]maskType{"0": enabled, "1": disabled}, nil),
			Entry("Correctly extracts a multiple vCPU index and negated ids", "0,1,2,^3,^4", map[string]maskType{"0": enabled, "1": enabled, "2": enabled, "3": disabled, "4": disabled}, nil),
			Entry("Correctly extracts a multiple vCPU index with an overlapping negated id sandwitched in between", "0,^0,1", map[string]maskType{"0": disabled, "1": enabled}, nil),
			Entry("Correctly extracts a single vCPU index with overlapping negated id with negated as first entry", "^0,0", map[string]maskType{"0": disabled}, nil),
			// Combination of range and negate index
			Entry("Correctly extracts a single range and a negate id outside range", "0-2,^3", map[string]maskType{"0": enabled, "1": enabled, "2": enabled, "3": disabled}, nil),
			Entry("Correctly extracts a single range and a negate id within range", "0-2,^1", map[string]maskType{"0": enabled, "1": disabled, "2": enabled}, nil),
			Entry("Correctly extracts a multiple range and two negate ids within range", "^1,0-2, 3-6,^4 ", map[string]maskType{"0": enabled, "1": disabled, "2": enabled, "3": enabled, "4": disabled, "5": enabled, "6": enabled}, nil),
			// Combination of all 3 expressions
			Entry("Correctly extracts a single range, single index and negate ids without overlap", "^10,0-2,3", map[string]maskType{"0": enabled, "1": enabled, "2": enabled, "3": enabled, "10": disabled}, nil),
			Entry("Correctly extracts a multiple range, multiple index and multiple negate ids without overlap", "^1,0-2,^5,3,10,6-6", map[string]maskType{"0": enabled, "1": disabled, "2": enabled, "3": enabled, "5": disabled, "6": enabled, "10": enabled}, nil),
			Entry("Correctly extracts a multiple range, multiple index and negate ids with overlap", "^1,0-3,^3,4,5,6-8,^7", map[string]maskType{"0": enabled, "1": disabled, "2": enabled, "3": disabled, "4": enabled, "5": enabled, "6": enabled, "7": disabled, "8": enabled}, nil),
		)
	})
})
