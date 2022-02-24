package device_manager

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Mediated Devices Types configuration", func() {
	var mockMDEV *MockDeviceHandler
	var ctrl *gomock.Controller

	type mdevTypesDetails struct {
		name               string
		availableInstances int
	}
	var fakeMdevBasePath string
	var fakeMdevDevicesPath string
	var configuredMdevTypesOnCards map[string]map[string]struct{}
	var clientTest *fake.Clientset
	var mdevTypesDetailsMap = map[string]mdevTypesDetails{
		"nvidia-222": mdevTypesDetails{
			name:               "GRID T4-1B",
			availableInstances: 16,
		},
		"nvidia-223": mdevTypesDetails{
			name:               "GRID T4-2B",
			availableInstances: 8,
		},
		"nvidia-224": mdevTypesDetails{
			name:               "GRID T4-2B4",
			availableInstances: 8,
		},
		"nvidia-228": mdevTypesDetails{
			name:               "GRID T4-8A",
			availableInstances: 2,
		},
		"nvidia-229": mdevTypesDetails{
			name:               "GRID T4-16A",
			availableInstances: 1,
		},
		"i915-GVTg_V5_1": mdevTypesDetails{
			availableInstances: 1,
		},
		"i915-GVTg_V5_2": mdevTypesDetails{
			availableInstances: 1,
		},
		"i915-GVTg_V5_4": mdevTypesDetails{
			availableInstances: 1,
		},
		"i915-GVTg_V5_8": mdevTypesDetails{
			availableInstances: 2,
		},
	}

	createTempMDEVSysfsStructure := func(pciMdevTypesMap map[string][]string) {
		// create an alternative mdev_supported_types dir instead of /sys/bus/mdev/devices/
		var err error
		fakeMdevDevicesPath, err = ioutil.TempDir("/tmp", "mdev")
		Expect(err).ToNot(HaveOccurred())
		mdevBasePath = fakeMdevDevicesPath
		// create an alternative mdev_supported_types dir instead of /sys/class/mdev_bus/[pciAddress]/
		fakeMdevBasePath, err = ioutil.TempDir("/tmp", "mdev_bus")
		Expect(err).ToNot(HaveOccurred())
		mdevClassBusPath = fakeMdevBasePath
		for pciAddr, mdevTypesForPciDevices := range pciMdevTypesMap {
			for _, mdevType := range mdevTypesForPciDevices {
				// create a fake path to mdev type for each card
				fakeNvidiaTypePath := filepath.Join(fakeMdevBasePath, pciAddr, "mdev_supported_types", mdevType)
				err = os.MkdirAll(fakeNvidiaTypePath, 0700)
				Expect(err).ToNot(HaveOccurred())

				// create a create file in the nvidia type directory
				_, err := os.Create(filepath.Join(fakeNvidiaTypePath, "create"))
				Expect(err).ToNot(HaveOccurred())

				if mdevNameContent := mdevTypesDetailsMap[mdevType].name; mdevNameContent != "" {
					// create a name file in the nvidia type directory
					mdevName, err := os.Create(filepath.Join(fakeNvidiaTypePath, "name"))
					Expect(err).ToNot(HaveOccurred())
					mdevNameWriter := bufio.NewWriter(mdevName)
					_, err = mdevNameWriter.WriteString(mdevNameContent + "\n")
					Expect(err).ToNot(HaveOccurred())
					mdevNameWriter.Flush()
				}

				// create available_instances
				// create a name file in the nvidia type directory
				mdevInstances, err := os.Create(filepath.Join(fakeNvidiaTypePath, "available_instances"))
				Expect(err).ToNot(HaveOccurred())
				mdevNameWriter := bufio.NewWriter(mdevInstances)
				mdevInstancesNum := mdevTypesDetailsMap[mdevType].availableInstances
				_, err = mdevNameWriter.WriteString(strconv.Itoa(mdevInstancesNum) + "\n")
				Expect(err).ToNot(HaveOccurred())
				mdevNameWriter.Flush()

			}
		}
	}

	countCreatedMdevs := func(mdevType string) int {
		i := 0
		files, err := ioutil.ReadDir(fakeMdevDevicesPath)
		Expect(err).ToNot(HaveOccurred())
		for _, file := range files {
			if file.IsDir() {
				linkTypePath, err := os.Readlink(filepath.Join(fakeMdevDevicesPath, file.Name(), "mdev_type"))
				Expect(err).ToNot(HaveOccurred())
				if filepath.Base(linkTypePath) == mdevType {
					i++
				}
			}
		}
		return i
	}

	BeforeEach(func() {
		By("mocking MDEV functions to simulate an mdev creation and removal")
		ctrl = gomock.NewController(GinkgoT())
		clientTest = fake.NewSimpleClientset()
		mockMDEV = NewMockDeviceHandler(ctrl)
		Handler = mockMDEV
		configuredMdevTypesOnCards = make(map[string]map[string]struct{})

		mockMDEV.EXPECT().CreateMDEVType(gomock.Any(), gomock.Any()).DoAndReturn(func(mdevType string, parentID string) error {
			mdevUUID := string(uuid.NewUUID())
			mdevUUIDDirPath := filepath.Join(fakeMdevDevicesPath, mdevUUID)
			err := os.MkdirAll(mdevUUIDDirPath, 0700)
			Expect(err).ToNot(HaveOccurred())
			mdevTypeDirPath := filepath.Join(fakeMdevBasePath, parentID, "mdev_supported_types", mdevType)
			err = os.Symlink(mdevTypeDirPath, filepath.Join(mdevUUIDDirPath, "mdev_type"))
			Expect(err).ToNot(HaveOccurred())
			parentsMap := configuredMdevTypesOnCards[mdevType]
			if parentsMap == nil {
				parentsMap = make(map[string]struct{})
			}
			parentsMap[parentID] = struct{}{}
			configuredMdevTypesOnCards[mdevType] = parentsMap
			return nil
		}).AnyTimes()

		mockMDEV.EXPECT().ReadMDEVAvailableInstances(gomock.Any(), gomock.Any()).DoAndReturn(func(mdevType string, parentID string) (int, error) {
			return mdevTypesDetailsMap[mdevType].availableInstances, nil
		}).AnyTimes()

		mockMDEV.EXPECT().RemoveMDEVType(gomock.Any()).DoAndReturn(func(mdevUUID string) error {
			mdevUUIDDirPath := filepath.Join(fakeMdevDevicesPath, mdevUUID)
			err := os.RemoveAll(mdevUUIDDirPath)
			Expect(err).ToNot(HaveOccurred())
			return nil
		}).AnyTimes()

	})
	AfterEach(func() {
		os.RemoveAll(fakeMdevBasePath)
	})

	type scenarioValues struct {
		pciMDEVDevicesMap       map[string][]string
		desiredDevicesList      []string
		expectedConfiguredTypes []string
		nodeLabels              map[string]string
	}

	spreadTypesAccossIdenticalCard := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
			"0000:66:00.0": mdevTypesForIdenticalPciDevices,
			"0000:67:00.0": mdevTypesForIdenticalPciDevices,
			"0000:00:02.0": []string{"i915-GVTg_V5_1", "i915-GVTg_V5_2", "i915-GVTg_V5_4", "i915-GVTg_V5_8"},
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-223", "nvidia-224", "nvidia-229", "i915-GVTg_V5_4"},
			expectedConfiguredTypes: []string{"nvidia-223", "nvidia-224", "nvidia-229", "i915-GVTg_V5_4"},
		}
	}
	oneTypeManyCards := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
			"0000:66:00.0": mdevTypesForIdenticalPciDevices,
			"0000:67:00.0": mdevTypesForIdenticalPciDevices,
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-223"},
			expectedConfiguredTypes: []string{"nvidia-223"},
		}
	}
	multipleTypeOneCards := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"},
			expectedConfiguredTypes: []string{"ANY"},
		}
	}
	noCardsSupportTypes := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
			"0000:66:00.0": mdevTypesForIdenticalPciDevices,
			"0000:67:00.0": mdevTypesForIdenticalPciDevices,
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"i915-GVTg_V5_1", "i915-GVTg_V5_2", "i915-GVTg_V5_4", "i915-GVTg_V5_8"},
			expectedConfiguredTypes: []string{},
		}
	}
	deafultTypesNotNodeSpecific := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
			"0000:66:00.0": mdevTypesForIdenticalPciDevices,
			"0000:00:02.0": []string{"i915-GVTg_V5_1", "i915-GVTg_V5_2", "i915-GVTg_V5_4", "i915-GVTg_V5_8"},
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-222", "nvidia-228", "i915-GVTg_V5_4"},
			expectedConfiguredTypes: []string{"nvidia-222", "nvidia-228", "i915-GVTg_V5_4"},
		}
	}
	matchAllNodeLabels := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-224"},
			expectedConfiguredTypes: []string{"nvidia-224"},
			nodeLabels:              map[string]string{"testLabel3": "true", "testLabel4": "true"},
		}
	}
	matchSingleNodeLabel := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-223"},
			expectedConfiguredTypes: []string{"nvidia-223"},
			nodeLabels:              map[string]string{"testLabel1": "true"},
		}
	}
	mergeAllTypesMatchedByNodeLabels := func() *scenarioValues {
		mdevTypesForIdenticalPciDevices := []string{"nvidia-222", "nvidia-223", "nvidia-224", "nvidia-228", "nvidia-229"}
		pciMDEVDevicesMap := map[string][]string{
			"0000:65:00.0": mdevTypesForIdenticalPciDevices,
			"0000:66:00.0": mdevTypesForIdenticalPciDevices,
		}
		return &scenarioValues{
			pciMDEVDevicesMap:       pciMDEVDevicesMap,
			desiredDevicesList:      []string{"nvidia-223", "nvidia-229"},
			expectedConfiguredTypes: []string{"nvidia-223", "nvidia-229"},
			nodeLabels:              map[string]string{"testLabel1": "true", "testLabel2": "true"},
		}
	}

	Context("Handle mediated devices", func() {
		AfterEach(func() {
			os.RemoveAll(fakeMdevBasePath)
		})
		DescribeTable("should create and remove relevant mdev types", func(scenario func() *scenarioValues) {
			sc := scenario()
			createTempMDEVSysfsStructure(sc.pciMDEVDevicesMap)
			mdevManager := NewMDEVTypesManager()
			mdevManager.updateMDEVTypesConfiguration(sc.desiredDevicesList)

			By("creating the desired mdev types")
			desiredDevicesToConfigure := make(map[string]struct{})
			for _, dev := range sc.expectedConfiguredTypes {
				desiredDevicesToConfigure[dev] = struct{}{}
			}
			By("making sure that a correct amount of mdevs is created for each type")
			// in cases where multiple mdev types are required to be configured but the amount of cards is significantly lower
			// it will be hard to estimate which of the requested types will be created. Simply check that amount of created types matches the avaiable cards.
			if len(sc.expectedConfiguredTypes) == 1 && sc.expectedConfiguredTypes[0] == "ANY" {
				Expect(len(configuredMdevTypesOnCards)).To(Equal(len(sc.pciMDEVDevicesMap)))
			} else {
				for mdevType, _ := range mdevTypesDetailsMap {
					numberOfCreatedMDEVs := countCreatedMdevs(mdevType)
					if _, exist := desiredDevicesToConfigure[mdevType]; exist {
						numberOfCardsConfiguredWithMdevType := len(configuredMdevTypesOnCards[mdevType])
						mdevInstancesNum := mdevTypesDetailsMap[mdevType].availableInstances * numberOfCardsConfiguredWithMdevType
						msg := fmt.Sprintf("created amount of mdevs for type %s doesn't match the expected", mdevType)
						Expect(numberOfCreatedMDEVs).To(Equal(mdevInstancesNum), msg)
						delete(desiredDevicesToConfigure, mdevType)
					} else {
						msg := fmt.Sprintf("there should not be any mdevs created for type %s", mdevType)
						Expect(numberOfCreatedMDEVs).To(BeZero(), msg)
					}
				}
				Expect(len(desiredDevicesToConfigure)).To(BeZero(), "add types should be created")
			}

			By("removing all created mdevs")
			mdevManager.updateMDEVTypesConfiguration([]string{})
			files, err := ioutil.ReadDir(fakeMdevDevicesPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(BeZero())
		},
			Entry("spread types accoss identical cards", spreadTypesAccossIdenticalCard),
			Entry("one yype many cards", oneTypeManyCards),
			Entry("many types many cards", multipleTypeOneCards),
			Entry("no cards support requeted types", noCardsSupportTypes),
		)
		DescribeTable("should create and remove relevant mdev types matching a specific node", func(scenario func() *scenarioValues) {
			sc := scenario()
			clientTest = fake.NewSimpleClientset()
			createTempMDEVSysfsStructure(sc.pciMDEVDevicesMap)
			By("creating a cluster config")
			kv := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{},
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeploying,
				},
			}
			fakeClusterConfig, _, kvInformer := testutils.NewFakeClusterConfigUsingKV(kv)
			kvConfig := kv.DeepCopy()
			kvConfig.Spec.Configuration.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{

					"nvidia-222",
					"nvidia-228",
					"i915-GVTg_V5_4",
				},
				NodeMediatedDeviceTypes: []v1.NodeMediatedDeviceTypesConfig{
					{
						NodeSelector: map[string]string{
							"testLabel1": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-223",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel2": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-229",
						},
					},
					{
						NodeSelector: map[string]string{
							"testLabel3": "true",
							"testLabel4": "true",
						},
						MediatedDevicesTypes: []string{
							"nvidia-224",
						},
					},
				},
			}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
			node := &kubev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testNode",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: v1.GroupVersion.String(),
				},
			}
			node.Status.Phase = kubev1.NodeRunning
			node.ObjectMeta.Labels = sc.nodeLabels
			addNode(clientTest, node)

			By("creating an empty device controller")
			var noDevices []Device
			deviceController := NewDeviceController("master", noDevices, fakeClusterConfig, clientTest.CoreV1())
			deviceController.refreshMediatedDevicesTypes()
			By("creating the desired mdev types")
			desiredDevicesToConfigure := make(map[string]struct{})
			for _, dev := range sc.desiredDevicesList {
				desiredDevicesToConfigure[dev] = struct{}{}
			}
			By("making sure that a correct amount of mdevs is created for each type")
			// in cases where multiple mdev types are required to be configured but the amount of cards is significantly lower
			// it will be hard to estimate which of the requested types will be created. Simply check that amount of created types matches the avaiable cards.
			if len(sc.expectedConfiguredTypes) == 1 && sc.expectedConfiguredTypes[0] == "ANY" {
				Expect(len(configuredMdevTypesOnCards)).To(Equal(len(sc.pciMDEVDevicesMap)))
			} else {
				for mdevType, _ := range mdevTypesDetailsMap {
					numberOfCreatedMDEVs := countCreatedMdevs(mdevType)
					if _, exist := desiredDevicesToConfigure[mdevType]; exist {
						numberOfCardsConfiguredWithMdevType := len(configuredMdevTypesOnCards[mdevType])
						mdevInstancesNum := mdevTypesDetailsMap[mdevType].availableInstances * numberOfCardsConfiguredWithMdevType
						msg := fmt.Sprintf("created amount of mdevs for type %s doesn't match the expected", mdevType)
						Expect(numberOfCreatedMDEVs).To(Equal(mdevInstancesNum), msg)
						delete(desiredDevicesToConfigure, mdevType)
					} else {
						msg := fmt.Sprintf("there should not be any mdevs created for type %s", mdevType)
						Expect(numberOfCreatedMDEVs).To(BeZero(), msg)
					}
				}
				Expect(len(desiredDevicesToConfigure)).To(BeZero(), "add types should be created")
			}

			By("removing all created mdevs")
			kvConfig.Spec.Configuration.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
			deviceController.refreshMediatedDevicesTypes()
			files, err := ioutil.ReadDir(fakeMdevDevicesPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(BeZero())
		},
			Entry("configure default mdev types", deafultTypesNotNodeSpecific),
			Entry("configure mdev types that match all node selectors", matchAllNodeLabels),
			Entry("configure mdev types that match a node selector", matchSingleNodeLabel),
			Entry("configure a merged list of mdev types when multiple selectors match node", mergeAllTypesMatchedByNodeLabels),
		)
	})
})

func addNode(client *fake.Clientset, node *kubev1.Node) {
	client.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
		return true, node, nil
	})
}
