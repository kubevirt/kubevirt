package cgroup

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/devices"

	runc_configs "github.com/opencontainers/runc/libcontainer/configs"

	"kubevirt.io/client-go/log"
)

const (
	cgroupStr = "cgroup"

	procMountPoint = "/proc"

	HostRootPath       = "/proc/1/root"
	cgroupBasePath     = "/sys/fs/" + cgroupStr
	HostCgroupBasePath = HostRootPath + cgroupBasePath
)

// Templates for logging / error messages
const (
	errApplyingDeviceRule     = "error occurred while applying device rule: %v"
	errApplyingNonDeviceRules = "error occurred while applying non-device rules: %v"
	settingDeviceRule         = "setting device rule for cgroup %s: %v"

	v1Str = "v1"
	v2Str = "v2"

	loggingVerbosity = 2
)

// getNewResourcesWithoutDevices returns a new Resources struct with Devices attributes dropped
func getNewResourcesWithoutDevices(r *runc_configs.Resources) runc_configs.Resources {
	resourcesWithoutDevices := *r
	resourcesWithoutDevices.Devices = nil

	return resourcesWithoutDevices
}

func logAndReturnErrorWithSprintfIfNotNil(err error, template string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	newErr := fmt.Errorf(template, args...)
	log.Log.Error(newErr.Error())
	return newErr
}

func areResourcesEmpty(r runc_configs.Resources) bool {
	return reflect.DeepEqual(r, runc_configs.Resources{})
}

// RunWithChroot changes the root directory (via "chroot") into newPath, then
// runs toRun function. When the function finishes, changes back the root directory
// to the original one that
func RunWithChroot(newPath string, toRun func() error) error { // ihol3 bad place to define func
	// Ensure no other goroutines are effected by this
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	originalRoot, err := os.Open("/")
	if err != nil {
		return fmt.Errorf("failed to run with chroot - failed to open root directory. error: %v", err)
	}

	err = syscall.Chroot(newPath)
	if err != nil {
		return fmt.Errorf("failed to chroot into \"%s\". error: %v", newPath, err)
	}

	changeRootToOriginal := func() {
		const errFormat = "cannot change root to original path. %s error: %+v"

		err = originalRoot.Chdir()
		if err != nil {
			log.Log.Errorf(errFormat, "chdir", err)
		}

		err = syscall.Chroot(".")
		if err != nil {
			log.Log.Errorf(errFormat, "chroot", err)
		}
	}
	defer changeRootToOriginal()

	err = toRun()
	return err
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
// Than we expect rule (1) to stay unchanged.
func addCurrentRules(currentRules, newRules []*devices.Rule) ([]*devices.Rule, error) {
	if currentRules == nil {
		return newRules, nil
	}
	if newRules == nil {
		return nil, fmt.Errorf("new rules cannot be nil")
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
		if !isCurrentRulePartOfRequestedRules(currentRule, newRules) {
			newRules = append(newRules, currentRule)
		}
	}

	return newRules, nil
}
