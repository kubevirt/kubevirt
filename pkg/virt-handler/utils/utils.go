package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"syscall"

	"strconv"
	"strings"

	"github.com/golang/glog"
)

type NSResult struct {
	Pid    string
	Net    string
	Mnt    string
	User   string
	Ipc    string
	Cgroup string
	Uts    string
}

func GetLibvirtPidFromFile(file string) (int, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		glog.Errorf("Cannot open libvirt pid file: %s", err)
		return -1, err
	}
	lines := strings.Split(string(content), "\n")
	pid, _ := strconv.Atoi(lines[0])
	glog.Errorf("Got libvirt pid file: %d", pid)
	return pid, nil

}

func GetPid(socket string) (int, error) {
	sock, err := net.Dial("unix", socket)
	if err != nil {
		return -1, err
	}
	defer sock.Close()

	ufile, err := sock.(*net.UnixConn).File()
	glog.Errorf("Got FD: %d", int(ufile.Fd()))
	if err != nil {
		return -1, err
	}
	ucreds, err := syscall.GetsockoptUcred(int(ufile.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return -1, err
	}
	return int(ucreds.Pid), nil
}

func GetNSFromPid(pid int) *NSResult {
	return &NSResult{
		Pid:    getNSPath(pid, "pid"),
		Net:    getNSPath(pid, "net"),
		Mnt:    getNSPath(pid, "mnt"),
		User:   getNSPath(pid, "user"),
		Ipc:    getNSPath(pid, "ipc"),
		Cgroup: getNSPath(pid, "cgroup"),
		Uts:    getNSPath(pid, "uts"),
	}

}
func getNSPath(pid int, ns string) string {
	return fmt.Sprintf("/proc/%d/ns/%s", pid, ns)
}
