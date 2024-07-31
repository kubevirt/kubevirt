package main

import (
	"fmt"
	"os"
	"os/user"
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
			cmd.Printf(cmd.UsageString())
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
			rawPath, err := cmd.Flags().GetBool("raw-path")
			if err != nil {
				return err
			}
			for _, opt := range strings.Split(mntOptions, ",") {
				opt = strings.TrimSpace(opt)
				switch opt {
				case "ro":
					mntOpts = mntOpts | syscall.MS_RDONLY
				case "bind":
					mntOpts = mntOpts | syscall.MS_BIND
				case "sync":
					mntOpts = mntOpts | syscall.MS_SYNC
				case "":
					if !rawPath {
						return fmt.Errorf("empty option is only supported when --raw-path is used")
					}
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

			if rawPath {
				return syscall.Mount(args[0], args[1], fsType, uintptr(mntOpts), "")
			}
			return syscall.Mount(sourceFile.SafePath(), targetFile.SafePath(), fsType, uintptr(mntOpts), "")
		},
	}
	mntCmd.Flags().StringP("options", "o", "", "comma separated list of mount options")
	mntCmd.Flags().StringP("type", "t", "", "fstype")
	mntCmd.Flags().BoolP("raw-path", "p", false, "using raw path")

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
			cmd.Printf(cmd.UsageString())
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
			isRootless, err := strconv.ParseBool(cmd.Flag("rootless").Value.String())
			if err != nil {
				return fmt.Errorf("cannot convert rootless into bool. err: %v", err)
			}
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

			if err = setCgroupResources(unmarshalledPaths, unmarshalledResources, isRootless, isV2); err != nil {
				return err
			}

			return nil
		},
	}

	cgroupsCmd.Flags().String("subsystem-paths", "", "marshalled map[string]string type, encoded to base64 format. "+
		"For v1 key is cgroup subsystem and value is its path, for v2 the only key is an empty string and the value is cgroup dir path.")
	cgroupsCmd.Flags().String("resources", "", "marshalled Resources type (defined in github.com/opencontainers/runc/libcontainer/configs/cgroup_linux.go), encoded to base64 format")
	cgroupsCmd.Flags().Bool("rootless", false, "true to run rootless")
	cgroupsCmd.Flags().Bool("isV2", false, "true to run rootless")

	rootCmd.AddCommand(
		execCmd,
		mntCmd,
		umntCmd,
		selinuxCmd,
		createTapCmd,
		createMDEVCmd,
		removeMDEVCmd,
		cgroupsCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
