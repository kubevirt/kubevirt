package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"
)

var (
	mntNamespace string
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
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s", cmd.UsageString())
		},
	}

	rootCmd.PersistentFlags().StringVar(&mntNamespace, "mount", "", "mount namespace to use")

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

			// Ensure that sourceFile is a real path. It will be kept open until used
			// by the syscall via the file descriptor path in proc (SafePath) to ensure
			// that no symlink injection can happen after the check.
			sourceFile, err := safepath.NewFileNoFollow(args[0])
			if err != nil {
				return fmt.Errorf("mount source invalid: %v", err)
			}
			defer sourceFile.Close()

			// Ensure that targetFile is a real path. It will be kept open until used
			// by the syscall via the file descriptor path in proc (SafePath) to ensure
			// that no symlink injection can happen after the check.
			targetFile, err := safepath.NewFileNoFollow(args[1])
			if err != nil {
				return fmt.Errorf("mount target invalid: %v", err)
			}
			defer targetFile.Close()

			return syscall.Mount(sourceFile.SafePath(), targetFile.SafePath(), fsType, uintptr(mntOpts), "")
		},
	}
	mntCmd.Flags().StringP("options", "o", "", "comma separated list of mount options")
	mntCmd.Flags().StringP("type", "t", "", "fstype")

	umntCmd := &cobra.Command{
		Use:   "umount",
		Short: "unmount in a specific mount namespace",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure that targetFile is a real path. It will be kept open until used
			// by the syscall via the file descriptor path in proc (SafePath) to ensure
			// that no symlink injection can happen after the check.
			targetFile, err := safepath.NewPathNoFollow(args[0])
			if err != nil {
				return fmt.Errorf("mount target invalid: %v", err)
			}
			err = targetFile.ExecuteNoFollow(func(safePath string) error {
				// we actively hold an open reference to the mount point,
				// we have to lazy unmount, to not block ourselves
				// with the active file-descriptor.
				return syscall.Unmount(safePath, unix.MNT_DETACH)
			})
			if err != nil {
				return fmt.Errorf("umount failed: %v", err)
			}
			return nil
		},
	}

	selinuxCmd := &cobra.Command{
		Use:   "selinux",
		Short: "run selinux operations in specific namespaces",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("%s", cmd.UsageString())
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

	cgroupsCmd := &cobra.Command{
		Use:   "set-cgroups-resources",
		Short: "Set cgroups resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			marshalledPathsHash := cmd.Flag("subsystem-paths").Value.String()
			if marshalledPathsHash == "" {
				return fmt.Errorf("path argument cannot be empty")
			}

			marshalledResourcesHash := cmd.Flag("resources").Value.String()
			isV2, err := strconv.ParseBool(cmd.Flag("isV2").Value.String())
			if err != nil {
				return fmt.Errorf("cannot convert isV2 into bool. err: %v", err)
			}

			unmarshalledResources, err := decodeResources(marshalledResourcesHash)
			if err != nil {
				return err
			}

			unmarshalledPaths, err := decodePaths(marshalledPathsHash)
			if err != nil {
				return err
			}

			if err = setCgroupResources(unmarshalledPaths, unmarshalledResources, isV2); err != nil {
				return err
			}

			return nil
		},
	}

	cgroupsCmd.Flags().String("subsystem-paths", "", "marshalled map[string]string type, encoded to base64 format. "+
		"For v1 key is cgroup subsystem and value is its path, for v2 the only key is an empty string and the value is cgroup dir path.")
	cgroupsCmd.Flags().String("resources", "", "marshalled Resources type (defined in github.com/opencontainers/cgroups/config_linux.go), encoded to base64 format")
	cgroupsCmd.Flags().Bool("isV2", false, "true for cgroups v2")

	updateDeviceCmd := &cobra.Command{
		Use:   "update-device",
		Short: "Allow or deny a device in the eBPF device map",
		RunE: func(cmd *cobra.Command, args []string) error {
			cgroupPath := cmd.Flag("cgroup-path").Value.String()
			if cgroupPath == "" {
				return fmt.Errorf("--cgroup-path is required")
			}

			devTypeStr := cmd.Flag("device-type").Value.String()
			devType, err := deviceTypeToU32(devTypeStr)
			if err != nil {
				return err
			}

			major, err := strconv.ParseUint(cmd.Flag("major").Value.String(), 10, 32)
			if err != nil {
				return fmt.Errorf("invalid --major: %w", err)
			}
			minor, err := strconv.ParseUint(cmd.Flag("minor").Value.String(), 10, 32)
			if err != nil {
				return fmt.Errorf("invalid --minor: %w", err)
			}

			allow, _ := strconv.ParseBool(cmd.Flag("allow").Value.String())
			permsStr := cmd.Flag("permissions").Value.String()
			perms := permissionsToU32(permsStr)

			pinPath := deviceMapPinPath(cgroupPath)
			return updateDeviceMap(pinPath, devType, uint32(major), uint32(minor), perms, allow)
		},
	}
	updateDeviceCmd.Flags().String("cgroup-path", "", "the cgroup directory path whose device map to update")
	updateDeviceCmd.Flags().String("device-type", "b", "device type: 'b' (block) or 'c' (char)")
	updateDeviceCmd.Flags().Uint32("major", 0, "device major number")
	updateDeviceCmd.Flags().Uint32("minor", 0, "device minor number")
	updateDeviceCmd.Flags().Bool("allow", true, "true to allow, false to deny (remove from map)")
	updateDeviceCmd.Flags().String("permissions", "", "device permissions (combination of r, w, m)")

	listDevicesCmd := &cobra.Command{
		Use:   "list-devices",
		Short: "List all devices in the eBPF device map for a cgroup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cgroupPath := cmd.Flag("cgroup-path").Value.String()
			if cgroupPath == "" {
				return fmt.Errorf("--cgroup-path is required")
			}

			pinPath := deviceMapPinPath(cgroupPath)
			entries, err := listDeviceMap(pinPath)
			if err != nil {
				return err
			}

			out, err := json.Marshal(entries)
			if err != nil {
				return fmt.Errorf("cannot marshal device list: %w", err)
			}
			fmt.Println(string(out))
			return nil
		},
	}
	listDevicesCmd.Flags().String("cgroup-path", "", "the cgroup directory path whose device map to list")

	spliceDeviceMapCmd := &cobra.Command{
		Use:   "splice-device-map",
		Short: "Splice an eBPF device map lookup into the cgroup's device filter program",
		RunE: func(cmd *cobra.Command, args []string) error {
			pathsRaw := cmd.Flag("cgroup-paths").Value.String()
			if pathsRaw == "" {
				return fmt.Errorf("--cgroup-paths is required")
			}
			cgroupPaths := strings.Split(pathsRaw, ",")

			pinPath, err := spliceDeviceMapLookup(cgroupPaths)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "device map pinned at %s\n", pinPath)
			return nil
		},
	}
	spliceDeviceMapCmd.Flags().String("cgroup-paths", "", "comma-separated cgroup directory paths to splice")

	rootCmd.AddCommand(
		execCmd,
		mntCmd,
		umntCmd,
		selinuxCmd,
		createTapCmd,
		createMDEVCmd,
		removeMDEVCmd,
		cgroupsCmd,
		spliceDeviceMapCmd,
		updateDeviceCmd,
		listDevicesCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
