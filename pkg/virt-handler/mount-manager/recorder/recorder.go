package recorder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE
type MountTargetEntry struct {
	TargetFile string `json:"targetFile"`
	SocketFile string `json:"socketFile,omitempty"`
}

type vmiMountTargetRecord struct {
	MountTargetEntries []MountTargetEntry `json:"mountTargetEntries"`
	UsesSafePaths      bool               `json:"usesSafePaths"`
}

func readRecordFile(recordFile string) (*vmiMountTargetRecord, error) {
	record := &vmiMountTargetRecord{}
	// #nosec No risk for path injection. Using static base and cleaned filename
	bytes, err := os.ReadFile(recordFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func writeRecordFile(recordFile string, entries []MountTargetEntry) error {
	r := vmiMountTargetRecord{
		MountTargetEntries: entries,
		// XXX: backward compatibility for old unresolved paths, can be removed in July 2023
		// After a one-time convert and persist, old records are safe too.
		UsesSafePaths: true,
	}
	bytes, err := json.Marshal(r)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(recordFile), 0750)
	if err != nil {
		return err
	}

	return os.WriteFile(recordFile, bytes, 0600)
}

func deleteRecordFiles(recordFile string, entries []MountTargetEntry) error {
	exists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return err
	}

	if exists {
		for _, target := range entries {
			os.Remove(target.TargetFile)
			os.Remove(target.SocketFile)
		}

		os.Remove(recordFile)
	}

	return nil
}

type MountRecorder interface {
	SetMountRecord(vmi *v1.VirtualMachineInstance, entries []MountTargetEntry) error
	AddMountRecord(vmi *v1.VirtualMachineInstance, entries []MountTargetEntry) error
	GetMountRecord(vmi *v1.VirtualMachineInstance) ([]MountTargetEntry, error)
	DeleteMountRecord(vmi *v1.VirtualMachineInstance) error
	DeleteMountRecordEntry(vmi *v1.VirtualMachineInstance, idxToRemove int) error
}

type mounter struct {
	mountStateDir    string
	mountRecords     map[types.UID]*vmiMountTargetRecord
	mountRecordsLock sync.Mutex
}

func NewMountRecorder(mountStateDir string) MountRecorder {
	return &mounter{
		mountStateDir: mountStateDir,
		mountRecords:  make(map[types.UID]*vmiMountTargetRecord),
	}
}

func (m *mounter) SetMountRecord(vmi *v1.VirtualMachineInstance, entries []MountTargetEntry) error {
	return m.setAddMountRecord(vmi, entries, false)
}

func (m *mounter) AddMountRecord(vmi *v1.VirtualMachineInstance, entries []MountTargetEntry) error {
	return m.setAddMountRecord(vmi, entries, true)
}

func (m *mounter) GetMountRecord(vmi *v1.VirtualMachineInstance) ([]MountTargetEntry, error) {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return nil, err
	}
	return record.MountTargetEntries, nil
}

func (m *mounter) DeleteMountRecord(vmi *v1.VirtualMachineInstance) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("cannot find the mount record without the VMI uid")
	}

	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}

	recordFile := filepath.Join(m.mountStateDir, filepath.Clean(string(vmi.UID)))
	if err := deleteRecordFiles(recordFile, record.MountTargetEntries); err != nil {
		return err
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	delete(m.mountRecords, vmi.UID)

	return nil
}

func (m *mounter) DeleteMountRecordEntry(vmi *v1.VirtualMachineInstance, idxToRemove int) error {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}
	record.MountTargetEntries = removeSliceElement(record.MountTargetEntries, idxToRemove)
	return nil
}

func (m *mounter) getMountTargetRecord(vmi *v1.VirtualMachineInstance) (*vmiMountTargetRecord, error) {
	var ok bool
	var existingRecord *vmiMountTargetRecord

	if string(vmi.UID) == "" {
		return nil, fmt.Errorf("unable to find container disk mounted directories for vmi without uid")
	}

	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()
	existingRecord, ok = m.mountRecords[vmi.UID]

	// first check memory cache
	if ok {
		return existingRecord, nil
	}

	recordFile := filepath.Join(m.mountStateDir, filepath.Clean(string(vmi.UID)))
	exists, err := diskutils.FileExists(recordFile)
	if err != nil {
		return nil, err
	}

	if exists {
		record, err := readRecordFile(recordFile)
		if err != nil {
			return nil, err
		}

		// XXX: backward compatibility for old unresolved paths, can be removed in July 2023
		// After a one-time convert and persist, old records are safe too.
		if !record.UsesSafePaths {
			record.UsesSafePaths = true
			for i, entry := range record.MountTargetEntries {
				safePath, err := safepath.JoinAndResolveWithRelativeRoot("/", entry.TargetFile)
				if err != nil {
					return nil, fmt.Errorf("failed converting legacy path to safepath: %v", err)
				}
				record.MountTargetEntries[i].TargetFile = unsafepath.UnsafeAbsolute(safePath.Raw())
			}
		}

		m.mountRecords[vmi.UID] = record
		return record, nil
	}

	// not found
	return &vmiMountTargetRecord{}, nil
}

func (m *mounter) setMountTargetRecord(vmi *v1.VirtualMachineInstance, record *vmiMountTargetRecord) error {
	if string(vmi.UID) == "" {
		return fmt.Errorf("unable to find mounted directories for vmi without uid")
	}
	m.mountRecordsLock.Lock()
	defer m.mountRecordsLock.Unlock()

	recordFile := filepath.Join(m.mountStateDir, filepath.Clean(string(vmi.UID)))
	if err := writeRecordFile(recordFile, record.MountTargetEntries); err != nil {
		return err
	}

	m.mountRecords[vmi.UID] = record
	return nil
}

func (m *mounter) setAddMountRecord(vmi *v1.VirtualMachineInstance, entries []MountTargetEntry, add bool) error {
	record, err := m.getMountTargetRecord(vmi)
	if err != nil {
		return err
	}

	if add {
		record.MountTargetEntries = append(record.MountTargetEntries, entries...)
	} else {
		record.MountTargetEntries = entries
	}

	return m.setMountTargetRecord(vmi, record)
}

func removeSliceElement(s []MountTargetEntry, idxToRemove int) []MountTargetEntry {
	// removes slice element efficiently
	s[idxToRemove] = s[len(s)-1]
	return s[:len(s)-1]
}
