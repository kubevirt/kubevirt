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
 * Copyright 2021
 *
 */

package launchsecurity

import (
	"encoding/xml"
	"fmt"
)

type Features struct {
	SEV SEVConfiguration `xml:"features>sev"`
}

type SEVConfiguration struct {
	Supported       string `xml:"supported,attr"`
	Cbitpos         string `xml:"cbitpos"`
	ReducedPhysBits string `xml:"reducedPhysBits"`
}

func QuerySEVConfiguration(virsh Virsh) (*SEVConfiguration, error) {
	out, err := virsh.Domcapabilities()
	if err != nil {
		return nil, fmt.Errorf("failed to query domain capabilities: %v", err)
	}
	features := &Features{}
	if err := xml.Unmarshal(out, features); err != nil {
		return nil, fmt.Errorf("failed to parse domain capabilities: %v", err)
	}
	return &features.SEV, nil
}
