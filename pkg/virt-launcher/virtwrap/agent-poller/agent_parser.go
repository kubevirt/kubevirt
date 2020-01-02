package agentpoller

import (
	"encoding/json"
	"fmt"
	"regexp"

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

// stripAgentResponse use regex to strip the wrapping item and returns the
// embedded object.
// It is a workaround so the amount of copy paste code is limited
func stripAgentResponse(agentReply string) string {
	return stripRE.FindStringSubmatch(agentReply)[1]
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

	resultInterfaces := convertInterfaceStatusesFromAgentJson(interfaces)

	return resultInterfaces, nil
}

// MergeAgentStatusesWithDomainData merges QEMU interfaces with agent interfaces
func MergeAgentStatusesWithDomainData(domInterfaces []api.Interface, interfaceStatuses []api.InterfaceStatus) []api.InterfaceStatus {
	aliasByMac := map[string]string{}
	for _, ifc := range domInterfaces {
		mac := ifc.MAC.MAC
		alias := ifc.Alias.Name
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

// calculateInterfaceStatusesFromAgentJson does the conversion from agent info to api domain interfaces
func convertInterfaceStatusesFromAgentJson(agentResult []Interface) []api.InterfaceStatus {
	interfaceStatuses := []api.InterfaceStatus{}
	for _, ifc := range agentResult {
		if ifc.Name == "lo" {
			continue
		}
		interfaceIP, interfaceIPs := extractIps(ifc.IPs)
		interfaceStatuses = append(interfaceStatuses, api.InterfaceStatus{
			Mac:           ifc.MAC,
			Ip:            interfaceIP,
			IPs:           interfaceIPs,
			InterfaceName: ifc.Name,
		})
	}
	return interfaceStatuses
}

func extractIps(ipAddresses []IP) (string, []string) {
	interfaceIPs := []string{}
	var interfaceIP string
	for _, ipAddr := range ipAddresses {
		ip := fmt.Sprintf("%s/%d", ipAddr.IP, ipAddr.Prefix)
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
