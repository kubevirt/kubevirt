package agentpoller

import (
	"encoding/json"
	"fmt"
	"regexp"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var (
	stripRE       = regexp.MustCompile(`{\s*\"return\":\s*([{\[][\s\S]*[}\]])\s*}`)
	stripStringRE = regexp.MustCompile(`{\s*\"return\":\s*\"([\s\S]*)\"\s*}`)
)

// stripAgentResponse use regex to strip the wrapping item and returns the
// embedded object.
// It is a workaround so the amount of copy paste code is limited
func stripAgentResponse(agentReply string) string {
	return stripRE.FindStringSubmatch(agentReply)[1]
}

// stripAgentStringResponse use regex to stip the wrapping item
// and returns the embedded string response
// unlike stripAgentResponse the response is a simple string
// rather then a complex object
func stripAgentStringResponse(agentReply string) string {
	const minMatchGroups = 2

	result := stripStringRE.FindStringSubmatch(agentReply)
	if len(result) < minMatchGroups {
		return ""
	}

	return result[1]
}

// Filesystem disk of the host
type FSDisk struct {
	Serial  string `json:"serial,omitempty"`
	BusType string `json:"bus-type"`
}

// Filesystem of the host
type Filesystem struct {
	Name       string   `json:"name"`
	Mountpoint string   `json:"mountpoint"`
	Type       string   `json:"type"`
	UsedBytes  int      `json:"used-bytes,omitempty"`
	TotalBytes int      `json:"total-bytes,omitempty"`
	Disk       []FSDisk `json:"disk,omitempty"`
}

// AgentInfo from the guest VM serves the purpose
// of checking the GA presence and version compatibility
type AgentInfo struct {
	Version           string                     `json:"version"`
	SupportedCommands []v1.GuestAgentCommandInfo `json:"supported_commands,omitempty"`
}

// parseFSFreezeStatus from the agent response
func ParseFSFreezeStatus(agentReply string) (api.FSFreeze, error) {
	response := stripAgentStringResponse(agentReply)
	if response == "" {
		return api.FSFreeze{}, fmt.Errorf("failed to strip FSFreeze status: %v", agentReply)
	}

	return api.FSFreeze{
		Status: response,
	}, nil
}

// parseFilesystem from the agent response
func parseFilesystem(agentReply string) ([]api.Filesystem, error) {
	result := []Filesystem{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		return []api.Filesystem{}, err
	}

	convertedResult := []api.Filesystem{}

	for _, fs := range result {
		convertedResult = append(convertedResult, api.Filesystem{
			Name:       fs.Name,
			Mountpoint: fs.Mountpoint,
			Type:       fs.Type,
			TotalBytes: fs.TotalBytes,
			UsedBytes:  fs.UsedBytes,
			Disk:       parseFSDisks(fs.Disk),
		})
	}

	return convertedResult, nil
}

func parseFSDisks(fsDisks []FSDisk) []api.FSDisk {
	disks := []api.FSDisk{}
	for _, fsDisk := range fsDisks {
		disks = append(disks, api.FSDisk{
			Serial:  fsDisk.Serial,
			BusType: fsDisk.BusType,
		})
	}

	return disks
}

// parseAgent gets the agent version from response
func parseAgent(agentReply string) (AgentInfo, error) {
	const logLevelDebug = 3

	gaInfo := AgentInfo{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &gaInfo)
	if err != nil {
		return AgentInfo{}, err
	}

	log.Log.V(logLevelDebug).Infof("guest agent info: %v", gaInfo)

	return gaInfo, nil
}
