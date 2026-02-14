package vhostmd

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"os"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	"kubevirt.io/kubevirt/pkg/safepath"
)

const fileSize = 262144
const maxBodyLength = fileSize - 24

var signature = [4]byte{'m', 'v', 'b', 'd'}

type vhostmd struct {
	filePath *safepath.Path
}

type Header struct {
	Signature [4]byte
	Flag      int32
	Checksum  int32
	Length    int32
}

type Disk struct {
	Header *Header
	Raw    []byte
}

func (d *Disk) String() string {
	return fmt.Sprintf("%v:%v:%v:%v", string(d.Header.Signature[:]), d.Header.Flag, d.Header.Checksum, d.Header.Length)
}

func (v *vhostmd) Write(metrics *api.Metrics) (err error) {
	f, err := safepath.OpenAtNoFollow(v.filePath)
	if err != nil {
		return fmt.Errorf("failed to open vhostmd disk: %v", err)
	}
	defer func() {
		if fileErr := f.Close(); fileErr != nil && err == nil {
			err = fileErr
		}
	}()
	file, err := os.OpenFile(f.SafePath(), os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open vhostmd disk: %v", err)
	}
	defer func() {
		if fileErr := file.Close(); fileErr != nil && err == nil {
			err = fileErr
		}
	}()
	if err := writeDisk(file, metrics); err != nil {
		return fmt.Errorf("failed to write metrics: %v", err)
	}
	return nil
}

func CreateDisk(filePath string) (err error) {
	var f *os.File

	if f, err = os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0755); err != nil {
		return fmt.Errorf("failed getting vhostmd disk filestats: %v", err)
	}
	defer func() {
		if fileErr := f.Close(); fileErr != nil && err == nil {
			err = fileErr
		}
	}()

	_, err = f.Seek(fileSize-1, 0)
	if err != nil {
		return fmt.Errorf("preallocating vhostmd disk failed: %v", err)
	}
	_, err = f.Write([]byte{0})
	if err != nil {
		return fmt.Errorf("preallocating vhostmd disk failed: %v", err)
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("moving back to file start failed: %v", err)
	}
	return writeDisk(f, &api.Metrics{})
}

func writeDisk(file *os.File, m *api.Metrics) (err error) {
	d := emptyLockedDisk()
	if d.Raw, err = xml.MarshalIndent(m, "", "  "); err != nil {
		return fmt.Errorf("failed to encode metrics: %v", err)
	}
	// Add newline, since `vm-dump-metrics` does not append a newline when writing to metrics
	d.Raw = append(d.Raw, '\n')

	if len(d.Raw) > maxBodyLength {
		return fmt.Errorf("vhostmd metrics body is too big, expected a maximum of %v, got %v", maxBodyLength, len(d.Raw))
	}
	var checksum int32
	for _, b := range d.Raw {
		checksum = checksum + int32(b)
	}
	d.Header.Checksum = checksum
	d.Header.Length = int32(len(d.Raw))

	if err = binary.Write(file, binary.BigEndian, d.Header); err != nil {
		return fmt.Errorf("failed to write vhostmd header: %v", err)
	}

	if err = file.Sync(); err != nil {
		return fmt.Errorf("failed to flush to vhostmd file, when trying to lock it: %v", err)
	}

	if _, err = file.Write(d.Raw); err != nil {
		return fmt.Errorf("failed to write vhostmd body: %v", err)
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("moving back to file start failed: %v", err)
	}
	d.Header.Flag = 0
	if err = binary.Write(file, binary.BigEndian, d.Header); err != nil {
		return fmt.Errorf("failed to unlock vhostmd file: %v", err)
	}
	return nil
}

func emptyLockedDisk() *Disk {
	return &Disk{
		Header: &Header{
			Signature: signature,
			Flag:      1,
			Checksum:  0,
			Length:    0,
		},
	}
}
