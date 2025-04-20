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

package logverbosity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	jsonpatch "github.com/evanphx/json-patch"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/adm/logverbosity"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

var _ = Describe("Log Verbosity", func() {
	var kvInterface *kubecli.MockKubeVirtInterface

	var kv *v1.KubeVirt
	var kvs *v1.KubeVirtList

	const (
		installNamespace = "kubevirt"
		installName      = "kubevirt"
	)

	commonShowDescribeTable := func() {
		DescribeTable("show operation", commonShowTest,
			Entry("all components", []uint{2, 2, 2, 2, 2}, "--all"),
			Entry(
				"one component (1st component (i.e. virt-api))",
				[]uint{2, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-api",
			),
			Entry(
				"one component (last component (i.e. virt-operator))",
				[]uint{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, 2},
				"--virt-operator",
			),
			Entry(
				"two components",
				[]uint{logverbosity.NoFlag, 2, 2, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-controller",
				"--virt-handler",
			),
			Entry("all + one component", []uint{2, 2, 2, 2, 2}, "--all", "--virt-launcher"),
		)
	}

	commonSetDescribeTable := func() {
		DescribeTable("set", func(output []uint, args ...string) {
			// should set logVerbosity field for the specified components in the KubeVirt CR
			commonSetCommand(args...)

			expectAllComponentVerbosity(kv, output) // check the verbosity of all components if it is expected
		},
			Entry("one component (1st component (i.e. virt-api))", []uint{1, 0, 0, 0, 0}, "--virt-api=1"),
			Entry("one component (last component (i.e. virt-operator))", []uint{0, 0, 0, 0, 2}, "--virt-operator=2"),
			Entry("two components", []uint{0, 3, 4, 0, 0}, "--virt-controller=3", "--virt-handler=4"),
			Entry("other two components", []uint{0, 0, 0, 5, 6}, "--virt-launcher=5", "--virt-operator=6"),
			Entry("all components", []uint{7, 7, 7, 7, 7}, "--all=7"),
			// corner case
			Entry("same component different verbosity (last one is a winner)", []uint{4, 0, 0, 0, 0}, "--virt-api=3", "--virt-api=4"),
		)
	}

	BeforeEach(func() {
		kv = NewKubeVirtWithoutDeveloperConfiguration(installNamespace, installName)
		kvs = kubecli.NewKubeVirtList(*kv)

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		kvInterface = kubecli.NewMockKubeVirtInterface(ctrl)

		kubecli.MockKubevirtClientInstance.EXPECT().KubeVirt(kvs.Items[0].Namespace).Return(kvInterface).AnyTimes() // Get & Patch
		kubecli.MockKubevirtClientInstance.EXPECT().KubeVirt(k8smetav1.NamespaceAll).Return(kvInterface).AnyTimes() // List

		kvInterface.EXPECT().Patch(context.Background(), gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, name string, _ any, patchData []byte, _ any, _ ...any) (*v1.KubeVirt, error) {
				Expect(name).To(Equal(kvs.Items[0].Name))

				patch, err := jsonpatch.DecodePatch(patchData)
				Expect(err).ToNot(HaveOccurred())
				kvJSON, err := json.Marshal(kv)
				Expect(err).ToNot(HaveOccurred())
				modifiedKvJSON, err := patch.Apply(kvJSON)
				Expect(err).ToNot(HaveOccurred())

				// reset the object in preparation for unmarshal,
				// since unmarshal does not guarantee that fields in kv will be removed by the patch
				kv = &v1.KubeVirt{}

				err = json.Unmarshal(modifiedKvJSON, kv)
				Expect(err).ToNot(HaveOccurred())
				return kv, nil
			}).AnyTimes()
	})

	When("with erroneous running environment", func() {
		Context("client has an error", func() {
			BeforeEach(func() {
				// GET and LIST mock interfaces are not necessary, because an error is returned before GET and LIST are called
				kubecli.GetKubevirtClientFromClientConfig = kubecli.GetInvalidKubevirtClientFromClientConfig
			})

			It("should fail (not executing the command)", func() {
				cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all")
				Expect(cmd).NotTo(BeNil())
			})
		})

		Context("detectInstallNamespaceAndName has en error", func() {
			expectListError := func() {
				kvInterface.EXPECT().List(context.Background(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, _ any) (*v1.KubeVirt, error) {
						return nil, errors.New("List error")
					}).AnyTimes()
			}

			It("should fail", func() {
				expectListError() // simulate something like no permission to access the namespace
				cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all")
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("could not list KubeVirt CRs across all namespaces: List error")))
			})
		})

		Context("Get function has an error", func() {
			BeforeEach(func() {
				kvInterface.EXPECT().List(context.Background(), gomock.Any()).Return(kvs, nil).AnyTimes()
			})

			expectGetError := func() {
				kvInterface.EXPECT().Get(context.Background(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, _ any) (*v1.KubeVirt, error) {
						Expect(name).To(Equal(kvs.Items[0].Name))
						return nil, errors.New("Get error")
					}).AnyTimes()
			}

			It("should fail", func() {
				expectGetError() // for some reason, Get function returns an error
				cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all")
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("Get error")))
			})
		})
	})

	When("with install namespace and name other than kubevirt", func() {
		BeforeEach(func() {
			kv = NewKubeVirtWithoutDeveloperConfiguration("foo", "foo")
			kvs = kubecli.NewKubeVirtList(*kv)

			kubecli.MockKubevirtClientInstance.EXPECT().KubeVirt(kvs.Items[0].Namespace).Return(kvInterface).AnyTimes() // Get & Patch
			kvInterface.EXPECT().List(context.Background(), gomock.Any()).Return(kvs, nil).AnyTimes()
		})

		expectGetKv := func() {
			kvInterface.EXPECT().Get(context.Background(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, _ any) (*v1.KubeVirt, error) {
					Expect(name).To(Equal(kvs.Items[0].Name))
					return &kvs.Items[0], nil
				}).AnyTimes()
		}

		It("show: should succeed", func() {
			expectGetKv()
			bytes, err := testing.NewRepeatableVirtctlCommandWithOut("adm", "log-verbosity", "--all")()
			Expect(err).To(Succeed())
			output := []uint{2, 2, 2, 2, 2}
			message := createOutputMessage(output)
			Expect(string(bytes)).To(ContainSubstring(*message))
		})

		It("set: should succeed", func() {
			expectGetKv()
			cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all=7")
			Expect(cmd()).To(Succeed())
			output := []uint{7, 7, 7, 7, 7}
			expectAllComponentVerbosity(kv, output)
		})
	})

	When("with invalid set of flags", func() {
		BeforeEach(func() {
			commonSetup(kvInterface, kvs)
		})

		Context("with empty set of flags", func() {
			It("should fail (return help)", func() {
				cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity")
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("no flag specified - expecting at least one flag")))
			})
		})

		DescribeTable("should fail handled by the CLI package", func(args ...string) {
			argStr := strings.Join(args, ",")
			cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", argStr)
			Expect(cmd()).NotTo(Succeed())
		},
			Entry("reset and all coexist", "--reset", "--all=3"),
			Entry("invalid argument (negative verbosity)", "--virt-api=-1"),
			Entry("invalid argument (character)", "--virt-api=a"),
			Entry("unknown flag", "--node"),
			Entry("invalid flag format", "--all", "3"),
		)

		DescribeTable("should fail handled by error handler", func(output string, args ...string) {
			commandAndArgs := []string{"adm", "log-verbosity"}
			commandAndArgs = append(commandAndArgs, args...)
			_, err := testing.NewRepeatableVirtctlCommandWithOut(commandAndArgs...)()
			Expect(err).NotTo(Succeed())

			Expect(err).To(MatchError(ContainSubstring(output)))
		},
			Entry("show and set mix", "only show or set is allowed", "--virt-handler", "--virt-launcher=3"),
			Entry("show and reset mix", "only show or set is allowed", "--reset", "--virt-launcher"),
			Entry("invalid verbosity (=noFlag)", "virt-api: log verbosity must be 0-9", "--virt-api=11"),
			Entry("invalid verbosity", "virt-api: log verbosity must be 0-9", "--virt-api=20"),
			Entry("one valid verbosity, one invalid verbosity", "virt-handler: log verbosity must be 0-9", "--virt-api=5", "--virt-handler=20"),
		)
	})

	When("no DeveloperConfiguration field in the KubeVirt CR", func() {
		BeforeEach(func() {
			commonSetup(kvInterface, kvs)
		})

		// fill the unattended verbosity with default verbosity (2)
		commonShowDescribeTable()

		Describe("set operation", func() {
			Context("reset", func() {
				It("do nothing", func() {
					cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--reset")
					Expect(cmd()).To(Succeed())
					Expect(kv.Spec.Configuration.DeveloperConfiguration).To(BeNil())
				})
			})

			commonSetDescribeTable()
		})
	})

	When("no logVerbosity field in the KubeVirt CR", func() {
		BeforeEach(func() {
			dc := &v1.DeveloperConfiguration{}
			kv.Spec.Configuration.DeveloperConfiguration = dc
			kvs = kubecli.NewKubeVirtList(*kv)

			commonSetup(kvInterface, kvs)
		})

		// fill the unattended verbosity with default verbosity (2)
		commonShowDescribeTable()

		Describe("set operation", func() {
			Context("reset", func() {
				It("do nothing", func() {
					cmd := testing.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--reset")
					Expect(cmd()).To(Succeed())
					Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity).To(BeNil())
				})
			})

			commonSetDescribeTable()
		})
	})

	When("existing logVerbosity in the KubeVirt CR", func() {
		BeforeEach(func() {
			dc := &v1.DeveloperConfiguration{
				LogVerbosity: &v1.LogVerbosity{
					VirtAPI:        5,
					VirtController: 6,
					VirtLauncher:   3,
					VirtOperator:   4,
				},
			}
			kv.Spec.Configuration.DeveloperConfiguration = dc
			kvs = kubecli.NewKubeVirtList(*kv)

			commonSetup(kvInterface, kvs)
		})

		// should show the verbosity for components from the KubeVirt CR
		// get and show the attended verbosity
		// show the default verbosity (2), when the logVerbosity is unattended
		DescribeTable("show operation", commonShowTest,
			Entry("all components", []uint{5, 6, 2, 3, 4}, "--all"),
			Entry(
				"one component attended verbosity",
				[]uint{5, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-api",
			),
			Entry(
				"one component unattended verbosity",
				[]uint{logverbosity.NoFlag, logverbosity.NoFlag, 2, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-handler",
			),
			Entry(
				"two components with one unattended verbosity",
				[]uint{logverbosity.NoFlag, 6, 2, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-handler",
				"--virt-controller",
			),
			// corner case
			Entry("all components with default argument (equals show operation)", []uint{5, 6, 2, 3, 4}, "--all=10"),
		)

		Describe("set operation", func() {
			DescribeTable("set", func(output []uint, args ...string) {
				// should set logVerbosity filed for the specified components in the KubeVirt CR
				commonSetCommand(args...)

				expectAllComponentVerbosity(kv, output)
			},
				Entry("reset", []uint{0, 0, 0, 0, 0}, "--reset"), // CR's logVerbosity field is replaced by {}. logVerbosity struct of each filed is 0.
				Entry("one component (1st component (i.e. virt-api))", []uint{1, 6, 0, 3, 4}, "--virt-api=1"),
				Entry("one component (last component (i.e. virt-operator))", []uint{5, 6, 0, 3, 2}, "--virt-operator=2"),
				Entry("one component (filled in unattended verbosity)", []uint{5, 6, 8, 3, 4}, "--virt-handler=8"),
				Entry("all components", []uint{7, 7, 7, 7, 7}, "--all=7"),
				Entry("two components", []uint{5, 0, 9, 3, 4}, "--virt-controller=0", "--virt-handler=9"),
				Entry("set all and then set two components", []uint{9, 0, 8, 8, 8}, "--all=8", "--virt-api=9", "--virt-controller=0"),
				Entry("reset and then set two components", []uint{0, 0, 1, 2, 0}, "--reset", "--virt-handler=1", "--virt-launcher=2"),
				// corner case
				Entry("two same operations (come down to one operation)", []uint{3, 6, 0, 3, 4}, "--virt-api=3", "--virt-api=3"),
				Entry("same component different verbosity (last one is a winner)", []uint{4, 6, 0, 3, 4}, "--virt-api=3", "--virt-api=4"),
			)
		})
	})

})

func expectAllComponentVerbosity(kv *v1.KubeVirt, output []uint) {
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtAPI).To(Equal(output[0]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtController).To(Equal(output[1]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtHandler).To(Equal(output[2]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtLauncher).To(Equal(output[3]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtOperator).To(Equal(output[4]))
}

// create an expected output message
func createOutputMessage(output []uint) *string {
	var message string
	var components = []string{"virt-api", "virt-controller", "virt-handler", "virt-launcher", "virt-operator"}
	for component := 0; component < len(components); component++ {
		if output[component] == logverbosity.NoFlag {
			continue
		}
		// output format is [componentName]=[verbosity] like:
		// 		virt-api=1
		// 		virt-controller=2
		componentName := components[component]
		verbosity := output[component]
		message += fmt.Sprintf("%s=%d\n", componentName, verbosity)
	}
	return &message
}

func NewKubeVirtWithoutDeveloperConfiguration(namespace, name string) *v1.KubeVirt {
	return &v1.KubeVirt{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "KubeVirt",
			APIVersion: v1.GroupVersion.String(),
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1.KubeVirtSpec{
			ImageTag:      "devel",
			Configuration: v1.KubeVirtConfiguration{},
		},
	}
}

func commonSetup(kvInterface *kubecli.MockKubeVirtInterface, kvs *v1.KubeVirtList) {
	kvInterface.EXPECT().List(context.Background(), gomock.Any()).Return(kvs, nil).AnyTimes()
	kvInterface.EXPECT().Get(context.Background(), kvs.Items[0].Name, gomock.Any()).Return(&kvs.Items[0], nil).AnyTimes()
}

func commonShowTest(output []uint, args ...string) {
	commandAndArgs := []string{"adm", "log-verbosity"}
	commandAndArgs = append(commandAndArgs, args...)
	bytes, err := testing.NewRepeatableVirtctlCommandWithOut(commandAndArgs...)()
	Expect(err).To(Succeed())

	message := createOutputMessage(output) // create an expected output message
	Expect(string(bytes)).To(ContainSubstring(*message))
}

func commonSetCommand(args ...string) {
	commandAndArgs := []string{"adm", "log-verbosity"}
	commandAndArgs = append(commandAndArgs, args...)
	cmd := testing.NewRepeatableVirtctlCommand(commandAndArgs...)
	Expect(cmd()).To(Succeed())
}
