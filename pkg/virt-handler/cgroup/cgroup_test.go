package cgroup

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
)

var _ = Describe("cgroup manager", func() {

	var (
		ctrl         *gomock.Controller
		rulesDefined []*devices.Rule
	)

	newMockManagerFromCtrl := func(ctrl *gomock.Controller, version CgroupVersion) (Manager, error) {
		mockRuncCgroupManager := NewMockruncManager(ctrl)
		mockRuncCgroupManager.EXPECT().GetPaths().DoAndReturn(func() map[string]string {
			paths := make(map[string]string)

			// See documentation here for more info: https://github.com/opencontainers/runc/blob/release-1.0/libcontainer/cgroups/cgroups.go#L48
			if version == V1 {
				paths["devices"] = "/sys/fs/cgroup/devices"
			} else {
				paths[""] = "/sys/fs/cgroup/"
			}

			return paths
		}).AnyTimes()

		execVirtChrootFunc := func(r *runc_configs.Resources, subsystemPaths map[string]string, rootless bool, version CgroupVersion) error {
			rulesDefined = r.Devices
			return nil
		}

		getCurrentlyDefinedRulesFunc := func(runcManager runc_cgroups.Manager) ([]*devices.Rule, error) {
			return rulesDefined, nil
		}

		if version == V1 {
			return newCustomizedV1Manager(mockRuncCgroupManager, false, execVirtChrootFunc, getCurrentlyDefinedRulesFunc)
		} else {
			return newCustomizedV2Manager(mockRuncCgroupManager, false, nil, execVirtChrootFunc)
		}
	}

	newMockManager := func(version CgroupVersion) (Manager, error) {
		return newMockManagerFromCtrl(ctrl, version)
	}

	newResourcesWithRule := func(rule *devices.Rule) *runc_configs.Resources {
		return &runc_configs.Resources{
			Devices: []*devices.Rule{
				rule,
			},
		}
	}

	newDeviceRule := func(UID int64) *devices.Rule {
		return &devices.Rule{
			Type:        'z',
			Major:       UID,
			Minor:       UID,
			Permissions: "fakePermissions",
			Allow:       true,
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		rulesDefined = make([]*devices.Rule, 0)
	})

	DescribeTable("ensure that default rules are added", func(version CgroupVersion) {
		manager, err := newMockManager(version)
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule := newDeviceRule(123)

		err = manager.Set(newResourcesWithRule(fakeRule))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(rulesDefined).To(ContainElement(fakeRule), "defined rule is expected to exist")

		defaultDeviceRules := GenerateDefaultDeviceRules()
		for _, defaultRule := range defaultDeviceRules {
			Expect(rulesDefined).To(ContainElement(defaultRule), "default rules are expected to be defined")
		}
		Expect(rulesDefined).To(HaveLen(len(defaultDeviceRules) + 1))
	},
		Entry("for v1", V1),
		Entry("for v2", V2),
	)

	DescribeTable("ensure that past rules are not overridden", func(version CgroupVersion) {
		manager, err := newMockManager(version)
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule1 := newDeviceRule(123)
		fakeRule2 := newDeviceRule(456)

		err = manager.Set(newResourcesWithRule(fakeRule1))
		Expect(err).ShouldNot(HaveOccurred())

		err = manager.Set(newResourcesWithRule(fakeRule2))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(rulesDefined).To(ContainElement(fakeRule1), "previous rule is expected to not be overridden")

	},
		Entry("for v1", V1),
		Entry("for v2", V2),
	)

	DescribeTable("ensure that past rules are overridden if explicitly set", func(version CgroupVersion) {
		manager, err := newMockManager(version)
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule := newDeviceRule(123)
		fakeRule.Permissions = "fake-permissions-123"

		err = manager.Set(newResourcesWithRule(fakeRule))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rulesDefined).To(ContainElement(fakeRule), "defined rule is expected to exist")

		fakeRule.Permissions = "fake-permissions-456"
		Expect(rulesDefined).To(ContainElement(fakeRule), "rule needs to be overridden since explicitly re-set")

	},
		Entry("for v1", V1),
		Entry("for v2", V2),
	)

})
