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
				searchDomains = append(searchDomains, dom)
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
