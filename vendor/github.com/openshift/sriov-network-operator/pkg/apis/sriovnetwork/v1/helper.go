package v1

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	netattdefv1 "github.com/openshift/sriov-network-operator/pkg/apis/k8s/v1"
	render "github.com/openshift/sriov-network-operator/pkg/render"
)

const (
	MANIFESTS_PATH       = "./bindata/manifests/cni-config"
	LASTNETWORKNAMESPACE = "operator.sriovnetwork.openshift.io/last-network-namespace"
	FINALIZERNAME        = "netattdef.finalizers.sriovnetwork.openshift.io"
)

const invalidVfIndex = -1

var SriovPfVfMap = map[string](string){
	"1583": "154c",
	"158b": "154c",
	"10fb": "10ed",
	"1015": "1016",
	"1017": "1018",
}

var VfIds = []string{}

func init() {
	for _, v := range SriovPfVfMap {
		id := "0x" + v
		if !StringInArray(id, VfIds) {
			VfIds = append(VfIds, id)
		}
	}
}

var log = logf.Log.WithName("sriovnetwork")

type ByPriority []SriovNetworkNodePolicy

func (a ByPriority) Len() int {
	return len(a)
}

func (a ByPriority) Less(i, j int) bool {
	if a[i].Spec.Priority != a[j].Spec.Priority {
		return a[i].Spec.Priority > a[j].Spec.Priority
	}
	return a[i].GetName() < a[j].GetName()
}

func (a ByPriority) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Match check if node is selected by NodeSelector
func (p *SriovNetworkNodePolicy) Selected(node *corev1.Node) bool {
	for k, v := range p.Spec.NodeSelector {
		if nv, ok := node.Labels[k]; ok && nv == v {
			continue
		}
		return false
	}
	log.Info("Selected():", "node", node.Name)
	return true
}

func StringInArray(val string, array []string) bool {
	for i := range array {
		if array[i] == val {
			return true
		}
	}
	return false
}

