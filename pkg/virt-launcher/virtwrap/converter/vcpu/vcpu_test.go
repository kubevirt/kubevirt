/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package vcpu

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type factoryFunc func(requestedToplogy *api.CPUTopology, nodeTopology *v1.Topology, cpuSet []int) VCPUPool

var _ = Describe("VCPU pinning", func() {

	BeforeEach(func() {
		seed := time.Now().UnixNano()
		rand.NewSource(seed)
		log.DefaultLogger().Infof("using seed %v for cpu pinning tests", seed)
	})

	for _, factory := range []factoryFunc{NewRelaxedCPUPool, NewStrictCPUPool} {
		generatePositiveCPUPinningTests(
			defaultArgs().HostThreadsPerCore(1).CellCount(1).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(2).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(3).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(4).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(1).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(2).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(3).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(1).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(1).CellCount(2).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(1).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(2).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(3).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(4).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory).CPUTuneThreads(1, 7, 4, 10, 0, 6, 5, 11, 2, 8, 3, 9),
			defaultArgs().HostThreadsPerCore(2).CellCount(1).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(2).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(3).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(1).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(2).CellCount(2).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(3).CellCount(1).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(3).CellCount(2).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(3).CellCount(3).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory).CPUTuneThreads(1, 7, 0, 10, 5, 11, 6, 2, 8, 3, 9, 4),
			defaultArgs().HostThreadsPerCore(3).CellCount(4).VCPUCores(12).VCPUThreadsPerCore(1).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(3).CellCount(1).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory).CPUTuneThreads(1, 7, 6, 2, 3, 9, 10, 5, 0, 8, 4, 11),
			defaultArgs().HostThreadsPerCore(3).CellCount(2).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(factory).CPUTuneThreads(1, 7, 6, 2, 3, 9, 10, 5, 0, 8, 4, 11),
			defaultArgs().HostThreadsPerCore(3).CellCount(1).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(3).CellCount(2).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory),
			defaultArgs().HostThreadsPerCore(3).CellCount(3).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(factory).CPUTuneThreads(1, 7, 0, 10, 5, 11, 6, 2, 8, 3, 9, 4),
		)
	}
	generatePositiveCPUPinningTests(
		defaultArgs().HostThreadsPerCore(1).CellCount(3).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(NewRelaxedCPUPool).CPUTuneThreads(1, 7, 0, 2, 8, 3, 4, 10, 5, 6, 9, 11),
		defaultArgs().HostThreadsPerCore(2).CellCount(3).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(NewRelaxedCPUPool).CPUTuneThreads(1, 7, 0, 2, 8, 3, 4, 10, 5, 6, 9, 11),
		defaultArgs().HostThreadsPerCore(3).CellCount(3).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(NewRelaxedCPUPool).CPUTuneThreads(1, 7, 10, 5, 6, 2, 3, 9, 0, 11, 8, 4),
	)
	generateNegativeCPUPinningTests(
		defaultArgs().HostThreadsPerCore(1).CellCount(3).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(NewStrictCPUPool),
		defaultArgs().HostThreadsPerCore(2).CellCount(3).VCPUCores(4).VCPUThreadsPerCore(3).CPUPoolFactory(NewStrictCPUPool),
		defaultArgs().HostThreadsPerCore(3).CellCount(3).VCPUCores(6).VCPUThreadsPerCore(2).CPUPoolFactory(NewStrictCPUPool),
	)

	It("should depend on the host topology reporting order for stable results with random shuffled cpusets", func() {
		for x := 0; x < 10; x++ {
			pool := NewRelaxedCPUPool(
				&api.CPUTopology{Sockets: 1, Cores: 12, Threads: 1},
				hostTopology(
					1,
					2,
					1, 7,
					0, 6,
					2, 8,
					3, 9,
					4, 10,
					5, 11,
				),
				shuffleCPUSet(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
			)
			cpuTune, err := pool.FitCores()
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuTuneToThreads(cpuTune)).To(Equal([]int{1, 7, 0, 6, 2, 8, 3, 9, 4, 10, 5, 11}))
		}
	})

	It("should fail assigning vCPUs if there are not enough cores reserved", func() {
		pool := NewRelaxedCPUPool(
			&api.CPUTopology{Sockets: 1, Cores: 12, Threads: 1},
			hostTopology(
				1,
				2,
				1, 7,
				0, 6,
				2, 8,
				3, 9,
				4, 10,
				5, 11,
			),
			shuffleCPUSet(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
		)
		_, err := pool.FitCores()
		Expect(err).To(MatchError(ContainSubstring("not enough exclusive threads provided, could not fit 1 core(s)")))
	})

	It("should fail assigning vCPUs with the strict policy if cores can't fully be place on a numa node", func() {
		pool := NewStrictCPUPool(
			&api.CPUTopology{Sockets: 1, Cores: 3, Threads: 2},
			hostTopology(
				2,
				3,
				1, 7, 0,
				6, 2, 8,
			),
			shuffleCPUSet(0, 1, 2, 6, 7, 8),
		)
		_, err := pool.FitCores()
		Expect(err).To(MatchError(ContainSubstring("could not fit 1 core(s) without crossing numa cell boundaries for individual cores")))
	})

	It("should pass assigning vCPUs with the relaxed policy if cores can't fully be place on a numa node", func() {
		pool := NewRelaxedCPUPool(
			&api.CPUTopology{Sockets: 1, Cores: 3, Threads: 2},
			hostTopology(
				2,
				3,
				1, 7, 0,
				6, 2, 8,
			),
			shuffleCPUSet(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11),
		)
		cpuTune, err := pool.FitCores()
		Expect(err).ToNot(HaveOccurred())
		Expect(cpuTuneToThreads(cpuTune)).To(Equal([]int{1, 7, 6, 2, 0, 8}))
	})

	DescribeTable("should pick individual threads", func(threadsPerCore int, cpuSet []int, expectedMapping []uint32) {
		pool := NewRelaxedCPUPool(
			&api.CPUTopology{Sockets: 1, Cores: 1, Threads: 2},
			hostTopology(
				2,
				threadsPerCore,
				1, 7, 0, 6, 2, 8, 3, 9, 4, 10, 5, 11,
			),
			shuffleCPUSet(cpuSet...),
		)

		threadCandidates := []uint32{}

		for {
			thread, err := pool.FitThread()
			if err != nil {
				break
			}
			threadCandidates = append(threadCandidates, thread)
		}

		Expect(threadCandidates).To(Equal(expectedMapping))
	},
		Entry("with 1 thread per host cpu and no missing CPUs in a predictable order",
			1,
			[]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			[]uint32{1, 7, 0, 6, 2, 8, 3, 9, 4, 10, 5, 11},
		),
		Entry("with 2 thread per host cpu and missing CPUs (1, 0) from small chunks first",
			2,
			[]int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
			[]uint32{7, 6, 2, 8, 3, 9, 4, 10, 5, 11},
		),
	)
})

