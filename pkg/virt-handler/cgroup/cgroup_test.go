package cgroup

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
)

var _ = Describe("cgroup manager", func() {

	var (
		ctrl *gomock.Controller
	)

	newMockManager := func(version CgroupVersion) (*mockManager, error) {
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

	areRulesEqual := func(rule1, rule2 *devices.Rule) bool {
		if rule1 == nil && rule2 == nil {
			return true
		}

		if rule1 == nil || rule2 == nil {
			return false
		}

		return rule1.Allow == rule2.Allow && rule1.Major == rule2.Major && rule1.Minor == rule2.Minor &&
			rule1.Type == rule2.Type && rule1.Permissions == rule2.Permissions
	}

	isRulesInRuleList := func(rule *devices.Rule, ruleList []*devices.Rule) bool {
		for _, ruleInList := range ruleList {
			if areRulesEqual(rule, ruleInList) {
				return true
			}
		}
		return false
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	DescribeTable("ensure that default rules are added", func(version CgroupVersion) {
		manager, err := newMockManager(version)
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule := newDeviceRule(123)

		err = manager.Set(newResourcesWithRule(fakeRule))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(isRulesInRuleList(fakeRule, manager.rulesDefined)).To(BeTrue(), "defined rule is expected to exist")

		for _, defaultRule := range GenerateDefaultDeviceRules() {
			Expect(isRulesInRuleList(defaultRule, manager.rulesDefined)).To(BeTrue(), "default rules are expected to be defined")
		}

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

		previousRuleExists := isRulesInRuleList(fakeRule1, manager.rulesDefined)
		Expect(previousRuleExists).To(BeTrue(), "previous rule is expected to not be overridden")

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
		Expect(isRulesInRuleList(fakeRule, manager.rulesDefined)).To(BeTrue(), "defined rule is expected to exist")

		fakeRule.Permissions = "fake-permissions-456"
		Expect(isRulesInRuleList(fakeRule, manager.rulesDefined)).To(BeTrue(), "rule needs to be overridden since explicitly re-set")

	},
		Entry("for v1", V1),
		Entry("for v2", V2),
	)

})

type mockManager struct {
	*MockManager
	rulesDefined []*devices.Rule
	realManager  *Manager
}

func newMockManagerFromCtrl(ctrl *gomock.Controller, version CgroupVersion) (*mockManager, error) {
	mockCgroupManager := NewMockManager(ctrl)

	mockV1 := &mockManager{ //ihol3 change name
		mockCgroupManager,
		make([]*devices.Rule, 0),
		nil,
	}

	execVirtChrootFunc := func(r *runc_configs.Resources, pid int, subsystemPaths map[string]string, rootless bool, version CgroupVersion) error {
		mockV1.rulesDefined = r.Devices
		return nil
	}
	getCurrentlyDefinedRulesFunc := func(runcManager runc_cgroups.Manager) ([]*devices.Rule, error) {
		return mockV1.rulesDefined, nil
	}

	var realManager Manager
	var err error

	if version == V1 {
		realManager, err = newCustomizedV1Manager(&runc_configs.Cgroup{}, nil, false, 123, execVirtChrootFunc, getCurrentlyDefinedRulesFunc)
	} else {
		realManager, err = newCustomizedV2Manager(&runc_configs.Cgroup{}, "fake/dir/path", false, 123, execVirtChrootFunc)
	}

	if err != nil {
		return mockV1, err
	}
	mockV1.realManager = &realManager

	mockCgroupManager.EXPECT().Set(gomock.Any()).DoAndReturn(realManager.Set).AnyTimes()
	mockCgroupManager.EXPECT().GetBasePathToHostSubsystem(gomock.Any()).DoAndReturn(realManager.GetBasePathToHostSubsystem).AnyTimes()
	mockCgroupManager.EXPECT().GetCgroupVersion().DoAndReturn(realManager.GetCgroupVersion).AnyTimes()
	mockCgroupManager.EXPECT().GetCpuSet().DoAndReturn(realManager.GetCpuSet).AnyTimes()

	return mockV1, nil
}
