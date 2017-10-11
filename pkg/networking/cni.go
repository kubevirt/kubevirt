package networking

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/pkg/types/current"
)

type CNIToolInterface interface {
	CNIAdd(id string, netConf string, devName string, pid int) (*current.Result, error)
	CNIDel(id string, netConf string, devName string, pid int) error
}

type cnitool struct {
	toolDir    string
	cniDir     string
	cniConfDir string
}

func NewCNITool(toolDir string, cniDir string, cniConfDir string) CNIToolInterface {
	return &cnitool{strings.TrimSuffix(toolDir, "/"), cniDir, cniConfDir}
}

func (i *cnitool) CNIAdd(id string, netConf string, devName string, pid int) (*current.Result, error) {
	cmd := exec.Command(i.toolDir+"/cnitool",
		"add", id, netConf, devName,
		"--from-ns", strconv.Itoa(pid),
		"--to-ns", strconv.Itoa(pid),
		"--cni-path", i.cniDir,
		"--cni-config-path", i.cniConfDir)

	resp, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Failed with %v, output: %v", err, string(resp))
	}

	res, err := current.NewResult(resp)

	if err != nil {
		return nil, fmt.Errorf("Error reading CNI resposponse: %v", err)
	}
	return current.NewResultFromResult(res)
}

func (i *cnitool) CNIDel(id string, netConf string, devName string, pid int) error {
	cmd := exec.Command(i.toolDir+"/cnitool",
		"del", id, netConf, devName,
		"--from-ns", strconv.Itoa(pid),
		"--to-ns", strconv.Itoa(pid),
		"--cni-path", i.cniDir,
		"--cni-config-path", i.cniConfDir)

	resp, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed with %v, output: %v", err, string(resp))
	}

	return cmd.Run()
}
