package agentpoller

import (
	"fmt"
	"regexp"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var stripStringRE = regexp.MustCompile(`{\s*\"return\":\s*\"([\s\S]*)\"\s*}`)

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
