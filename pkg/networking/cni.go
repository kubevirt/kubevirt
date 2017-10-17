package networking

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"encoding/json"
	"io/ioutil"

	"github.com/containernetworking/cni/pkg/types/current"
)

type CNIToolInterface interface {
	CNIAdd(id string, netConf string, devName string, mac *string, pid int) (*current.Result, error)
	CNIDel(id string, netConf string, devName string, mac *string, pid int) error
}

type cnitool struct {
	toolDir    string
	cniDir     string
	cniConfDir string
}

func NewCNITool(toolDir string, cniDir string, cniConfDir string) CNIToolInterface {
	return &cnitool{strings.TrimSuffix(toolDir, "/"), cniDir, cniConfDir}
}

func (i *cnitool) CNIAdd(id string, netConf string, devName string, mac *string, pid int) (*current.Result, error) {

	args := []string{
		"add", id, netConf, devName,
		"--from-ns", strconv.Itoa(pid),
		"--to-ns", strconv.Itoa(pid),
		"--cni-path", i.cniDir,
		"--cni-config-path", i.cniConfDir,
	}

	if mac != nil && *mac != "" {
		args = append(args, "--args", fmt.Sprintf("mac=%s", *mac))
	}

	cmd := exec.Command(i.toolDir+"/cnitool", args...)

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

func (i *cnitool) CNIDel(id string, netConf string, devName string, mac *string, pid int) error {

	args := []string{
		"del", id, netConf, devName,
		"--from-ns", strconv.Itoa(pid),
		"--to-ns", strconv.Itoa(pid),
		"--cni-path", i.cniDir,
		"--cni-config-path", i.cniConfDir}

	if mac != nil && *mac != "" {
		args = append(args, "--args", fmt.Sprintf("mac=%s", *mac))
	}

	cmd := exec.Command(i.toolDir+"/cnitool", args...)

	resp, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Failed with %v, output: %v", err, string(resp))
	}

	return cmd.Run()
}

func SetNetConfMaster(cniConfigDir, name, master, via string) error {
	b, err := ioutil.ReadFile(cniConfigDir + "/" + name)
	if err != nil {
		return fmt.Errorf("error reading config file %s", name)
	}

	var raw map[string]interface{}
	err = json.Unmarshal(b, &raw)
	if err != nil {
		return fmt.Errorf("error unmarshalling config file %s", name)
	}

	raw["master"] = master
	raw["ipam"].(map[string]interface{})["via"] = via

	b, err = json.MarshalIndent(&raw, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling config file %s", name)
	}

	err = ioutil.WriteFile(cniConfigDir+"/"+name, b, 0)
	if err != nil {
		return fmt.Errorf("error writing config %s to file", name)
	}
	return nil
}
