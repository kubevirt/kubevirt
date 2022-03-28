// +build !nofilesystem

package stats

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"kubevirt.io/client-go/log"
)

const (
	defMountPointsExcluded = "^/(dev|proc|sys|var/lib/docker/.+)($|/)"
	defFSTypesExcluded     = "^(autofs|binfmt_misc|bpf|cgroup2?|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|iso9660|mqueue|nsfs|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|selinuxfs|squashfs|sysfs|tracefs)$"
)

var stuckMounts = make(map[string]struct{})
var stuckMountsMtx = &sync.Mutex{}

// GetDomainStatsFs returns filesystem stats.
func GetDomainStatsFs() ([]DomainStatsFs, error) {
	mps, err := mountPointDetails()
	if err != nil {
		return nil, err
	}
	stats := []DomainStatsFs{}
	for _, labels := range mps {
		if regexp.MustCompile(defMountPointsExcluded).MatchString(labels.MountPoint) {
			log.DefaultLogger().Infof("Ignoring mount point, mountpoint: %s", labels.MountPoint)
			continue
		}
		if regexp.MustCompile(defFSTypesExcluded).MatchString(labels.FsType) {
			log.DefaultLogger().Infof("Ignoring fs, type: %s", labels.FsType)
			continue
		}
		stuckMountsMtx.Lock()
		if _, ok := stuckMounts[labels.MountPoint]; ok {
			stats = append(stats, DomainStatsFs{
				Labels:      labels,
				DeviceError: 1,
			})
			log.DefaultLogger().Infof("Mount point is in an unresponsive state, mountpoint: %s", labels.MountPoint)
			stuckMountsMtx.Unlock()
			continue
		}
		stuckMountsMtx.Unlock()

		// The success channel is used do tell the "watcher" that the stat
		// finished successfully. The channel is closed on success.
		success := make(chan struct{})
		go stuckMountWatcher(labels.MountPoint, success)

		buf := new(unix.Statfs_t)
		err = unix.Statfs(rootfsFilePath(labels.MountPoint), buf)
		stuckMountsMtx.Lock()
		close(success)
		// If the mount has been marked as stuck, unmark it and log it's recovery.
		if _, ok := stuckMounts[labels.MountPoint]; ok {
			log.DefaultLogger().Infof("Mount point has recovered, monitoring will resume， mountpoint：%s", labels.MountPoint)
			delete(stuckMounts, labels.MountPoint)
		}
		stuckMountsMtx.Unlock()

		if err != nil {
			stats = append(stats, DomainStatsFs{
				Labels:      labels,
				DeviceError: 1,
			})

			log.DefaultLogger().Infof("Error on statfs() system call, rootfs: %s, err: %s", rootfsFilePath(labels.MountPoint), err)
			continue
		}

		var ro float64
		for _, option := range strings.Split(labels.Options, ",") {
			if option == "ro" {
				ro = 1
				break
			}
		}

		stats = append(stats, DomainStatsFs{
			Labels:    labels,
			Size:      float64(buf.Blocks) * float64(buf.Bsize),
			Free:      float64(buf.Bfree) * float64(buf.Bsize),
			Avail:     float64(buf.Bavail) * float64(buf.Bsize),
			Files:     float64(buf.Files),
			FilesFree: float64(buf.Ffree),
			Ro:        ro,
		})
	}
	return stats, nil
}

// stuckMountWatcher listens on the given success channel and if the channel closes
// then the watcher does nothing. If instead the timeout is reached, the
// mount point that is being watched is marked as stuck.
func stuckMountWatcher(mountPoint string, success chan struct{}) {
	select {
	case <-success:
		// Success
	case <-time.After(time.Duration(5 * time.Second)):
		// Timed out, mark mount as stuck
		stuckMountsMtx.Lock()
		select {
		case <-success:
			// Success came in just after the timeout was reached, don't label the mount as stuck
		default:
			log.DefaultLogger().Infof("Mount point timed out, it is being labeled as stuck and will not be monitored, mountpoint: %s", mountPoint)
			stuckMounts[mountPoint] = struct{}{}
		}
		stuckMountsMtx.Unlock()
	}
}

func mountPointDetails() ([]FsLabels, error) {
	file, err := os.Open(procFilePath("1/mounts"))
	if errors.Is(err, os.ErrNotExist) {
		// Fallback to `/proc/mounts` if `/proc/1/mounts` is missing due hidepid.
		log.DefaultLogger().Infof("Reading root mounts failed, falling back to system mounts, err: %s", err)
		file, err = os.Open(procFilePath("mounts"))
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseFilesystemLabels(file)
}

func parseFilesystemLabels(r io.Reader) ([]FsLabels, error) {
	var filesystems []FsLabels

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 4 {
			return nil, fmt.Errorf("malformed mount point information: %q", scanner.Text())
		}

		// Ensure we handle the translation of \040 and \011
		// as per fstab(5).
		parts[1] = strings.Replace(parts[1], "\\040", " ", -1)
		parts[1] = strings.Replace(parts[1], "\\011", "\t", -1)

		filesystems = append(filesystems, FsLabels{
			Device:     parts[0],
			MountPoint: parts[1],
			FsType:     parts[2],
			Options:    parts[3],
		})
	}

	return filesystems, scanner.Err()
}

func procFilePath(name string) string {
	return filepath.Join("/proc", name)
}

func rootfsFilePath(name string) string {
	return filepath.Join("/", name)
}
