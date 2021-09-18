package templates

import (
	"errors"
	"strings"
)

// ParseTarget argument supporting the form of vmi/name.namespace (or simpler)
func ParseTarget(arg string) (kind string, namespace string, name string, err error) {
	kind = "vmi"

	kinds := strings.Split(arg, "/")
	if len(kinds) > 1 {
		kind = kinds[0]
		if !KindIsVM(kind) && !KindIsVMI(kind) {
			return "", "", "", errors.New("unsupported resource kind " + kind)
		}
		arg = kinds[1]
	}

	if len(arg) < 1 {
		return "", "", "", errors.New("expected name after '/'")
	}
	if arg[0] == '.' {
		return "", "", "", errors.New("expected name before '.'")
	}
	if arg[len(arg)-1] == '.' {
		return "", "", "", errors.New("expected namespace after '.'")
	}

	parts := strings.FieldsFunc(arg, func(r rune) bool {
		return r == '.'
	})

	name = parts[0]

	if len(parts) > 1 {
		namespace = parts[1]
	}

	return kind, namespace, name, nil
}

// ParseSSHTarget argument supporting the form of username@vmi/name.namespace (or simpler)
func ParseSSHTarget(arg string) (kind, namespace, name, username string, err error) {
	kind = "vmi"

	usernameAndTarget := strings.Split(arg, "@")
	if len(usernameAndTarget) > 1 {
		username = usernameAndTarget[0]
		if len(username) < 1 {
			return "", "", "", "", errors.New("expected username before '@'")
		}
		arg = usernameAndTarget[1]
	}

	if len(arg) < 1 {
		return "", "", "", "", errors.New("expected target after '@'")
	}

	kind, namespace, name, err = ParseTarget(arg)
	return kind, namespace, name, username, err
}

// KindIsVMI helps validating input parameters for specifying the VMI resource
func KindIsVMI(kind string) bool {
	return kind == "vmi" ||
		kind == "vmis" ||
		kind == "virtualmachineinstance" ||
		kind == "virtualmachineinstances"
}

// KindIsVM helps validating input parameters for specifying the VM resource
func KindIsVM(kind string) bool {
	return kind == "vm" ||
		kind == "vms" ||
		kind == "virtualmachine" ||
		kind == "virtualmachines"
}
