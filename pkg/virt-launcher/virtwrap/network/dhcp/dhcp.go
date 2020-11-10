/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package dhcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	dhcp "github.com/krolaw/dhcp4"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

const (
	infiniteLease             = 999 * 24 * time.Hour
	errorSearchDomainNotValid = "Search domain is not valid"
	errorSearchDomainTooLong  = "Search domains length exceeded allowable size"
	errorNTPConfiguration     = "Could not parse NTP server as IPv4 address: %s"
)

// simple domain validation regex. Put it here to avoid compiling each time.
// Note this requires that unicode domains be presented in their ASCII format
var searchDomainValidationRegex = regexp.MustCompile(`^(?:[_a-z0-9](?:[_a-z0-9-]{0,61}[a-z0-9])?\.)*(?:[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?)?$`)

func SingleClientDHCPServer(
	clientMAC net.HardwareAddr,
	clientIP net.IP,
	clientMask net.IPMask,
	serverIface string,
	serverIP net.IP,
	routerIP net.IP,
	dnsIPs [][]byte,
	routes *[]netlink.Route,
	searchDomains []string,
	mtu uint16,
	customDHCPOptions *v1.DHCPOptions) error {

	log.Log.Info("Starting SingleClientDHCPServer")

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("reading the pods hostname failed: %v", err)
	}

	options, err := prepareDHCPOptions(clientMask, routerIP, dnsIPs, routes, searchDomains, mtu, hostname, customDHCPOptions)
	if err != nil {
		return err
	}

	handler := &DHCPHandler{
		clientIP:      clientIP,
		clientMAC:     clientMAC,
		serverIP:      serverIP.To4(),
		leaseDuration: infiniteLease,
		options:       options,
	}

	// turn TX offload checksum because it causes dhcp failures
	if err := EthtoolTXOff(serverIface); err != nil {
		log.Log.Reason(err).Errorf("Failed to set tx offload for interface %s off", serverIface)
		return err
	}

	l, err := NewUDP4FilterListener(serverIface, ":67")
	if err != nil {
		return err
	}
	defer l.Close()
	err = dhcp.Serve(l, handler)
	if err != nil {
		return err
	}
	return nil
}

func prepareDHCPOptions(
	clientMask net.IPMask,
	routerIP net.IP,
	dnsIPs [][]byte,
	routes *[]netlink.Route,
	searchDomains []string,
	mtu uint16,
	hostname string,
	customDHCPOptions *v1.DHCPOptions) (dhcp.Options, error) {

	mtuArray := make([]byte, 2)
	binary.BigEndian.PutUint16(mtuArray, mtu)

	dhcpOptions := dhcp.Options{
		dhcp.OptionSubnetMask:       []byte(clientMask),
		dhcp.OptionRouter:           []byte(routerIP),
		dhcp.OptionDomainNameServer: bytes.Join(dnsIPs, nil),
		dhcp.OptionInterfaceMTU:     mtuArray,
	}

	netRoutes := formClasslessRoutes(routes)

	if netRoutes != nil {
		dhcpOptions[dhcp.OptionClasslessRouteFormat] = netRoutes
	}

	searchDomainBytes, err := convertSearchDomainsToBytes(searchDomains)
	if err != nil {
		return nil, err
	}
	if searchDomainBytes != nil {
		dhcpOptions[dhcp.OptionDomainSearch] = searchDomainBytes
	}

	dhcpOptions[dhcp.OptionHostName] = []byte(hostname)

	// Windows will ask for the domain name and use it for DNS resolution
	domainName := getDomainName(searchDomains)
	if len(domainName) > 0 {
		dhcpOptions[dhcp.OptionDomainName] = []byte(domainName)
	}

	if customDHCPOptions != nil {
		if customDHCPOptions.TFTPServerName != "" {
			log.Log.Infof("Setting dhcp option tftp server name to %s", customDHCPOptions.TFTPServerName)
			dhcpOptions[dhcp.OptionTFTPServerName] = []byte(customDHCPOptions.TFTPServerName)
		}
		if customDHCPOptions.BootFileName != "" {
			log.Log.Infof("Setting dhcp option boot file name to %s", customDHCPOptions.BootFileName)
			dhcpOptions[dhcp.OptionBootFileName] = []byte(customDHCPOptions.BootFileName)
		}

		if len(customDHCPOptions.NTPServers) > 0 {
			log.Log.Infof("Setting dhcp option NTP server name to %s", customDHCPOptions.NTPServers)

			ntpServers := [][]byte{}

			for _, server := range customDHCPOptions.NTPServers {
				ip := net.ParseIP(server).To4()

				if ip == nil {
					return nil, fmt.Errorf(errorNTPConfiguration, server)
				}
				ntpServers = append(ntpServers, []byte(ip))
			}

			dhcpOptions[dhcp.OptionNetworkTimeProtocolServers] = bytes.Join(ntpServers, nil)
		}

		if customDHCPOptions.PrivateOptions != nil {
			for _, privateOptions := range customDHCPOptions.PrivateOptions {
				if privateOptions.Option >= 224 && privateOptions.Option <= 254 {
					dhcpOptions[dhcp.OptionCode(byte(privateOptions.Option))] = []byte(privateOptions.Value)
				}
			}
		}
	}

	return dhcpOptions, nil
}

