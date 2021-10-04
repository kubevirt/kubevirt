package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	runcFs2 "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	mntNamespace string
	cpuTime      uint64
	memoryBytes  uint64
	targetUser   string
)

func init() {
	// main needs to be locked on one thread and no go routines
	runtime.LockOSThread()
}

func main() {

	rootCmd := &cobra.Command{
		Use: "virt-chroot",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			if mntNamespace != "" {
				// join the mount namespace of a process
				fd, err := os.Open(mntNamespace)
				if err != nil {
					return fmt.Errorf("failed to open mount namespace: %v", err)
				}
				defer fd.Close()

				if err = unix.Unshare(unix.CLONE_NEWNS); err != nil {
					return fmt.Errorf("failed to detach from parent mount namespace: %v", err)
				}
				if err := unix.Setns(int(fd.Fd()), unix.CLONE_NEWNS); err != nil {
					return fmt.Errorf("failed to join the mount namespace: %v", err)
				}
			}

			// Looking up users needs resources, let's do it before we set rlimits.
			var u *user.User
			if targetUser != "" {
				var err error
				u, err = user.Lookup(targetUser)
				if err != nil {
					return fmt.Errorf("failed to look up user: %v", err)
				}
			}

			if cpuTime > 0 {
				value := &syscall.Rlimit{
					Cur: cpuTime,
					Max: cpuTime,
				}
				err := syscall.Setrlimit(unix.RLIMIT_CPU, value)
				if err != nil {
					return fmt.Errorf("error setting prlimit on cpu time with value %d: %v", value, err)
				}
			}

			if memoryBytes > 0 {
				value := &syscall.Rlimit{
					Cur: memoryBytes,
					Max: memoryBytes,
				}
				err := syscall.Setrlimit(unix.RLIMIT_AS, value)
				if err != nil {
					return fmt.Errorf("error setting prlimit on virtual memory with value %d: %v", value, err)
				}
			}

			// Now let's switch users and drop privileges
			if u != nil {
				uid, err := strconv.ParseInt(u.Uid, 10, 32)
				if err != nil {
					return fmt.Errorf("failed to parse uid: %v", err)
				}
				gid, err := strconv.ParseInt(u.Gid, 10, 32)
				if err != nil {
					return fmt.Errorf("failed to parse gid: %v", err)
				}
				err = unix.Setgroups([]int{int(gid)})
				if err != nil {
					return fmt.Errorf("failed to drop auxiliary groups: %v", err)
				}
				_, _, errno := syscall.Syscall(syscall.SYS_SETGID, uintptr(gid), 0, 0)
				if errno != 0 {
					return fmt.Errorf("failed to join the group of the user: %v", err)
				}
				_, _, errno = syscall.Syscall(syscall.SYS_SETUID, uintptr(uid), 0, 0)
				if errno != 0 {
					return fmt.Errorf("failed to switch to user: %v", err)
				}
			}
			return nil

		},
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}

	rootCmd.PersistentFlags().Uint64Var(&cpuTime, "cpu", 0, "cpu time in seconds for the process")
	rootCmd.PersistentFlags().Uint64Var(&memoryBytes, "memory", 0, "memory in bytes for the process")
	rootCmd.PersistentFlags().StringVar(&mntNamespace, "mount", "", "mount namespace to use")
	rootCmd.PersistentFlags().StringVar(&targetUser, "user", "", "switch to this targetUser to e.g. drop privileges")

	execCmd := &cobra.Command{
		Use:   "exec",
		Short: "execute a sandboxed command in a specific mount namespace",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := syscall.Exec(args[0], args, os.Environ())
			if err != nil {
				return fmt.Errorf("failed to execute command: %v", err)
			}
			return nil
		},
	}

	mntCmd := &cobra.Command{
		Use:   "mount",
		Short: "mount operations in a specific mount namespace",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var mntOpts uint = 0

			fsType := cmd.Flag("type").Value.String()
			mntOptions := cmd.Flag("options").Value.String()
			for _, opt := range strings.Split(mntOptions, ",") {
				opt = strings.TrimSpace(opt)
				switch opt {
				case "ro":
					mntOpts = mntOpts | syscall.MS_RDONLY
				case "bind":
					mntOpts = mntOpts | syscall.MS_BIND
				default:
					return fmt.Errorf("mount option %s is not supported", opt)
				}
			}

			return syscall.Mount(args[0], args[1], fsType, uintptr(mntOpts), "")
		},
	}
	mntCmd.Flags().StringP("options", "o", "", "comma separated list of mount options")
	mntCmd.Flags().StringP("type", "t", "", "fstype")

	umntCmd := &cobra.Command{
		Use:   "umount",
		Short: "unmount in a specific mount namespace",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return syscall.Unmount(args[0], 0)
		},
	}

	selinuxCmd := &cobra.Command{
		Use:   "selinux",
		Short: "run selinux operations in specific namespaces",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}

	selinuxCmd.AddCommand(
		NewGetEnforceCommand(), RelabelCommand(),
	)

	createTapCmd := NewCreateTapCommand()
	createTapCmd.Flags().String("tap-name", "tap0", "the name of the tap device")
	createTapCmd.Flags().Uint("uid", 0, "the owner of the tap device")
	createTapCmd.Flags().Uint("gid", 0, "the group of the owner of the tap device")
	createTapCmd.Flags().Uint32("queue-number", 0, "the number of queues to use on multi-queued devices")
	createTapCmd.Flags().Uint32("mtu", 1500, "the link MTU of the tap device")

	createMDEVCmd := NewCreateMDEVCommand()
	createMDEVCmd.Flags().String("type", "", "the type of a mediated device")
	createMDEVCmd.Flags().String("parent", "", "id of a parent (e.g. PCI_ID) for the new mediated device")
	createMDEVCmd.Flags().String("uuid", "", "uuid for the new mediated device")

	removeMDEVCmd := NewRemoveMDEVCommand()
	removeMDEVCmd.Flags().String("uuid", "", "uuid of the mediated device to remove")

	cgroupsV2DeviceCmd := &cobra.Command{
		Use:   "set-cgroupsv2-devices",
		Short: "Set cgroups v2 device rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			vmiPidFromHostView, err := strconv.ParseUint(cmd.Flag("pid").Value.String(), 10, 32)
			if err != nil {
				return fmt.Errorf("cannot convert PID into uint32. err: %v", err)
			}
			if vmiPidFromHostView == 0 {
				return fmt.Errorf("\"pid\" argument must be greater than zero")
			}

			cgroupDirPath := cmd.Flag("path").Value.String()
			if cgroupDirPath == "" {
				return fmt.Errorf("path argument cannot be empty")
			}

			marshalledRulesHash := cmd.Flag("rules").Value.String()
			isRootless, err := strconv.ParseBool(cmd.Flag("rootless").Value.String())
			if err != nil {
				return fmt.Errorf("cannot convert rootless into bool. err: %v", err)
			}

			unmarshalledRules, err := decodeDeviceRules(marshalledRulesHash)
			if err != nil {
				return err
			}
			if err = setCgroupDeviceRules(vmiPidFromHostView, cgroupDirPath, unmarshalledRules, isRootless); err != nil {
				return err
			}

			return nil
		},
	}

	cgroupsV2DeviceCmd.Flags().Uint32("pid", 0, "VMI's PID from the host's viewpoint")
	cgroupsV2DeviceCmd.Flags().String("path", "", "path to cgroups v2 directory") // ihol3 example
	cgroupsV2DeviceCmd.Flags().String("rules", "", "marshalled []*Rule type (defined in github.com/opencontainers/runc/libcontainer/devices), encoded to base64 format")
	cgroupsV2DeviceCmd.Flags().Bool("rootless", false, "true to run rootless")

	rootCmd.AddCommand(
		execCmd,
		mntCmd,
		umntCmd,
		selinuxCmd,
		createTapCmd,
		createMDEVCmd,
		removeMDEVCmd,
		cgroupsV2DeviceCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func decodeDeviceRules(marshalledRulesHash string) (unmarshalledRules []*devices.Rule, err error) {
	marshalledRules, err := base64.StdEncoding.DecodeString(marshalledRulesHash)
	if err != err {
		return nil, fmt.Errorf("cannot decode marshalled cgroups v2 rules. "+
			"encoded rules: %s. err: %v", marshalledRulesHash, err)
	}

	err = json.Unmarshal(marshalledRules, &unmarshalledRules)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall cgroup v2 rules. "+
			"marshalled rules: %s. err: %v", marshalledRules, err)
	}

	return unmarshalledRules, err
}

func setCgroupDeviceRules(vmiPidFromHostView uint64, cgroupDirPath string, deviceRules []*devices.Rule, isRootless bool) error {
	if deviceRules == nil || len(deviceRules) == 0 {
		return nil
	}

	config := &configs.Cgroup{
		Path:      cgroup.HostCgroupBasePath,
		Paths:     map[string]string{"": cgroupDirPath},
		Resources: &configs.Resources{},
	}

	cgroupV2Manager, err := runcFs2.NewManager(config, cgroupDirPath, isRootless)
	if err != nil {
		return fmt.Errorf("cannot create cgroups v2 manager from pid %d. err: %v", vmiPidFromHostView, err)
	}

	err = cgroupV2Manager.Set(&configs.Resources{
		Devices: deviceRules,
	})
	if err != nil {
		return fmt.Errorf("cannot set device rules. err: %v", err)
	}

	return nil
}
