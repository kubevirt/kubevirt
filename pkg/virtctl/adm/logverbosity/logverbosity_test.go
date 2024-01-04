package logverbosity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/tests/clientcmd"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"kubevirt.io/client-go/kubecli"

	corev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/adm/logverbosity"
)

var _ = Describe("Log Verbosity", func() {
	// for virt component verbosity
	var kvInterface *kubecli.MockKubeVirtInterface

	var kv *v1.KubeVirt
	var kvs *v1.KubeVirtList

	const (
		installNamespace = "kubevirt"
		installName      = "kubevirt"
	)

	// for vm level verbosity
	var kubeClient *fake.Clientset
	var vmInterface *kubecli.MockVirtualMachineInterface

	var testVM1 *v1.VirtualMachine
	var testVM2 *v1.VirtualMachine

	var patchedVM1 *v1.VirtualMachine
	var patchedVM2 *v1.VirtualMachine

	var virtLauncherPod1 *corev1.Pod
	var virtLauncherPod2 *corev1.Pod
	var virtLauncerPodDummy *corev1.Pod
	var provisionerPod *corev1.Pod
	var pods *corev1.PodList

	const (
		vmName1                  = "testvm1"
		vmName2                  = "testvm2"
		vmNamespace              = "default"
		virtLauncherPodName      = "virt-launcher-"
		dummyVirtLauncherPodName = "virt-launcher-dummy-"
		provisionerPodName       = "local-volume-provisioner-"
	)

	// for virt component verbosity
	commonShowDescribeTable := func() {
		DescribeTable("show operation", commonCompShowTest,
			Entry("all components", []string{"2", "2", "2", "2", "2"}, "--all"),
			Entry(
				"one component (1st component (i.e. virt-api))",
				[]string{"2", logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-api",
			),
			Entry(
				"one component (last component (i.e. virt-operator))",
				[]string{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, "2"},
				"--virt-operator",
			),
			Entry(
				"two components",
				[]string{logverbosity.NoFlag, "2", "2", logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-controller",
				"--virt-handler",
			),
			Entry("all + one component", []string{"2", "2", "2", "2", "2"}, "--all", "--virt-launcher"),
			// corner case
			Entry("all=noArg", []string{"2", "2", "2", "2", "2"}, "--all=show"),
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

	// for vm level verbosity
	setRunningState := func(state bool) {
		testVM1.Spec.Running = &state
		testVM2.Spec.Running = &state
	}

	configureAndStartVMs := func() {
		setRunningState(true)

		virtLauncherPod1 = createVirtLauncherPod(virtLauncherPodName+vmName1+"-", vmNamespace, vmName1, getVerbosityFromLabels(testVM1))
		virtLauncherPod2 = createVirtLauncherPod(virtLauncherPodName+vmName2+"-", vmNamespace, vmName2, getVerbosityFromLabels(testVM2))
		virtLauncerPodDummy = newPodNoVerbosity(dummyVirtLauncherPodName, vmNamespace)
		provisionerPod = newPodNoVerbosity(provisionerPodName, vmNamespace)

		pods = newPodList(*virtLauncherPod1, *virtLauncherPod2, *virtLauncerPodDummy, *provisionerPod)
	}

	expectedPatch := func(vm1, vm2 *v1.VirtualMachine) {
		vmInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, name string, _ any, patchData []byte, _ any, _ ...any) (*v1.VirtualMachine, error) {
				var err error
				switch name {
				case vmName1:
					patchedVM1, err = applyVMPatch(patchData, vm1)
					return patchedVM1, err
				case vmName2:
					patchedVM2, err = applyVMPatch(patchData, vm2)
					return patchedVM2, err
				default:
					return nil, errors.New("patch error")
				}
			}).AnyTimes()
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

		kvInterface.EXPECT().Patch(gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
			func(name string, _ any, patchData []byte, _ any, _ ...any) (*v1.KubeVirt, error) {
				Expect(name).To(Equal(kvs.Items[0].Name))

				patch, err := jsonpatch.DecodePatch(patchData)
				Expect(err).ToNot(HaveOccurred())

				kvJSON, err := json.Marshal(kvs.Items[0])
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

		// for VM level verbosity
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubeClient = fake.NewSimpleClientset()

		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(vmNamespace).Return(vmInterface).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		kubeClient.Fake.PrependReactor("list", "pods", func(action testing.Action) (bool, runtime.Object, error) {
			return true, pods, nil
		})

		testVM1 = newVMNoVerbosityLabel(vmName1, vmNamespace)
		testVM2 = newVMNoVerbosityLabel(vmName2, vmNamespace)

		vmInterface.EXPECT().Get(context.Background(), testVM1.Name, gomock.Any()).Return(testVM1, nil).AnyTimes()
		vmInterface.EXPECT().Get(context.Background(), testVM2.Name, gomock.Any()).Return(testVM2, nil).AnyTimes()
	})

	When("with erroneous running environment", func() {
		Context("client has an error", func() {
			BeforeEach(func() {
				// GET and LIST mock interfaces are not necessary, because an error is returned before GET and LIST are called
				kubecli.GetKubevirtClientFromClientConfig = kubecli.GetInvalidKubevirtClientFromClientConfig
			})

			It("should fail (not executing the command)", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all")
				Expect(cmd).NotTo(BeNil())
			})
		})

		Context("detectInstallNamespaceAndName has en error", func() {
			expectListError := func() {
				kvInterface.EXPECT().List(gomock.Any()).DoAndReturn(
					func(_ any) (*v1.KubeVirt, error) {
						return nil, errors.New("List error")
					}).AnyTimes()
			}

			It("should fail", func() {
				expectListError() // simulate something like no permission to access the namespace
				cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all")
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("could not list KubeVirt CRs across all namespaces: List error")))
			})
		})

		Context("Get function has an error", func() {
			BeforeEach(func() {
				kvInterface.EXPECT().List(gomock.Any()).Return(kvs, nil).AnyTimes()
			})

			expectGetError := func() {
				kvInterface.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(
					func(name string, _ any) (*v1.KubeVirt, error) {
						Expect(name).To(Equal(kvs.Items[0].Name))
						return nil, errors.New("Get error")
					}).AnyTimes()
			}

			It("should fail", func() {
				expectGetError() // for some reason, Get function returns an error
				cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all")
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
			kvInterface.EXPECT().List(gomock.Any()).Return(kvs, nil).AnyTimes()
		})

		expectGetKv := func() {
			kvInterface.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(
				func(name string, _ any) (*v1.KubeVirt, error) {
					Expect(name).To(Equal(kvs.Items[0].Name))
					return &kvs.Items[0], nil
				}).AnyTimes()
		}

		It("show: should succeed", func() {
			expectGetKv()
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut("adm", "log-verbosity", "--all")()
			Expect(err).To(Succeed())
			output := []string{"2", "2", "2", "2", "2"}
			lines := createOutputMessage(output)
			message := strings.Join(lines, "\n")
			Expect(string(bytes)).To(ContainSubstring(message))
		})

		It("set: should succeed", func() {
			expectGetKv()
			cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all=7")
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
				cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity")
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("no flag specified - expecting at least one flag")))
			})
		})

		Context("same as the NoFlag variable", func() {
			It("return help", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all="+logverbosity.NoFlag)
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("no flag specified - expecting at least one flag")))
			})
		})

		DescribeTable("should fail handled by the CLI package", func(args ...string) {
			commandAndArgs := []string{"adm", "log-verbosity"}
			commandAndArgs = append(commandAndArgs, args...)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Expect(cmd()).NotTo(Succeed())
		},
			Entry("unknown flag", "--node"),
			Entry("invalid flag format", "--all", "3"),
		)

		DescribeTable("should fail handled by error handler", func(output string, args ...string) {
			commandAndArgs := []string{"adm", "log-verbosity"}
			commandAndArgs = append(commandAndArgs, args...)
			_, err := clientcmd.NewRepeatableVirtctlCommandWithOut(commandAndArgs...)()
			Expect(err).NotTo(Succeed())

			Expect(err).To(MatchError(ContainSubstring(output)))
		},
			Entry("show and set mix", "only show or set is allowed", "--virt-handler", "--virt-launcher=3"),
			Entry("show and reset mix", "only show or set is allowed", "--reset", "--virt-launcher"),
			Entry("invalid verbosity (negative verbosity)", "virt-api: log verbosity must be 0-9", "--virt-api=-1"),
			Entry("invalid verbosity (character)", "virt-api: log verbosity must be 0-9", "--virt-api=a"),
			Entry("invalid verbosity (boarder)", "virt-api: log verbosity must be 0-9", "--virt-api=10"),
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
					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--reset")
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
					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--reset")
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
		DescribeTable("show operation", commonCompShowTest,
			Entry("all components", []string{"5", "6", "2", "3", "4"}, "--all"),
			Entry(
				"one component attended verbosity",
				[]string{"5", logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-api",
			),
			Entry(
				"one component unattended verbosity",
				[]string{logverbosity.NoFlag, logverbosity.NoFlag, "2", logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-handler",
			),
			Entry(
				"two components with one unattended verbosity",
				[]string{logverbosity.NoFlag, "6", "2", logverbosity.NoFlag, logverbosity.NoFlag},
				"--virt-handler",
				"--virt-controller",
			),
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
				Entry("set all and reset", []uint{3, 3, 3, 3, 3}, "--all=3", "--reset"),
				// corner case
				Entry("two same operations (come down to one operation)", []uint{3, 6, 0, 3, 4}, "--virt-api=3", "--virt-api=3"),
				Entry("same component different verbosity (last one is a winner)", []uint{4, 6, 0, 3, 4}, "--virt-api=3", "--virt-api=4"),
			)
		})
	})

	/*
	 * for vm level verbosity
	 */
	When("with invalid set of flags and arguments for vm", func() {
		BeforeEach(func() {
			commonSetup(kvInterface, kvs)
		})

		Context("same as the default variable", func() {
			It("vm flag: return help", func() {
				cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--vm=")
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("no flag specified - expecting at least one flag")))
			})
			It("level flag: should succeed (just ignore --level flag)", func() {
				// start VMs with no label, and virt-launcher=empty (default)
				configureAndStartVMs()
				compOutput := []string{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag}
				vmOutput := []string{vmName1 + " = 2", ""}
				commonVMShowTest(compOutput, vmOutput, "--vm="+vmName1, "--level=")
			})
		})

		Context("Get function has an error", func() {
			// when the specified name is different from the vm name, Get function returns an error
			expectGetError := func(testName string) {
				vmInterface.EXPECT().Get(context.Background(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ any, name string, _ any) (*v1.VirtualMachine, error) {
						Expect(name).NotTo(Equal(testName))
						return nil, errors.New("Get error")
					}).AnyTimes()
			}

			DescribeTable("should fail", func(args ...string) {
				expectGetError(testVM1.Name)

				commandAndArgs := []string{"adm", "log-verbosity"}
				commandAndArgs = append(commandAndArgs, args...)
				cmd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
				err := cmd()
				Expect(err).NotTo(Succeed())
				Expect(err).To(MatchError(ContainSubstring("Get error")))
			},
				Entry("vm: unknown vm name", "--vm=unknown"),
				Entry("reset: unknown vm name", "--reset=unknown"),
				// coexistence of virt components and vm
				Entry("error on vm (show): unknown vm name", "--virt-api", "--vm=unknown"),
				Entry("error on vm (set): unknown vm name", "--virt-api=1", "--vm=unknown", "--level=5"),
				Entry("error on vm (vm first): vm no arg", "--vm", "--virt-api"), // --virt-api as a parameter of --vm flag
			)
		})

		DescribeTable("should fail handled by the CLI package", func(args ...string) {
			commandAndArgs := []string{"adm", "log-verbosity"}
			commandAndArgs = append(commandAndArgs, args...)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			Expect(cmd()).NotTo(Succeed())
		},
			Entry("vm: no arg", "--vm"),
			Entry("level: no arg", "--vm="+vmName1, "--level"),
			Entry("reset + vm: no arg", "--reset="+vmName1, "--vm"),
			Entry("reset + level: no arg", "--reset="+vmName1, "--vm=testvm1", "--level"),
			// coexistence of virt components and vm
			Entry("error on vm (vm last): vm no arg", "--virt-api", "--vm"),
			Entry("error on vm: level no arg", "--virt=api=1", "--vm="+vmName1, "--level"),
		)

		DescribeTable("should fail handled by error handler", func(output string, args ...string) {
			commandAndArgs := []string{"adm", "log-verbosity"}
			commandAndArgs = append(commandAndArgs, args...)
			_, err := clientcmd.NewRepeatableVirtctlCommandWithOut(commandAndArgs...)()
			Expect(err).NotTo(Succeed())

			Expect(err).To(MatchError(ContainSubstring(output)))
		},
			Entry("level: no vm flag", "level: need vm flag", "--level=5"),
			Entry("reset + level: no vm flag", "level: need vm flag", "--reset="+vmName1, "--level=5"),
			Entry("invalid verbosity (boarder)", vmName1+": log verbosity must be 0-9", "--vm="+vmName1, "--level=10"),
			Entry("invalid verbosity (negative)", vmName1+": log verbosity must be 0-9", "--vm="+vmName1, "--level=-1"),
			Entry("invalid verbosity (character)", vmName1+": log verbosity must be 0-9", "--vm="+vmName1, "--level=a"),
			Entry("show and reset mix (same vm)", "only show or set is allowed", "--reset="+vmName1, "--vm="+vmName1),
			Entry("show and reset mix (different vm)", "only show or set is allowed", "--reset="+vmName1, "--vm="+vmName2),
			Entry(
				"show and set mix (same vm)",
				"number of vm flags 2 not equal to number of level flags 1",
				"--vm="+vmName1, "--vm="+vmName1, "--level=5",
			),
			Entry(
				"show and set mix (different vm)",
				"number of vm flags 2 not equal to number of level flags 1",
				"--vm="+vmName1, "--vm="+vmName2, "--level=5",
			),
			// coexistence of virt components and vm
			Entry("error on component: invalid verbosity (boarder)", "virt-api: log verbosity must be 0-9", "--virt-api=10", "--vm="+vmName1),
			Entry("error on vm: invalid verbosity (boarder)", vmName1+": log verbosity must be 0-9", "--virt-api=1", "--vm="+vmName1, "--level=10"),
			Entry("error on vm: invalid verbosity (negative)", vmName1+": log verbosity must be 0-9", "--virt-api=1", "--vm="+vmName1, "--level=-1"),
			Entry("show (vm) and set (comp) mix", "only show or set is allowed", "--virt-api=1", "--vm="+vmName1),
			Entry("show (vm) and set (all comp) mix", "only show or set is allowed", "--all=1", "--vm="+vmName1),
			Entry("show (vm) and set (reset comp) mix", "only show or set is allowed", "--reset", "--vm="+vmName1),
			Entry("show (comp) and set (vm) mix", "only show or set is allowed", "--virt-api", "--vm="+vmName1, "--level=5"),
			Entry("show (all comp) and set (vm) mix", "only show or set is allowed", "--all", "--vm="+vmName1, "--level=5"),
			Entry("show (comp) and set (reset vm) mix", "only show or set is allowed", "--virt-api", "--reset="+vmName1),
			Entry("show (all comp) and set (rest vm) mix", "only show or set is allowed", "--all", "--reset="+vmName1),
		)
	})

	When("only vm related flags and args", func() {
		BeforeEach(func() {
			// virt-launcher=default (no DeveloperConfiguration field in the KubeVirt CR)
			commonSetup(kvInterface, kvs)
		})

		DescribeTable("set and show 1 VM",
			func(testCase func(), vmOutput []string) {
				// start VMs with no label, and virt-launcher=empty (default)
				configureAndStartVMs()
				testCase()
				if vmOutput != nil {
					compOutput := []string{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag}
					commonVMShowTest(compOutput, vmOutput, "--vm="+vmName1)
				}
			},
			Entry("1 VM, show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{vmName1 + " = 2", ""},
			),
			Entry("1 VM, set logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--vm="+vmName1, "--level=5")
					Expect(cmd()).To(Succeed())
					Expect(patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("5"))
				},
				nil,
			),
			Entry("1 VM, show before restarting VM",
				func() {
					// set the logVerbosity label to an VM object
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "5"
				},
				[]string{vmName1 + " = 2", vmName1 + ": updated verbosity 5 will be applied after the VM is restarted"},
			),
			Entry("1 VM, show after restarting VM",
				func() {
					// restart VMs with label
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "5"
					configureAndStartVMs()
				},
				[]string{vmName1 + " = 5", ""},
			),
		)

		DescribeTable("set and show multiple VMs",
			func(testCase func(), vmOutput []string) {
				// start VMs with no label, and virt-launcher=empty (default)
				configureAndStartVMs()
				testCase()
				if vmOutput != nil {
					compOutput := []string{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag}
					commonVMShowTest(compOutput, vmOutput, "--vm="+vmName1, "--vm="+vmName2)
				}
			},
			Entry("2 VMs, show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{vmName1 + " = 2", "", vmName2 + " = 2", ""},
			),
			Entry("2 VMs, set logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--vm="+vmName1, "--level=8", "--vm="+vmName2, "--level=9")
					Expect(cmd()).To(Succeed())
					Expect(patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("8"))
					Expect(patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("9"))
				},
				nil,
			),
			Entry("2 VMs, show before restarting VM",
				func() {
					// set the logVerbosity label to VM objects
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "8"
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "9"
				},
				[]string{
					vmName1 + " = 2",
					vmName1 + ": updated verbosity 8 will be applied after the VM is restarted",
					vmName2 + " = 2",
					vmName2 + ": updated verbosity 9 will be applied after the VM is restarted",
				},
			),
			Entry("2 VMs, show after restarting VM",
				func() {
					// restart VMs with label
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "8"
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "9"
					configureAndStartVMs()
				},
				[]string{vmName1 + " = 8", "", vmName2 + " = 9", ""},
			),
			Entry("2 VMs (2 --level and then 2 --vm)",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--level=8", "--level=9", "--vm="+vmName1, "--vm="+vmName2)
					Expect(cmd()).To(Succeed())
					Expect(patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("8"))
					Expect(patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("9"))
				},
				nil,
			),
		)

		DescribeTable("reset 2 VMs",
			func(testCase func(), vmOutput []string, presetLabel bool) {
				if presetLabel {
					// start VMs with label (testvm1=5, testvm2=6)
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "5"
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "6"
				}
				configureAndStartVMs()
				testCase()
				if vmOutput != nil {
					compOutput := []string{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag}
					commonVMShowTest(compOutput, vmOutput, "--vm="+vmName1, "--vm="+vmName2)
				}
			},
			Entry("show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{vmName1 + " = 5", "", vmName2 + " = 6", ""},
				true,
			),
			Entry("reset logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--reset="+vmName1, "--reset="+vmName2)
					Expect(cmd()).To(Succeed())
					_, exist := patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]
					Expect(exist).To(BeFalse())
					_, exist = patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]
					Expect(exist).To(BeFalse())
				},
				nil,
				true,
			),
			Entry("show before restarting VM",
				func() {
					// reset/remove the logVerbosity label
					delete(testVM1.Spec.Template.ObjectMeta.Labels, "logVerbosity")
					delete(testVM2.Spec.Template.ObjectMeta.Labels, "logVerbosity")
				},
				[]string{
					vmName1 + " = 5",
					vmName1 + ": updated verbosity 2 will be applied after the VM is restarted",
					vmName2 + " = 6",
					vmName2 + ": updated verbosity 2 will be applied after the VM is restarted",
				},
				true,
			),
			Entry("show after restarting VM",
				func() {
					// restart VMs (start VMs with no label)
				},
				[]string{vmName1 + " = 2", "", vmName2 + " = 2", ""},
				false,
			),
		)

		DescribeTable("reset 1 VM and set 2 VMs",
			func(vm1Label, vm2Label, resetLabel1, resetLabel2 string, vmOutput []string) {
				// start VMs with label (testvm1=vm1Label, testvm2=vm2Label)
				testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = vm1Label
				testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = vm2Label
				configureAndStartVMs()
				if resetLabel1 != "" && resetLabel2 != "" && vmOutput != nil {
					// reset and set labels
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = resetLabel1
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = resetLabel2
				}

				if vmOutput == nil {
					// for reset test
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity",
						"--reset="+vmName1, "--reset="+vmName2,
						"--vm="+vmName1, "--level="+resetLabel1,
						"--vm="+vmName2, "--level="+resetLabel2,
					)
					Expect(cmd()).To(Succeed())
					Expect(patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal(resetLabel1))
					Expect(patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal(resetLabel2))
				} else {
					// for show test
					args := []string{"--vm=" + vmName1, "--vm=" + vmName2}
					compOutput := []string{logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag, logverbosity.NoFlag}
					commonVMShowTest(compOutput, vmOutput, args...)
				}
			},
			Entry("show 2 VMs before reset", "5", "6", "", "", []string{vmName1 + " = 5", "", vmName2 + " = 6", ""}),
			Entry("reset and set the logVerbosity label", "5", "6", "7", "8", nil),
			Entry("show before restart VMs", "5", "6", "7", "8",
				[]string{
					vmName1 + " = 5",
					vmName1 + ": updated verbosity 7 will be applied after the VM is restarted",
					vmName2 + " = 6",
					vmName2 + ": updated verbosity 8 will be applied after the VM is restarted",
				},
			),
			Entry("show after restarting VMs", "7", "8", "", "", []string{vmName1 + " = 7", "", vmName2 + " = 8", ""}),
		)
	})

	When("coexistence of virt components and vm", func() {
		BeforeEach(func() {
			// virt-launcher=default (no DeveloperConfiguration field in the KubeVirt CR)
			commonSetup(kvInterface, kvs)
		})

		DescribeTable("set 1 virt component and 1 VM",
			func(testCase func(), virtOutput, vmOutput []string, virtSetting bool) {
				if virtSetting {
					// set virt-handler=3
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtHandler: 3,
						},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				}
				configureAndStartVMs()
				testCase()
				if virtOutput != nil || vmOutput != nil {
					args := []string{"--virt-handler", "--vm=" + vmName2}
					commonVMShowTest(virtOutput, vmOutput, args...)
				}
			},
			Entry("show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{logverbosity.NoFlag, logverbosity.NoFlag, "2", logverbosity.NoFlag, logverbosity.NoFlag},
				[]string{vmName2 + " = 2", ""},
				false,
			),
			Entry("reset logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--virt-handler=3", "--vm="+vmName2, "--level=6")
					Expect(cmd()).To(Succeed())
					output := []uint{0, 0, 3, 0, 0}
					expectAllComponentVerbosity(kv, output)
					Expect(patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("6"))
				},
				nil,
				nil,
				false,
			),
			Entry("show before restarting VM",
				func() {
					// set the logVerbosity label
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "6"
				},
				[]string{logverbosity.NoFlag, logverbosity.NoFlag, "3", logverbosity.NoFlag, logverbosity.NoFlag},
				[]string{
					vmName2 + " = 2",
					vmName2 + ": updated verbosity 6 will be applied after the VM is restarted",
				},
				true, // set virt component
			),
			Entry("show after restarting VM",
				func() {
					// restart VMs
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "6"
					configureAndStartVMs()
				},
				[]string{logverbosity.NoFlag, logverbosity.NoFlag, "3", logverbosity.NoFlag, logverbosity.NoFlag},
				[]string{vmName2 + " = 6", ""},
				true, // set virt component
			),
		)

		DescribeTable("set all virt components and 1 VM",
			func(testCase func(), virtOutput, vmOutput []string, virtSetting bool) {
				if virtSetting {
					// set all=3
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtAPI:        3,
							VirtController: 3,
							VirtHandler:    3,
							VirtLauncher:   3,
							VirtOperator:   3,
						},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				}
				configureAndStartVMs()
				testCase()
				if virtOutput != nil || vmOutput != nil {
					args := []string{"--all", "--vm=" + vmName1}
					commonVMShowTest(virtOutput, vmOutput, args...)
				}
			},
			Entry("show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{"2", "2", "2", "2", "2"},
				[]string{vmName1 + " = 2", ""},
				false,
			),
			Entry("reset logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--all=3", "--vm="+vmName1, "--level=7")
					Expect(cmd()).To(Succeed())
					output := []uint{3, 3, 3, 3, 3}
					expectAllComponentVerbosity(kv, output)
					Expect(patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("7"))
				},
				nil,
				nil,
				false,
			),
			Entry("show before restarting VM",
				func() {
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "7"
				},
				[]string{"3", "3", "3", "3", "3"},
				[]string{
					vmName1 + " = 2",
					vmName1 + ": updated verbosity 7 will be applied after the VM is restarted",
				},
				true, // set virt component
			),
			Entry("show after restarting VM",
				func() {
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "7"
					configureAndStartVMs()
				},
				[]string{"3", "3", "3", "3", "3"},
				[]string{vmName1 + " = 7", ""},
				true, // set virt component
			),
		)

		DescribeTable("reset all virt components and reset 2 VMs",
			func(testCase func(), virtOutput, vmOutput []string, preVirtSetting, preLabelSetting bool) {
				if preVirtSetting {
					// set all=3
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtAPI:        3,
							VirtController: 3,
							VirtHandler:    3,
							VirtLauncher:   3,
							VirtOperator:   3,
						},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				} else {
					// reset all components
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				}
				if preLabelSetting {
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "5"
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "6"
				}
				configureAndStartVMs()
				testCase()
				if virtOutput != nil || vmOutput != nil {
					args := []string{"--all", "--vm=" + vmName1, "--vm=" + vmName2}
					commonVMShowTest(virtOutput, vmOutput, args...)
				}
			},
			Entry("show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{"3", "3", "3", "3", "3"},
				[]string{vmName1 + " = 5", "", vmName2 + " = 6", ""},
				true, // has virt component verbosity before reset
				true, // label set before reset
			),
			Entry("reset logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--reset", "--reset="+vmName1, "--reset="+vmName2)
					Expect(cmd()).To(Succeed())
					output := []uint{0, 0, 0, 0, 0}
					expectAllComponentVerbosity(kv, output)
					Expect(cmd()).To(Succeed())

					_, exist := patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]
					Expect(exist).To(BeFalse())
					_, exist = patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]
					Expect(exist).To(BeFalse())
				},
				nil,
				nil,
				true,
				true,
			),
			Entry("show before restarting VM",
				func() {
					// reset VM1 and VM2
					delete(testVM1.Spec.Template.ObjectMeta.Labels, "logVerbosity")
					delete(testVM2.Spec.Template.ObjectMeta.Labels, "logVerbosity")
				},
				[]string{"2", "2", "2", "2", "2"},
				[]string{
					vmName1 + " = 5",
					vmName1 + ": updated verbosity 2 will be applied after the VM is restarted",
					vmName2 + " = 6",
					vmName2 + ": updated verbosity 2 will be applied after the VM is restarted",
				},
				false,
				true,
			),
			Entry("show after restarting VM",
				func() {
					// restart virt components and VMs (start vms no label, virt-launcher=empty (default))
				},
				[]string{"2", "2", "2", "2", "2"},
				[]string{vmName1 + " = 2", "", vmName2 + " = 2", ""},
				false,
				false,
			),
		)

		DescribeTable("reset all virt components + set 1 vrit component + reset 1 VM + set 1 VM",
			func(testCase func(), virtOutput, vmOutput []string, preVirtSetting, preLabelSetting bool) {
				if preVirtSetting {
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtAPI:        3,
							VirtController: 3,
							VirtHandler:    3,
							VirtLauncher:   3,
							VirtOperator:   3,
						},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				} else {
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtHandler: 3,
						},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				}
				if preLabelSetting {
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "5"
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "6"
				}
				configureAndStartVMs()
				testCase()
				if virtOutput != nil || vmOutput != nil {
					args := []string{"--all", "--vm=" + vmName1, "--vm=" + vmName2}
					commonVMShowTest(virtOutput, vmOutput, args...)
				}
			},
			// "show before setting label" test
			// same as the test "reset all virt components and reset 2 VMs"
			Entry("reset logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand(
						"adm", "log-verbosity",
						"--reset", "--virt-handler=3", "--reset="+vmName1, "--vm="+vmName2, "--level=7",
					)
					Expect(cmd()).To(Succeed())
					output := []uint{0, 0, 3, 0, 0}
					expectAllComponentVerbosity(kv, output)
					Expect(cmd()).To(Succeed())

					_, exist := patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]
					Expect(exist).To(BeFalse())
					Expect(patchedVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"]).To(Equal("7"))
				},
				nil, nil, true, true,
			),
			Entry("show before restarting VM",
				func() {
					// reset VM1 and set VM2
					delete(testVM1.Spec.Template.ObjectMeta.Labels, "logVerbosity")
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "7"
				},
				[]string{"2", "2", "3", "2", "2"},
				[]string{
					vmName1 + " = 5",
					vmName1 + ": updated verbosity 2 will be applied after the VM is restarted",
					vmName2 + " = 6",
					vmName2 + ": updated verbosity 7 will be applied after the VM is restarted",
				},
				false,
				true,
			),
			Entry("show after restarting VM",
				func() {
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "7"
					configureAndStartVMs()
				},
				[]string{"2", "2", "3", "2", "2"},
				[]string{vmName1 + " = 2", "", vmName2 + " = 7", ""},
				false,
				false,
			),
		)

		DescribeTable("virt-launcher=3, reset 1 VM",
			func(testCase func(), virtOutput, vmOutput []string, preVirtSetting, preLabelSetting bool) {
				if !preVirtSetting {
					dc := &v1.DeveloperConfiguration{
						LogVerbosity: &v1.LogVerbosity{
							VirtLauncher: 3,
						},
					}
					kvs.Items[0].Spec.Configuration.DeveloperConfiguration = dc
				}
				if preLabelSetting {
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "5"
				}
				configureAndStartVMs()
				testCase()
				if virtOutput != nil || vmOutput != nil {
					args := []string{"--all", "--vm=" + vmName1, "--vm=" + vmName2}
					commonVMShowTest(virtOutput, vmOutput, args...)
				}
			},
			Entry("show before setting label",
				func() {
					// No additional setup needed for this case
				},
				[]string{"2", "2", "2", "2", "2"},
				[]string{vmName1 + " = 5", "", vmName2 + " = 2", ""},
				true, // has virt component verbosity before reset
				true, // label set before reset
			),
			Entry("reset logVerbosity label",
				func() {
					expectedPatch(testVM1, testVM2)

					cmd := clientcmd.NewRepeatableVirtctlCommand("adm", "log-verbosity", "--virt-launcher=3", "--reset="+vmName1)
					Expect(cmd()).To(Succeed())
					output := []uint{0, 0, 0, 3, 0}
					expectAllComponentVerbosity(kv, output)
					Expect(cmd()).To(Succeed())

					_, exist := patchedVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"]
					Expect(exist).To(BeFalse())
				},
				nil, nil, true, true,
			),
			Entry("show before restarting VM",
				func() {
					// reset VM1
					delete(testVM1.Spec.Template.ObjectMeta.Labels, "logVerbosity")
				},
				[]string{"2", "2", "2", "3", "2"},
				[]string{
					vmName1 + " = 5",
					vmName1 + ": updated verbosity 3 will be applied after the VM is restarted",
					vmName2 + " = 2",
					vmName2 + ": updated verbosity 3 will be applied after the VM is restarted",
				},
				false,
				true,
			),
			Entry("show after restarting VM",
				func() {
					testVM1.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "3"
					testVM2.Spec.Template.ObjectMeta.Labels["logVerbosity"] = "3"
					configureAndStartVMs()
				},
				[]string{"2", "2", "2", "3", "2"},
				[]string{vmName1 + " = 3", "", vmName2 + " = 3", ""},
				false,
				false,
			),
		)
	})
})

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
	kvInterface.EXPECT().List(gomock.Any()).Return(kvs, nil).AnyTimes()
	kvInterface.EXPECT().Get(kvs.Items[0].Name, gomock.Any()).Return(&kvs.Items[0], nil).AnyTimes()
}

