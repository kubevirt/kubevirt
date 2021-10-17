package dns

import (
	"bufio"
	"net"
	"regexp"
	"strings"
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

func AddSubdomainSearchDomain(searchDomains []string, subdomain string) []string {
	// workaround of https://github.com/kubernetes/kubernetes/issues/48019
	if subdomain == "" {
		return searchDomains
	}

	domainName := GetDomainName(searchDomains)
	if !strings.Contains(domainName, subdomain) && strings.Contains(domainName, ".svc.") {
		domainName = subdomain + "." + domainName
		searchDomains = append([]string{domainName}, searchDomains...)
	}

	return searchDomains
}

//GetDomainName returns the longest search domain entry, which is the most exact equivalent to a domain
func GetDomainName(searchDomains []string) string {
	selected := ""
	for _, d := range searchDomains {
		if len(d) > len(selected) {
			selected = d
		}
	}
	return selected
}