func RemoveString(s string, slice []string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func UniqueAppend(inSlice []string, strings ...string) []string {
	for _, s := range strings {
		if !StringInArray(s, inSlice) {
			inSlice = append(inSlice, s)
		}
	}
	return inSlice
}

// Apply policy to SriovNetworkNodeState CR
func (p *SriovNetworkNodePolicy) Apply(state *SriovNetworkNodeState, merge bool) {
	s := p.Spec.NicSelector
	if s.Vendor == "" && s.DeviceID == "" && len(s.RootDevices) == 0 && len(s.PfNames) == 0 {
		// Empty NicSelector match none
		return
	}
	for _, iface := range state.Status.Interfaces {
		if s.Selected(&iface) {
			log.Info("Update interface", "name:", iface.Name)
			result := Interface{
				PciAddress: iface.PciAddress,
				Mtu:        p.Spec.Mtu,
				Name:       iface.Name,
				LinkType:   p.Spec.LinkType,
			}
			var group *VfGroup
			if p.Spec.NumVfs > 0 {
				result.NumVfs = p.Spec.NumVfs
				group, _ = p.generateVfGroup(&iface)
				found := false
				for i := range state.Spec.Interfaces {
					if state.Spec.Interfaces[i].PciAddress == result.PciAddress {
						found = true
						// merge PF configurations when:
						// 1. SR-IOV partition is configured
						// 2. SR-IOV partition policies have the same priority
						result = state.Spec.Interfaces[i].mergePfConfigs(result, merge)
						result.VfGroups = state.Spec.Interfaces[i].mergeVfGroups(group)
						state.Spec.Interfaces[i] = result
						break
					}
				}
				if !found {
					result.VfGroups = []VfGroup{*group}
					state.Spec.Interfaces = append(state.Spec.Interfaces, result)
				}
			}
		}
	}
}

func (iface Interface) mergePfConfigs(input Interface, merge bool) Interface {
	if merge {
		if input.Mtu < iface.Mtu {
			input.Mtu = iface.Mtu
		}
		if input.NumVfs < iface.NumVfs {
			input.NumVfs = iface.NumVfs
		}
	}
	return input
}

func (iface Interface) mergeVfGroups(input *VfGroup) []VfGroup {
	groups := iface.VfGroups
	for i := range groups {
		if groups[i].ResourceName == input.ResourceName {
			groups[i] = *input
			return groups
		}
	}
	groups = append(groups, *input)
	return groups
}

func (p *SriovNetworkNodePolicy) generateVfGroup(iface *InterfaceExt) (*VfGroup, error) {
	var err error
	pfName := ""
	var rngStart, rngEnd int
	found := false
	for _, selector := range p.Spec.NicSelector.PfNames {
		pfName, rngStart, rngEnd, err = ParsePFName(selector)
		if err != nil {
			return nil, err
		}
		if pfName == iface.Name {
			found = true
			if rngStart == invalidVfIndex && rngEnd == invalidVfIndex {
				rngStart, rngEnd = 0, p.Spec.NumVfs-1
			}
			break
		}
	}
	if !found {
		// assign the default vf index range if the pfName is not specified by the nicSelector
		rngStart, rngEnd = 0, p.Spec.NumVfs-1
	}
	rng := strconv.Itoa(rngStart) + "-" + strconv.Itoa(rngEnd)
	return &VfGroup{
		ResourceName: p.Spec.ResourceName,
		DeviceType:   p.Spec.DeviceType,
		VfRange:      rng,
		PolicyName:   p.GetName(),
	}, nil
}

func IndexInRange(i int, r string) bool {
	rngSt, rngEnd, err := parseRange(r)
	if err != nil {
		return false
	}
	if i <= rngEnd && i >= rngSt {
		return true
	}
	return false
}

func parseRange(r string) (rngSt, rngEnd int, err error) {
	rng := strings.Split(r, "-")
	rngSt, err = strconv.Atoi(rng[0])
	if err != nil {
		return
	}
	rngEnd, err = strconv.Atoi(rng[1])
	if err != nil {
		return
	}
	return
}

// Parse PF name with VF range
func ParsePFName(name string) (ifName string, rngSt, rngEnd int, err error) {
	rngSt, rngEnd = invalidVfIndex, invalidVfIndex
	if strings.Contains(name, "#") {
		fields := strings.Split(name, "#")
		ifName = fields[0]
		rngSt, rngEnd, err = parseRange(fields[1])
	} else {
		ifName = name
	}
	return
}

func (selector *SriovNetworkNicSelector) Selected(iface *InterfaceExt) bool {
	if selector.Vendor != "" && selector.Vendor != iface.Vendor {
		return false
	}
	if selector.DeviceID != "" && selector.DeviceID != iface.DeviceID {
		return false
	}
	if len(selector.RootDevices) > 0 && !StringInArray(iface.PciAddress, selector.RootDevices) {
		return false
	}
	if len(selector.PfNames) > 0 {
		var pfNames []string
		for _, p := range selector.PfNames {
			if strings.Contains(p, "#") {
				fields := strings.Split(p, "#")
				pfNames = append(pfNames, fields[0])
			} else {
				pfNames = append(pfNames, p)
			}
		}
		if !StringInArray(iface.Name, pfNames) {
			return false
		}
	}
	return true
}

func (s *SriovNetworkNodeState) GetInterfaceStateByPciAddress(addr string) *InterfaceExt {
	for _, iface := range s.Status.Interfaces {
		if addr == iface.PciAddress {
			return &iface
		}
	}
	return nil
}

func (s *SriovNetworkNodeState) GetDriverByPciAddress(addr string) string {
	for _, iface := range s.Status.Interfaces {
		if addr == iface.PciAddress {
			return iface.Driver
		}
	}
	return ""
}

// RenderNetAttDef renders a net-att-def for ib-sriov CNI
func (cr *SriovIBNetwork) RenderNetAttDef() (*uns.Unstructured, error) {
	logger := log.WithName("renderNetAttDef")
	logger.Info("Start to render IB SRIOV CNI NetworkAttachementDefinition")
	var err error
	objs := []*uns.Unstructured{}

	// render RawCNIConfig manifests
	data := render.MakeRenderData()
	data.Data["CniType"] = "ib-sriov"
	data.Data["SriovNetworkName"] = cr.Name
	if cr.Spec.NetworkNamespace == "" {
		data.Data["SriovNetworkNamespace"] = cr.Namespace
	} else {
		data.Data["SriovNetworkNamespace"] = cr.Spec.NetworkNamespace
	}
	data.Data["SriovCniResourceName"] = os.Getenv("RESOURCE_PREFIX") + "/" + cr.Spec.ResourceName

	data.Data["StateConfigured"] = true
	switch cr.Spec.LinkState {
	case "enable":
		data.Data["SriovCniState"] = "enable"
	case "disable":
		data.Data["SriovCniState"] = "disable"
	case "auto":
		data.Data["SriovCniState"] = "auto"
	default:
		data.Data["StateConfigured"] = false
	}

	if cr.Spec.Capabilities == "" {
		data.Data["CapabilitiesConfigured"] = false
	} else {
		data.Data["CapabilitiesConfigured"] = true
		data.Data["SriovCniCapabilities"] = cr.Spec.Capabilities
	}

	if cr.Spec.IPAM != "" {
		data.Data["SriovCniIpam"] = "\"ipam\":" + strings.Join(strings.Fields(cr.Spec.IPAM), "")
	} else {
		data.Data["SriovCniIpam"] = "\"ipam\":{}"
	}

	objs, err = render.RenderDir(MANIFESTS_PATH, &data)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		raw, _ := json.Marshal(obj)
		logger.Info("render NetworkAttachementDefinition output", "raw", string(raw))
	}
	return objs[0], nil
}

