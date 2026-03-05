/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package conntrack

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Wire format: [version:byte][data_len:uint32][data:bytes]
type SyncMessage struct {
	Version byte
	Data    []byte
}

func (m *SyncMessage) Encode() []byte {
	dataLen := len(m.Data)
	buf := make([]byte, 1+4+dataLen)

	buf[0] = m.Version

	binary.BigEndian.PutUint32(buf[1:5], uint32(dataLen))
	copy(buf[5:], m.Data)

	return buf
}

func DecodeSyncMessage(r io.Reader) (*SyncMessage, error) {
	var version byte
	if err := binary.Read(r, binary.BigEndian, &version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	var dataLen uint32
	if err := binary.Read(r, binary.BigEndian, &dataLen); err != nil {
		return nil, fmt.Errorf("failed to read data length: %w", err)
	}

	data := make([]byte, dataLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	return &SyncMessage{
		Version: version,
		Data:    data,
	}, nil
}
