package device_manager

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/uuid"
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

	Context("Handle mediated devices", func() {
		AfterEach(func() {
			os.RemoveAll(fakeMdevBasePath)
			ctrl.Finish()
		})
		table.DescribeTable("should create and remove relevant mdev types", func(scenario func() *scenarioValues) {
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
			table.Entry("spread types accoss identical cards", spreadTypesAccossIdenticalCard),
			table.Entry("one yype many cards", oneTypeManyCards),
			table.Entry("many types many cards", multipleTypeOneCards),
			table.Entry("no cards support requeted types", noCardsSupportTypes),
		)
	})
})