func createOutputMessage(output []string) []string {
	lines := []string{}
	var components = []string{"virt-api", "virt-controller", "virt-handler", "virt-launcher", "virt-operator"}
	for component := 0; component < len(components); component++ {
		if output[component] == logverbosity.NoFlag {
			continue
		}
		// output format is [componentName] =　[verbosity] like:
		// 		virt-api = 1
		// 		virt-controller = 2
		componentName := components[component]
		verbosity := output[component]
		line := fmt.Sprintf("%s = %s", componentName, verbosity)
		lines = append(lines, line)
	}
	return lines
}

func createVMOutputMessage(output []string, lines []string) []string {
	for _, line := range output {
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func commonShowTest(compOutput, vmOutput []string, args ...string) {
	commandAndArgs := []string{"adm", "log-verbosity"}
	commandAndArgs = append(commandAndArgs, args...)
	bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(commandAndArgs...)()
	Expect(err).To(Succeed())

	lines := createOutputMessage(compOutput)
	lines = createVMOutputMessage(vmOutput, lines)
	message := strings.Join(lines, "\n")

	Expect(string(bytes)).To(ContainSubstring(message))
}

func commonCompShowTest(compOutput []string, args ...string) {
	commonShowTest(compOutput, nil, args...)
}

func commonVMShowTest(compOutput, vmOutput []string, args ...string) {
	commonShowTest(compOutput, vmOutput, args...)
}

func expectAllComponentVerbosity(kv *v1.KubeVirt, output []uint) {
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtAPI).To(Equal(output[0]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtController).To(Equal(output[1]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtHandler).To(Equal(output[2]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtLauncher).To(Equal(output[3]))
	Expect(kv.Spec.Configuration.DeveloperConfiguration.LogVerbosity.VirtOperator).To(Equal(output[4]))
}

func commonSetCommand(args ...string) {
	commandAndArgs := []string{"adm", "log-verbosity"}
	commandAndArgs = append(commandAndArgs, args...)
	cmd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
	Expect(cmd()).To(Succeed())
}

func newVMNoVerbosityLabel(name, namespace string) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		TypeMeta: k8smetav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachine",
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: new(bool),
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: k8smetav1.ObjectMeta{
					Labels: map[string]string{
						v1.AppLabel:                "virt-launcher",
						v1.DomainAnnotation:        name,
						v1.VirtualMachineNameLabel: name,
					},
				},
			},
		},
	}
}

