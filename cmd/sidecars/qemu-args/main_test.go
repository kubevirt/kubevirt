package main

import (
	"log"
	"os"
	"sort"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("with custom QEMU command line options", func() {
	Context("Merging custom QEMU command line options", func() {
		logger = log.New(os.Stderr, "test", log.Ldate)

		DescribeTable("with different configurations", func(curArgs, curEnvs, annArgs, annEnvs, resArgs, resEnvs []string) {
			var annotation = createCommandLine(annArgs, annEnvs)
			var current = createCommandLine(curArgs, curEnvs)
			var result = createCommandLine(resArgs, resEnvs)

			var merged = mergeQEMUCmd(current, annotation)

			Expect(merged.QEMUArg).To(HaveLen(len(result.QEMUArg)))
			Expect(merged.QEMUEnv).To(HaveLen(len(result.QEMUEnv)))
			// With args, we kept the order intact as that might be important
			Expect(merged.QEMUArg).To(Equal(result.QEMUArg))

			// Sort envs to compare them and avoid flaky results
			sort.Slice(merged.QEMUEnv, func(i, j int) bool {
				return strings.Compare(merged.QEMUEnv[i].Name, merged.QEMUEnv[j].Name) < 0
			})
			sort.Slice(result.QEMUEnv, func(i, j int) bool {
				return strings.Compare(result.QEMUEnv[i].Name, result.QEMUEnv[j].Name) < 0
			})
			Expect(merged.QEMUEnv).To(Equal(result.QEMUEnv))
		},
			Entry("No conflict, combine both",
				// Existing QEMU CommandLine
				[]string{"one", "two"},
				[]string{"foo1", "bar1", "foo2", "bar2"},
				// The Annotation sidecar will handle
				[]string{"three"},
				[]string{"foo3", "bar3"},
				// Expected result
				[]string{"one", "two", "three"},
				[]string{"foo1", "bar1", "foo2", "bar2", "foo3", "bar3"},
			),
			Entry("No current, only new args",
				// Existing QEMU CommandLine
				[]string{},
				[]string{},
				// The Annotation sidecar will handle
				[]string{"one"},
				[]string{"foo1", "bar1"},
				// Expected result
				[]string{"one"},
				[]string{"foo1", "bar1"},
			),
			Entry("Conflict args, already exists",
				// Existing QEMU CommandLine
				[]string{"one", "two"},
				[]string{"foo1", "bar1", "foo2", "bar2"},
				// The Annotation sidecar will handle
				[]string{"two"},
				[]string{"foo3", "bar3"},
				// Expected result
				[]string{"one", "two"},
				[]string{"foo1", "bar1", "foo2", "bar2", "foo3", "bar3"},
			),
			Entry("Conflict envs, should replace value",
				// Existing QEMU CommandLine
				[]string{"one", "two"},
				[]string{"foo1", "bar1", "foo2", "bar2"},
				// The Annotation sidecar will handle
				[]string{"three"},
				[]string{"foo1", "bar3"},
				// Expected result
				[]string{"one", "two", "three"},
				[]string{"foo1", "bar3", "foo2", "bar2"},
			),
		)
	})
})

func createArgs(strargs []string) []api.Arg {
	args := make([]api.Arg, len(strargs))
	for i, data := range strargs {
		args[i].Value = data
	}
	return args
}

func createEnvs(strenvs []string) []api.Env {
	envs := make([]api.Env, len(strenvs)/2)
	for i := 0; i < len(strenvs)/2; i++ {
		envs[i].Name = strenvs[i*2]
		envs[i].Value = strenvs[i*2+1]
	}
	return envs
}

func createCommandLine(args, envs []string) *api.Commandline {
	return &api.Commandline{
		QEMUArg: createArgs(args),
		QEMUEnv: createEnvs(envs),
	}
}
