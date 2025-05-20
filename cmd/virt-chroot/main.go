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
	targetUser   string
	targetUserID int
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

			var u *user.User
			if targetUserID >= 0 {
				_, _, errno := syscall.Syscall(syscall.SYS_SETUID, uintptr(targetUserID), 0, 0)
				if errno != 0 {
					return fmt.Errorf("failed to switch to user: %d. errno: %d", targetUserID, errno)
				}
			} else if targetUser != "" {
				var err error
				u, err = user.Lookup(targetUser)
				if err != nil {
					return fmt.Errorf("failed to look up user: %v", err)
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

	rootCmd.PersistentFlags().StringVar(&mntNamespace, "mount", "", "mount namespace to use")
	rootCmd.PersistentFlags().StringVar(&targetUser, "user", "", "switch to this targetUser to e.g. drop privileges")
	rootCmd.PersistentFlags().IntVar(&targetUserID, "userid", -1, "switch to this targetUser to e.g. drop privileges")

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
			var dataOpts []string

			fsType := cmd.Flag("type").Value.String()
			mntOptions := cmd.Flag("options").Value.String()
			var (
				uid = -1
				gid = -1
			)
			for _, opt := range strings.Split(mntOptions, ",") {
				opt = strings.TrimSpace(opt)
				switch {
				case opt == "ro":
					mntOpts = mntOpts | syscall.MS_RDONLY
				case opt == "bind":
					mntOpts = mntOpts | syscall.MS_BIND
				case opt == "remount":
					mntOpts = mntOpts | syscall.MS_REMOUNT
				case strings.HasPrefix(opt, "uid="):
					uidS := strings.TrimPrefix(opt, "uid=")
					uidI, err := strconv.Atoi(uidS)
					if err != nil {
						return fmt.Errorf("failed to parse uid: %w", err)
					}
					uid = uidI
					dataOpts = append(dataOpts, opt)
				case strings.HasPrefix(opt, "gid="):
					gidS := strings.TrimPrefix(opt, "gid=")
					gidI, err := strconv.Atoi(gidS)
					if err != nil {
						return fmt.Errorf("failed to parse gid: %w", err)
					}
					gid = gidI
					dataOpts = append(dataOpts, opt)
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
			if uid >= 0 && gid >= 0 {
				err = os.Chown(targetFile.SafePath(), uid, gid)
				if err != nil {
					return fmt.Errorf("chown target failed: %w", err)
				}
			}
			var data string
			if len(dataOpts) > 0 {
				data = strings.Join(dataOpts, ",")
			}
			return syscall.Mount(sourceFile.SafePath(), targetFile.SafePath(), fsType, uintptr(mntOpts), data)
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