func newPodBase(name, namespace string, verbosityVar ...corev1.EnvVar) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: k8smetav1.ObjectMeta{
			GenerateName: name,
			Namespace:    namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "compute",
					Env:  verbosityVar,
				},
				{
					Name: "volumecontainerdisk",
				},
				{
					Name: "guest-console-log",
				},
			},
		},
	}
}

func newPodNoVerbosity(name, namespace string) *corev1.Pod {
	return newPodBase(name, namespace)
}

func newPodWithVerbosity(name, namespace, verbosity string) *corev1.Pod {
	verbosityVar := []corev1.EnvVar{
		{
			Name:  "VIRT_LAUNCHER_LOG_VERBOSITY",
			Value: verbosity,
		},
	}
	return newPodBase(name, namespace, verbosityVar...)
}

func newPodList(pods ...corev1.Pod) *corev1.PodList {
	return &corev1.PodList{
		TypeMeta: k8smetav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "List",
		},
		Items: pods,
	}
}

func createVirtLauncherPod(name, namespace, vmName, verbosity string) *corev1.Pod {
	if verbosity != "" {
		return newPodWithVerbosity(name, namespace, verbosity)
	}
	return newPodNoVerbosity(name, namespace)
}

func getVerbosityFromLabels(vm *v1.VirtualMachine) string {
	if verbosity, exist := vm.Spec.Template.ObjectMeta.Labels["logVerbosity"]; exist {
		return verbosity
	}
	return ""
}

func applyVMPatch(patchData []byte, target *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	patch, err := jsonpatch.DecodePatch(patchData)
	Expect(err).ToNot(HaveOccurred())

	vmJSON, err := json.Marshal(target)
	Expect(err).ToNot(HaveOccurred())

	modifiedVMJSON, err := patch.Apply(vmJSON)
	Expect(err).ToNot(HaveOccurred())

	target = &v1.VirtualMachine{}
	err = json.Unmarshal(modifiedVMJSON, target)
	Expect(err).ToNot(HaveOccurred())

	return target, nil
}
