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
