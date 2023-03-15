package cloudinit

import "encoding/json"

type IsoCreationFunc func(isoOutFile, volumeID string, inDir string) error

var cloudInitIsoFunc = defaultIsoFunc

type DataSourceType string
type DeviceMetadataType string

const (
	DataSourceNoCloud     DataSourceType     = "noCloud"
	DataSourceConfigDrive DataSourceType     = "configDrive"
	NICMetadataType       DeviceMetadataType = "nic"
	HostDevMetadataType   DeviceMetadataType = "hostdev"
)

// CloudInitData is a data source independent struct that
// holds cloud-init user and network data
type CloudInitData struct {
	DataSource          DataSourceType
	NoCloudMetaData     *NoCloudMetadata
	ConfigDriveMetaData *ConfigDriveMetadata
	UserData            string
	NetworkData         string
	DevicesData         *[]DeviceData
	VolumeName          string
}

type PublicSSHKey struct {
	string
}

type NoCloudMetadata struct {
	InstanceType  string `json:"instance-type,omitempty"`
	InstanceID    string `json:"instance-id"`
	LocalHostname string `json:"local-hostname,omitempty"`
}

type ConfigDriveMetadata struct {
	InstanceType  string            `json:"instance_type,omitempty"`
	InstanceID    string            `json:"instance_id"`
	LocalHostname string            `json:"local_hostname,omitempty"`
	Hostname      string            `json:"hostname,omitempty"`
	UUID          string            `json:"uuid,omitempty"`
	Devices       *[]DeviceData     `json:"devices,omitempty"`
	PublicSSHKeys map[string]string `json:"public_keys,omitempty"`
}

type DeviceData struct {
	Type        DeviceMetadataType `json:"type"`
	Bus         string             `json:"bus"`
	Address     string             `json:"address"`
	MAC         string             `json:"mac,omitempty"`
	Serial      string             `json:"serial,omitempty"`
	NumaNode    uint32             `json:"numaNode,omitempty"`
	AlignedCPUs []uint32           `json:"alignedCPUs,omitempty"`
	Tags        []string           `json:"tags"`
}

type legacyConfigDriveMetadataFields struct {
	InstanceType  string `json:"instance-type,omitempty"`
	InstanceID    string `json:"instance-id"`
	LocalHostname string `json:"local-hostname,omitempty"`
}

func (c ConfigDriveMetadata) MarshalJSON() ([]byte, error) {
	// its important to have alias type to stop MarshalJSON recursion
	type aliasConfigDriveMetadata ConfigDriveMetadata
	return json.Marshal(&struct {
		*legacyConfigDriveMetadataFields
		aliasConfigDriveMetadata
	}{
		legacyConfigDriveMetadataFields: &legacyConfigDriveMetadataFields{
			InstanceID:    c.InstanceID,
			InstanceType:  c.InstanceType,
			LocalHostname: c.LocalHostname,
		},
		aliasConfigDriveMetadata: (aliasConfigDriveMetadata)(c),
	})
}
