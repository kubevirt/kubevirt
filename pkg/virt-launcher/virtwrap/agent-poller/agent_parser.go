package agentpoller

import (
	"encoding/json"
	"fmt"
	"regexp"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// GuestOsInfo is the response from 'guest-get-osinfo'
type GuestOsInfo struct {
	Name          string `json:"name"`
	KernelRelease string `json:"kernel-release"`
	Version       string `json:"version"`
	PrettyName    string `json:"pretty-name"`
	VersionId     string `json:"version-id"`
	KernelVersion string `json:"kernel-version"`
	Machine       string `json:"machine"`
	Id            string `json:"id"`
}

// Interface for json unmarshalling
type Interface struct {
	MAC  string `json:"hardware-address"`
	IPs  []IP   `json:"ip-addresses"`
	Name string `json:"name"`
}

// IP for json unmarshalling
type IP struct {
	IP     string `json:"ip-address"`
	Type   string `json:"ip-address-type"`
	Prefix int    `json:"prefix"`
}

var stripRE = regexp.MustCompile(`{\s*\"return\":\s*([{\[][\s\S]*[}\]])\s*}`)
var stripStringRE = regexp.MustCompile(`{\s*\"return\":\s*\"([\s\S]*)\"\s*}`)

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
	result := stripStringRE.FindStringSubmatch(agentReply)
	if len(result) < 2 {
		return ""
	}

	return result[1]
}

// Hostname of the guest vm
type Hostname struct {
	Hostname string `json:"host-name"`
}

// Timezone of the host
type Timezone struct {
	Zone   string `json:"zone,omitempty"`
	Offset int    `json:"offset"`
}

// User on the guest host
type User struct {
	Name      string  `json:"user"`
	Domain    string  `json:"domain"`
	LoginTime float64 `json:"login-time"`
}

// Filesystem of the host
type Filesystem struct {
	Name       string `json:"name"`
	Mountpoint string `json:"mountpoint"`
	Type       string `json:"type"`
	UsedBytes  int    `json:"used-bytes,omitempty"`
	TotalBytes int    `json:"total-bytes,omitempty"`
}

// AgentInfo from the guest VM serves the purpose
// of checking the GA presence and version compatibility
type AgentInfo struct {
	Version           string                     `json:"version"`
	SupportedCommands []v1.GuestAgentCommandInfo `json:"supported_commands,omitempty"`
}

// parseGuestOSInfo parse agent reply string, extract guest os info
// and converts the response to API domain guest os info
func parseGuestOSInfo(agentReply string) (api.GuestOSInfo, error) {
	guestOSInfo := GuestOsInfo{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &guestOSInfo)
	if err != nil {
		return api.GuestOSInfo{}, err
	}

	resultInfo := api.GuestOSInfo{
		Name:          guestOSInfo.Name,
		KernelRelease: guestOSInfo.KernelRelease,
		Version:       guestOSInfo.Version,
		PrettyName:    guestOSInfo.PrettyName,
		VersionId:     guestOSInfo.VersionId,
		KernelVersion: guestOSInfo.KernelVersion,
		Machine:       guestOSInfo.Machine,
		Id:            guestOSInfo.Id,
	}

	return resultInfo, nil
}

// parseInterfaces parses agent reply string, extracts network interfaces
// and converts the response to API domain list of interfaces
func parseInterfaces(agentReply string) ([]api.InterfaceStatus, error) {
	interfaces := []Interface{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &interfaces)
	if err != nil {
		return []api.InterfaceStatus{}, err
	}

	resultInterfaces := convertInterfaceStatusesFromAgentJSON(interfaces)

	return resultInterfaces, nil
}

// parseHostname from the agent response
func parseHostname(agentReply string) (string, error) {
	result := Hostname{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		return "", err
	}

	return result.Hostname, nil
}

// parseFSFreezeStatus from the agent response
func ParseFSFreezeStatus(agentReply string) (api.FSFreeze, error) {
	response := stripAgentStringResponse(agentReply)
	if response == "" {
		return api.FSFreeze{}, fmt.Errorf("Failed to strip FSFreeze status: %v", agentReply)
	}

	return api.FSFreeze{
		Status: response,
	}, nil
}

