package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
)

var defaultDataDir = "/var/lib/cni/networks"

type Value struct {
	IP  net.IP `json:"ip"`
	MAC string `json:"mac"`
}

type Store struct {
	dataDir string
}

func NewFileStore(dataDir string, network string) (*Store, error) {
	if dataDir == "" {
		dataDir = defaultDataDir
	}

	dir := filepath.Join(dataDir, network)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	return &Store{dataDir: dir}, nil
}

func (s *Store) Save(id string, ip net.IP, mac string) error {
	val := Value{MAC: mac, IP: ip}
	buf, err := json.MarshalIndent(&val, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling value: %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(s.dataDir, id), buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing to store for ID %s: %v", id, err)
	}
	return nil
}

func (s *Store) Load(id string) (*Value, error) {
	buf, err := ioutil.ReadFile(filepath.Join(s.dataDir, id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	val := Value{}
	err = json.Unmarshal(buf, &val)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling value: %v", err)
	}
	return &val, nil
}

func (s *Store) Delete(id string) error {
	return os.RemoveAll(filepath.Join(s.dataDir, id))
}