// DeleteNetAttDef deletes the generated net-att-def CR
func (cr *SriovIBNetwork) DeleteNetAttDef(c client.Client) error {
	// Fetch the NetworkAttachmentDefinition instance
	instance := &netattdefv1.NetworkAttachmentDefinition{}
	namespace := cr.GetNamespace()
	if cr.Spec.NetworkNamespace != "" {
		namespace = cr.Spec.NetworkNamespace
	}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: cr.GetName()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = c.Delete(context.TODO(), instance)
	if err != nil {
		return err
	}
	return nil
}

// RenderNetAttDef renders a net-att-def for sriov CNI
func (cr *SriovNetwork) RenderNetAttDef() (*uns.Unstructured, error) {
	logger := log.WithName("renderNetAttDef")
	logger.Info("Start to render SRIOV CNI NetworkAttachementDefinition")
	var err error
	objs := []*uns.Unstructured{}

	// render RawCNIConfig manifests
	data := render.MakeRenderData()
	data.Data["CniType"] = "sriov"
	data.Data["SriovNetworkName"] = cr.Name
	if cr.Spec.NetworkNamespace == "" {
		data.Data["SriovNetworkNamespace"] = cr.Namespace
	} else {
		data.Data["SriovNetworkNamespace"] = cr.Spec.NetworkNamespace
	}
	data.Data["SriovCniResourceName"] = os.Getenv("RESOURCE_PREFIX") + "/" + cr.Spec.ResourceName
	data.Data["SriovCniVlan"] = cr.Spec.Vlan

	if cr.Spec.VlanQoS <= 7 && cr.Spec.VlanQoS >= 0 {
		data.Data["VlanQoSConfigured"] = true
		data.Data["SriovCniVlanQoS"] = cr.Spec.VlanQoS
	} else {
		data.Data["VlanQoSConfigured"] = false
	}

	if cr.Spec.Capabilities == "" {
		data.Data["CapabilitiesConfigured"] = false
	} else {
		data.Data["CapabilitiesConfigured"] = true
		data.Data["SriovCniCapabilities"] = cr.Spec.Capabilities
	}

	data.Data["SpoofChkConfigured"] = true
	switch cr.Spec.SpoofChk {
	case "off":
		data.Data["SriovCniSpoofChk"] = "off"
	case "on":
		data.Data["SriovCniSpoofChk"] = "on"
	default:
		data.Data["SpoofChkConfigured"] = false
	}

	data.Data["TrustConfigured"] = true
	switch cr.Spec.Trust {
	case "on":
		data.Data["SriovCniTrust"] = "on"
	case "off":
		data.Data["SriovCniTrust"] = "off"
	default:
		data.Data["TrustConfigured"] = false
	}

	data.Data["StateConfigured"] = true
	switch cr.Spec.LinkState {
	case "enable":
		data.Data["SriovCniState"] = "enable"
	case "disable":
		data.Data["SriovCniState"] = "disable"
	case "auto":
		data.Data["SriovCniState"] = "auto"
	default:
		data.Data["StateConfigured"] = false
	}

	data.Data["MinTxRateConfigured"] = false
	if cr.Spec.MinTxRate != nil {
		if *cr.Spec.MinTxRate >= 0 {
			data.Data["MinTxRateConfigured"] = true
			data.Data["SriovCniMinTxRate"] = *cr.Spec.MinTxRate
		}
	}

	data.Data["MaxTxRateConfigured"] = false
	if cr.Spec.MaxTxRate != nil {
		if *cr.Spec.MaxTxRate >= 0 {
			data.Data["MaxTxRateConfigured"] = true
			data.Data["SriovCniMaxTxRate"] = *cr.Spec.MaxTxRate
		}
	}

	if cr.Spec.IPAM != "" {
		data.Data["SriovCniIpam"] = "\"ipam\":" + strings.Join(strings.Fields(cr.Spec.IPAM), "")
	} else {
		data.Data["SriovCniIpam"] = "\"ipam\":{}"
	}

	objs, err = render.RenderDir(MANIFESTS_PATH, &data)
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		raw, _ := json.Marshal(obj)
		logger.Info("render NetworkAttachementDefinition output", "raw", string(raw))
	}
	return objs[0], nil
}

// DeleteNetAttDef deletes the generated net-att-def CR
func (cr *SriovNetwork) DeleteNetAttDef(c client.Client) error {
	// Fetch the NetworkAttachmentDefinition instance
	instance := &netattdefv1.NetworkAttachmentDefinition{}
	namespace := cr.GetNamespace()
	if cr.Spec.NetworkNamespace != "" {
		namespace = cr.Spec.NetworkNamespace
	}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: cr.GetName()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = c.Delete(context.TODO(), instance)
	if err != nil {
		return err
	}
	return nil
}