// parseTimezone from the agent response
func parseTimezone(agentReply string) (api.Timezone, error) {
	result := Timezone{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		return api.Timezone{}, err
	}

	return api.Timezone{
		Zone:   result.Zone,
		Offset: result.Offset,
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
		})
	}

	return convertedResult, nil
}

// parseUsers from the agent response
func parseUsers(agentReply string) ([]api.User, error) {
	result := []User{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		return []api.User{}, err
	}

	convertedResult := []api.User{}

	for _, user := range result {
		convertedResult = append(convertedResult, api.User{
			Name:      user.Name,
			Domain:    user.Domain,
			LoginTime: user.LoginTime,
		})
	}

	return convertedResult, nil
}

// parseAgent gets the agent version from response
func parseAgent(agentReply string) (AgentInfo, error) {
	gaInfo := AgentInfo{}
	response := stripAgentResponse(agentReply)

	err := json.Unmarshal([]byte(response), &gaInfo)
	if err != nil {
		return AgentInfo{}, err
	}

	log.Log.V(3).Infof("guest agent info: %v", gaInfo)

	return gaInfo, nil
}

// MergeAgentStatusesWithDomainData merge QEMU interfaces with agent interfaces
func MergeAgentStatusesWithDomainData(domInterfaces []api.Interface, interfaceStatuses []api.InterfaceStatus) []api.InterfaceStatus {
	aliasByMac := map[string]string{}
	for _, ifc := range domInterfaces {
		mac := ifc.MAC.MAC
		alias := ifc.Alias.GetName()
		aliasByMac[mac] = alias
	}

	aliasesCoveredByAgent := []string{}
	// Add alias from domain to interfaceStatus
	for i, interfaceStatus := range interfaceStatuses {
		if alias, exists := aliasByMac[interfaceStatus.Mac]; exists {
			interfaceStatuses[i].Name = alias
			aliasesCoveredByAgent = append(aliasesCoveredByAgent, alias)
		}
	}

	// If interface present in domain was not found in interfaceStatuses, add it
	for mac, alias := range aliasByMac {
		isCoveredByAgentData := false
		for _, coveredAlias := range aliasesCoveredByAgent {
			if alias == coveredAlias {
				isCoveredByAgentData = true
				break
			}
		}
		if !isCoveredByAgentData {
			interfaceStatuses = append(interfaceStatuses,
				api.InterfaceStatus{
					Mac:  mac,
					Name: alias,
				},
			)
		}
	}
	return interfaceStatuses
}

// convertInterfaceStatusesFromAgentJSON does the conversion from agent info to api domain interfaces
func convertInterfaceStatusesFromAgentJSON(agentResult []Interface) []api.InterfaceStatus {
	interfaceStatuses := []api.InterfaceStatus{}
	for _, ifc := range agentResult {
		if ifc.Name == "lo" {
			continue
		}

		interfaceIP, interfaceIPs := extractIPs(ifc.IPs)
		interfaceStatuses = append(interfaceStatuses, api.InterfaceStatus{
			Mac:           ifc.MAC,
			Ip:            interfaceIP,
			IPs:           interfaceIPs,
			InterfaceName: ifc.Name,
		})
	}
	return interfaceStatuses
}

func extractIPs(ipAddresses []IP) (string, []string) {
	interfaceIPs := []string{}
	var interfaceIP string
	for _, ipAddr := range ipAddresses {
		ip := ipAddr.IP
		// Prefer ipv4 as the main interface IP
		if ipAddr.Type == "ipv4" && interfaceIP == "" {
			interfaceIP = ip
		}
		interfaceIPs = append(interfaceIPs, ip)
	}
	// If no ipv4 interface was found, set any IP as the main IP of interface
	if interfaceIP == "" && len(interfaceIPs) > 0 {
		interfaceIP = interfaceIPs[0]
	}
	return interfaceIP, interfaceIPs
}