type DHCPHandler struct {
	serverIP      net.IP
	clientIP      net.IP
	clientMAC     net.HardwareAddr
	leaseDuration time.Duration
	options       dhcp.Options
}

func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	log.Log.V(4).Info("Serving a new request")
	if mac := p.CHAddr(); !bytes.Equal(mac, h.clientMAC) {
		log.Log.V(4).Info("The request is not from our client")
		return nil // Is not our client
	}

	switch msgType {

	case dhcp.Discover:
		log.Log.V(4).Info("The request has message type DISCOVER")
		return dhcp.ReplyPacket(p, dhcp.Offer, h.serverIP, h.clientIP, h.leaseDuration,
			h.options.SelectOrderOrAll(nil))

	case dhcp.Request:
		log.Log.V(4).Info("The request has message type REQUEST")
		return dhcp.ReplyPacket(p, dhcp.ACK, h.serverIP, h.clientIP, h.leaseDuration,
			h.options.SelectOrderOrAll(nil))

	default:
		log.Log.V(4).Info("The request has unhandled message type")
		return nil // Ignored message type

	}
}

func sortRoutes(routes []netlink.Route) []netlink.Route {
	// Default route must come last, otherwise it may not get applied
	// because there is no route to its gateway yet
	var sortedRoutes []netlink.Route
	var defaultRoutes []netlink.Route
	for _, route := range routes {
		if route.Dst == nil {
			defaultRoutes = append(defaultRoutes, route)
			continue
		}
		sortedRoutes = append(sortedRoutes, route)
	}
	for _, defaultRoute := range defaultRoutes {
		sortedRoutes = append(sortedRoutes, defaultRoute)
	}

	return sortedRoutes
}

func formClasslessRoutes(routes *[]netlink.Route) (formattedRoutes []byte) {
	// See RFC4332 for additional information
	// (https://tools.ietf.org/html/rfc3442)
	// For example:
	// 		routes:
	//				10.0.0.0/8 ,  gateway: 10.1.2.3
	//              192.168.1/24, gateway: 192.168.2.3
	//		would result in the following structure:
	//      []byte{8, 10, 10, 1, 2, 3, 24, 192, 168, 1, 192, 168, 2, 3}
	if routes == nil {
		return []byte{}
	}

	sortedRoutes := sortRoutes(*routes)
	for _, route := range sortedRoutes {
		if route.Dst == nil {
			route.Dst = &net.IPNet{
				IP:   net.IPv4(0, 0, 0, 0),
				Mask: net.CIDRMask(0, 32),
			}
		}
		ip := route.Dst.IP.To4()
		width, _ := route.Dst.Mask.Size()
		octets := 0
		if width > 0 {
			octets = (width-1)/8 + 1
		}
		newRoute := append([]byte{byte(width)}, ip[0:octets]...)
		gateway := route.Gw.To4()
		if gateway == nil {
			gateway = []byte{0, 0, 0, 0}
		}
		newRoute = append(newRoute, gateway...)
		formattedRoutes = append(formattedRoutes, newRoute...)
	}
	return
}

func convertSearchDomainsToBytes(searchDomainStrings []string) ([]byte, error) {
	/*
	   https://tools.ietf.org/html/rfc3397
	   https://tools.ietf.org/html/rfc1035

	   Option for search domain string is covered by RFC3397, option contains
	   RFC1035 domain data.

	   Convert domain strings to a DNS RFC1035 section 3.1 compatible byte slice.
	   This is basically just splitting the domain on dot and prepending each
	   substring with a byte that indicates its length. Then we join and null terminate.

	   "example.com" becomes:
	   []byte{7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}

	   Note that there is a compression scheme described in section 4.1.4 where pointers
	   can be used to avoid duplication. This is optional for servers, and resolv.conf
	   limits max search domain length anyway, so we can skip compression.
	*/
	var searchDomainBytes []byte
	for _, domain := range searchDomainStrings {
		if isValidSearchDomain(domain) {
			labels := strings.Split(domain, ".")
			for _, label := range labels {
				searchDomainBytes = append(searchDomainBytes, byte(len(label)))
				searchDomainBytes = append(searchDomainBytes, []byte(label)...)
			}
			searchDomainBytes = append(searchDomainBytes, 0)
		} else {
			return searchDomainBytes, fmt.Errorf("%s: '%s'", errorSearchDomainNotValid, domain)
		}
	}

	// ensure we haven't gone past length limit of DHCP option data
	if len(searchDomainBytes) > 255 {
		return searchDomainBytes, fmt.Errorf("%s: was %d long", errorSearchDomainTooLong, len(searchDomainBytes))
	}

	return searchDomainBytes, nil
}

func isValidSearchDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return searchDomainValidationRegex.MatchString(domain)
}

//getDomainName returns the longest search domain entry, which is the most exact equivalent to a domain
func getDomainName(searchDomains []string) string {
	selected := ""
	for _, d := range searchDomains {
		if len(d) > len(selected) {
			selected = d
		}
	}
	return selected
}
