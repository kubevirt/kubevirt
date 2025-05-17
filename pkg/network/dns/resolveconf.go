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
 * Copyright The KubeVirt Authors.
 */

package dns

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"regexp"
	"strings"

	"kubevirt.io/client-go/log"
)

const (
	domainSearchPrefix  = "search"
	nameserverPrefix    = "nameserver"
	defaultDNS          = "8.8.8.8"
	defaultSearchDomain = "cluster.local"
)

func ParseNameservers(content string) ([][]byte, error) {
	var nameservers [][]byte

	re, err := regexp.Compile("([0-9]{1,3}.?){4}")
	if err != nil {
		return nameservers, err
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, nameserverPrefix) {
			nameserver := re.FindString(line)
			if nameserver != "" {
				nameservers = append(nameservers, net.ParseIP(nameserver).To4())
			}
		}
	}

	if err = scanner.Err(); err != nil {
		return nameservers, err
	}

	// apply a default DNS if none found from pod
	if len(nameservers) == 0 {
		nameservers = append(nameservers, net.ParseIP(defaultDNS).To4())
	}

	return nameservers, nil
}

func ParseSearchDomains(content string) ([]string, error) {
	var searchDomains []string

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, domainSearchPrefix) {
			doms := strings.Fields(strings.TrimPrefix(line, domainSearchPrefix))
			for _, dom := range doms {
				// domain names are case insensitive but kubernetes allows only lower-case
				searchDomains = append(searchDomains, strings.ToLower(dom))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(searchDomains) == 0 {
		searchDomains = append(searchDomains, defaultSearchDomain)
	}

	return searchDomains, nil
}

// GetLongestServiceDomainName returns the longest service search domain entry
func GetLongestServiceDomainName(searchDomains []string) string {
	serviceDomains := GetServiceDomainList(searchDomains)
	return GetDomainName(serviceDomains)
}

// GetDomainName returns the longest search domain entry, which is the most exact equivalent to a domain
func GetDomainName(searchDomains []string) string {
	selected := ""
	for _, d := range searchDomains {
		if len(d) > len(selected) {
			selected = d
		}
	}
	return selected
}

// GetServiceDomainList returns a list of search domains which are a service entry
func GetServiceDomainList(searchDomains []string) []string {
	const k8sServiceInfix = ".svc."

	serviceDomains := []string{}
	for _, d := range searchDomains {
		if strings.Contains(d, k8sServiceInfix) {
			serviceDomains = append(serviceDomains, d)
		}
	}
	return serviceDomains
}

// DomainNameWithSubdomain returns the DNS domain according subdomain.
// In case subdomain already exists in the domain, returns empty string, as nothing should be added.
// In case subdomain is empty, returns empty string, as nothing should be added.
// The motivation is that glibc prior to 2.26 had 6 domain / 256 bytes limit,
// Due to this limitation subdomain.namespace.svc.cluster.local DNS was not added by k8s to the pod /etc/resolv.conf.
// This function calculates the missing domain, which will be added by kubevirt.
// see https://github.com/kubernetes/kubernetes/issues/48019 for more details.
func DomainNameWithSubdomain(searchDomains []string, subdomain string) string {
	if subdomain == "" {
		return ""
	}

	domainName := GetLongestServiceDomainName(searchDomains)
	if domainName != "" && !strings.HasPrefix(domainName, subdomain+".") {
		return subdomain + "." + domainName
	}

	return ""
}

// GetResolvConfDetailsFromPod reads and parses the DNS resolver's configuration file.
func GetResolvConfDetailsFromPod() ([][]byte, []string, error) {
	// #nosec No risk for path injection. resolvConf is static "/etc/resolve.conf"
	const resolvConf = "/etc/resolv.conf"

	b, err := os.ReadFile(resolvConf)
	if err != nil {
		return nil, nil, err
	}

	nameservers, err := ParseNameservers(string(b))
	if err != nil {
		return nil, nil, err
	}

	searchDomains, err := ParseSearchDomains(string(b))
	if err != nil {
		return nil, nil, err
	}

	log.Log.Reason(err).Infof("Found nameservers in %s: %s", resolvConf, bytes.Join(nameservers, []byte{' '}))
	log.Log.Reason(err).Infof("Found search domains in %s: %s", resolvConf, strings.Join(searchDomains, " "))

	return nameservers, searchDomains, err
}
