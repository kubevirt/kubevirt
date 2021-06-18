package apputil

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"

	"github.com/openshift/app-netutil/pkg/types"
)

const (
	cpusetPath = "/sys/fs/cgroup/cpuset/cpuset.cpus"
)

type EnvResponse struct {
	Envs map[string]string
}

func GetCPUInfo() (*types.CPUResponse, error) {
	path := filepath.Join("/proc", strconv.Itoa(os.Getpid()), "root", cpusetPath)
	glog.Infof("getting cpuset from path: %s", path)
	cpus, err := ioutil.ReadFile(path)
	if err != nil {
		glog.Errorf("Error getting cpuset info: %v", err)
		return nil, err
	}
	return &types.CPUResponse{CPUSet: string(bytes.TrimSpace(cpus))}, nil
}

func getEnv() (*EnvResponse, error) {
	path := filepath.Join("/proc", strconv.Itoa(os.Getpid()), "environ")
	glog.Infof("getting environment variables from path: %s", path)
	file, err := os.Open(path)
	if err != nil {
		glog.Errorf("Error openning proc environ file: %v", err)
		return nil, err
	}
	defer file.Close()

	envAttrs := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		envs := strings.Split(string(line), "\x00")
		for _, e := range envs {
			parts := strings.Split(string(e), "=")
			if len(parts) == 2 {
				envAttrs[parts[0]] = parts[1]
			}
		}
	}
	return &EnvResponse{Envs: envAttrs}, nil
}