func shuffleCPUSet(cpuSet ...int) []int {
	rand.Shuffle(len(cpuSet), func(i, j int) { cpuSet[i], cpuSet[j] = cpuSet[j], cpuSet[i] })
	return cpuSet
}

func cpuTuneToThreads(cpuTune *api.CPUTune) (threads []int) {
	for _, vcpu := range cpuTune.VCPUPin {
		hostThread, err := strconv.Atoi(vcpu.CPUSet)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		threads = append(threads, hostThread)
	}
	return threads
}

func hostTopology(cellCount int, threadsPerCore int, threads ...uint32) *v1.Topology {
	topology := &v1.Topology{
		NumaCells: []*v1.Cell{},
	}

	for i := 0; i < cellCount; i++ {
		topology.NumaCells = append(topology.NumaCells, &v1.Cell{Id: uint32(i)})
	}

	cellCounter := 0
	threadsPerCell := len(threads) / cellCount
	threadsPerCell = threadsPerCell - threadsPerCell%threadsPerCore
	assigned := 0
	for i := 0; i < len(threads); i += threadsPerCore {
		for col := 0; col < threadsPerCore; col++ {
			topology.NumaCells[cellCounter].Cpus = append(topology.NumaCells[cellCounter].Cpus, &v1.CPU{
				Id:       threads[i+col],
				Siblings: threads[i : i+threadsPerCore],
			})
			assigned++
		}
		if assigned >= threadsPerCell {
			assigned = 0
			cellCounter++
			if cellCounter >= cellCount {
				cellCounter = 0
			}
		}
	}
	Expect(topology.NumaCells).To(HaveLen(cellCount))
	for _, cell := range topology.NumaCells {
		Expect(len(cell.Cpus)).To(BeNumerically(">=", threadsPerCell))
	}

	return topology
}

