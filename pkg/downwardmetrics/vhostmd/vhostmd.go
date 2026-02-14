package vhostmd

func NewMetricsIODisk(filePath string) *vhostmd {
	return &vhostmd{filePath: filePath}
}
