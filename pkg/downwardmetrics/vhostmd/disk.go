package vhostmd

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	"kubevirt.io/kubevirt/pkg/util"
)

const fileSize = 262144
const maxBodyLength = fileSize - 24

var signature = [4]byte{'m', 'v', 'b', 'd'}

type vhostmd struct {
	filePath string
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

func (d *Disk) Verify() error {
	var checksum int32
	for _, b := range d.Raw {
		checksum = checksum + int32(b)
	}
	if d.Header.Flag > 0 {
		return fmt.Errorf("file is locked")
	}
	if checksum != d.Header.Checksum {
		return fmt.Errorf("checksum is %v, but expected %v", checksum, d.Header.Checksum)
	}
	return nil
}

func (d *Disk) Metrics() (*api.Metrics, error) {
	m := &api.Metrics{}
	if err := xml.Unmarshal(d.Raw, m); err != nil {
		return nil, err
	}
	m.Text = strings.TrimSpace(m.Text)
	for i, metric := range m.Metrics {
		m.Metrics[i].Name = strings.TrimSpace(metric.Name)
		m.Metrics[i].Type = api.MetricType(strings.TrimSpace(string(metric.Type)))
		m.Metrics[i].Context = api.MetricContext(strings.TrimSpace(string(metric.Context)))
		m.Metrics[i].Value = strings.TrimSpace(metric.Value)
		m.Metrics[i].Text = strings.TrimSpace(metric.Text)
	}
	return m, nil
}

func (v *vhostmd) Create() error {
	return createDisk(v.filePath)
}

func (v *vhostmd) Read() (*api.Metrics, error) {
	disk, err := readDisk(v.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load vhostmd file: %v", err)
	}
	if err := disk.Verify(); err != nil {
		return nil, fmt.Errorf("failed to verify vhostmd file: %v", err)
	}
	return disk.Metrics()
}

func (v *vhostmd) Write(metrics *api.Metrics) (err error) {
	f, err := os.OpenFile(v.filePath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open vhostmd disk: %v", err)
	}
	defer func() {
		if fileErr := f.Close(); fileErr != nil && err == nil {
			err = fileErr
		}
	}()
	if err := writeDisk(f, metrics); err != nil {
		return fmt.Errorf("failed to write metrics: %v", err)
	}
	return nil
}

func readDisk(filePath string) (*Disk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	// If the read operation succeeds, but close fails, we have already read the data,
	// so it is ok to not return the error.
	defer util.CloseIOAndCheckErr(f, nil)

	d := &Disk{
		Header: &Header{},
	}
	if err = binary.Read(f, binary.BigEndian, d.Header); err != nil {
		return nil, err
	}

	if d.Header.Flag == 0 {
		if d.Header.Length > maxBodyLength {
			return nil, fmt.Errorf("Invalid metrics file. Expected a maximum body length of %v, got %v", maxBodyLength, d.Header.Length)
		}

		d.Raw = make([]byte, d.Header.Length, d.Header.Length)

		if _, err = io.ReadFull(f, d.Raw); err != nil {
			return nil, err
		}
	}
	return d, err
}

func createDisk(filePath string) (err error) {
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