type testArgs struct {
	cpuPoolFactory     factoryFunc
	hostThreadsPerCore int
	vcpuThreadsPerCore int
	vcpuCores          int
	cellCount          int
	threads            []uint32
	cpuSet             []int
	cpuTuneThreads     []int
	focus              bool
}

func defaultArgs() *testArgs {
	return &testArgs{
		cpuPoolFactory:     NewRelaxedCPUPool,
		vcpuThreadsPerCore: 2,
		vcpuCores:          6,
		hostThreadsPerCore: 1,
		cellCount:          1,
		threads:            []uint32{1, 7, 0, 6, 2, 8, 3, 9, 4, 10, 5, 11},
		cpuSet:             []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		cpuTuneThreads:     []int{1, 7, 0, 6, 2, 8, 3, 9, 4, 10, 5, 11},
	}
}

func (t *testArgs) HostThreadsPerCore(threadsPerCore int) *testArgs {
	t.hostThreadsPerCore = threadsPerCore
	return t
}

func (t *testArgs) Focus() *testArgs {
	t.focus = true
	return t
}

func (t *testArgs) VCPUThreadsPerCore(threadsPerCore int) *testArgs {
	t.vcpuThreadsPerCore = threadsPerCore
	return t
}

func (t *testArgs) VCPUCores(cores int) *testArgs {
	t.vcpuCores = cores
	return t
}

func (t *testArgs) CellCount(cellCount int) *testArgs {
	t.cellCount = cellCount
	return t
}

func (t *testArgs) CPUTuneThreads(cpuTuneThreads ...int) *testArgs {
	t.cpuTuneThreads = cpuTuneThreads
	return t
}

func (t *testArgs) CPUPoolFactory(factory factoryFunc) *testArgs {
	t.cpuPoolFactory = factory
	return t
}

func generatePositiveCPUPinningTests(testArgs ...*testArgs) {
	for idx := range testArgs {
		args := testArgs[idx]
		name := funcName(args.cpuPoolFactory)
		f := It
		if args.focus {
			f = FIt
		}
		f(fmt.Sprintf("should pin %v vcpus with %v thread(s) to %v host cpus with %v thread(s) per core and %v numa node(s), using %v",
			args.vcpuCores,
			args.vcpuThreadsPerCore,
			len(args.threads)/args.hostThreadsPerCore,
			args.hostThreadsPerCore,
			args.cellCount,
			name,
		), func() {
			pool := args.cpuPoolFactory(
				&api.CPUTopology{Sockets: 1, Cores: uint32(args.vcpuCores), Threads: uint32(args.vcpuThreadsPerCore)},
				hostTopology(
					args.cellCount,
					args.hostThreadsPerCore,
					args.threads...,
				),
				shuffleCPUSet(args.cpuSet...),
			)
			cpuTune, err := pool.FitCores()
			Expect(err).ToNot(HaveOccurred())
			Expect(cpuTuneToThreads(cpuTune)).To(Equal(args.cpuTuneThreads))
		})
	}
}

func generateNegativeCPUPinningTests(testArgs ...*testArgs) {
	for idx := range testArgs {
		args := testArgs[idx]
		name := funcName(args.cpuPoolFactory)
		f := It
		if args.focus {
			f = FIt
		}
		f(fmt.Sprintf("should not pin %v vcpus with %v thread(s) to %v host cpus with %v thread(s) per core and %v numa node(s), using %v",
			args.vcpuCores,
			args.vcpuThreadsPerCore,
			len(args.threads)/args.hostThreadsPerCore,
			args.hostThreadsPerCore,
			args.cellCount,
			name,
		), func() {
			pool := args.cpuPoolFactory(
				&api.CPUTopology{Sockets: 1, Cores: uint32(args.vcpuCores), Threads: uint32(args.vcpuThreadsPerCore)},
				hostTopology(
					args.cellCount,
					args.hostThreadsPerCore,
					args.threads...,
				),
				shuffleCPUSet(args.cpuSet...),
			)
			_, err := pool.FitCores()
			Expect(err).To(HaveOccurred())
		})
	}
}
func funcName(f interface{}) string {
	arr := strings.Split(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), ".")
	return arr[len(arr)-1]
}
