package cache

import v1 "kubevirt.io/client-go/api/v1"

type PodCacheInterface struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
}
