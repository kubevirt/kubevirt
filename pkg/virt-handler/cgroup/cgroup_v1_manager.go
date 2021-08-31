package cgroup

import (
	"bytes"
	"fmt"
	"path/filepath"

	"kubevirt.io/client-go/log"

	cgroup_devices "github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/devices"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
)

type v1Manager struct {
	runc_cgroups.Manager
}

func newV1Manager(config *runc_configs.Cgroup, controllerPaths map[string]string, rootless bool) (Manager, error) {
	runcManager := runc_fs.NewManager(config, controllerPaths, rootless)
	manager := v1Manager{
		runcManager,
	}

	return &manager, nil
}

func (v *v1Manager) GetBasePathToHostSubsystem(subsystem string) (string, error) {
	if v.Path(subsystem) == "" {
		return "", fmt.Errorf("controller %s does not exist", subsystem)
	}
	return filepath.Join(HostCgroupBasePath, subsystem), nil
}

func (v *v1Manager) Set(r *runc_configs.Resources) error {
	// We want to keep given resources untouched
	resourcesToSet := *r

	// Adding current rules, see addCurrentRules's documentation for more info
	requestedAndCurrentRules, err := v.addCurrentRules(r.Devices)
	if err != nil {
		return err
	}

	log.Log.V(loggingVerbosity).Infof("Adding current rules to requested for cgroup %s. Rules added: %d", v1Str, len(requestedAndCurrentRules)-len(r.Devices))
	resourcesToSet.Devices = requestedAndCurrentRules

	err = RunWithChroot(HostCgroupBasePath, func() error {
		err := v.Manager.Set(&resourcesToSet)
		return err
	})

	return logAndReturnErrorWithSprintfIfNotNil(err, errApplyingOtherRules, err)
}

func (v *v1Manager) GetCgroupVersion() string {
	return v1Str
}

// addCurrentRules gets a slice of rules as a parameter and returns a new slice that contains all given rules
// and all of the rules that are currently set. This way rules that are already defined won't be deleted by this
// current request. Every old rule that is part of the new request will be overridden.
//
// For example, if the following rules are defined:
// 1) {Minor: 111, Major: 111, Allow: true}
// 2) {Minor: 222, Major: 222, Allow: true}
//
// And we get a request to enable the following rule: {Minor: 222, Major: 222, Allow: false}
// Than we expect rule (1) to stay unchanged. Not adding it to our request will automatically delete it.
func (v *v1Manager) addCurrentRules(deviceRules []*devices.Rule) ([]*devices.Rule, error) {
	devicesPath, ok := v.GetPaths()["devices"]
	if !ok {
		return deviceRules, fmt.Errorf("devices subsystem's path is not defined for this manager")
	}
	devicesPath = filepath.Join(HostCgroupBasePath, devicesPath)

	currentRulesStr, err := runc_cgroups.ReadFile(devicesPath, "devices.list")
	if err != nil {
		return deviceRules, fmt.Errorf("error reading current rules: %v", err)
	}

	emulator, err := cgroup_devices.EmulatorFromList(bytes.NewBufferString(currentRulesStr))
	if err != nil {
		return deviceRules, fmt.Errorf("error creating emulator out of current rules: %v", err)
	}

	currentRules, err := emulator.Rules()
	if err != nil {
		return deviceRules, fmt.Errorf("error getting rules from emulator: %v", err)
	}

	isCurrentRulePartOfRequestedRules := func(rule *devices.Rule, rulesSlice []*devices.Rule) bool {
		for _, ruleInSlice := range rulesSlice {
			if rule.Type == ruleInSlice.Type && rule.Minor == ruleInSlice.Minor && rule.Major == ruleInSlice.Major {
				return true
			}
		}
		return false
	}

	for _, currentRule := range currentRules {
		if !isCurrentRulePartOfRequestedRules(currentRule, deviceRules) {
			deviceRules = append(deviceRules, currentRule)
		}
	}

	return deviceRules, nil
}
