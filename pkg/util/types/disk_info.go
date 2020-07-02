package types

type DiskInfo struct {
	Format      string `json:"format"`
	BackingFile string `json:"backing-filename"`
	ActualSize  int    `json:"actual-size"`
	VirtualSize int    `json:"virtual-size"`
}
