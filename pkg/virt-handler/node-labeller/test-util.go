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
 * Copyright 2021 Red Hat, Inc.
 */

package nodelabeller

import (
	"io/ioutil"
)

var (
	features = []string{"apic", "clflush", "cmov"}
)

const (
	domainCapabilities = `<domainCapabilities>
  <cpu>
    <mode name='host-passthrough' supported='yes'/>
    <mode name='host-model' supported='yes'>
      <model fallback='allow'>Skylake-Client-IBRS</model>
      <vendor>Intel</vendor>
      <feature policy='require' name='ds'/>
      <feature policy='require' name='acpi'/>
      <feature policy='require' name='ss'/>
    </mode>
    <mode name='custom' supported='yes'>
      <model usable='no'>EPYC-IBPB</model>
      <model>fake-model-without-usable</model>
      <model usable='no'>486</model>
      <model usable='no'>Conroe</model>
      <model usable='yes'>Penryn</model>
      <model usable='yes'>IvyBridge</model>
      <model usable='yes'>Haswell</model>
    </mode>
  </cpu>
</domainCapabilities>`

	hostSupportedFeatures = `<cpu mode='custom' match='exact'>
  <model fallback='forbid'>Skylake-Client-IBRS</model>
  <vendor>Intel</vendor>
  <feature policy='require' name='apic'/>
  <feature policy='require' name='clflush'/>
  <feature policy='require' name='vmx'/>
  <feature policy='require' name='xsaves'/>
</cpu>`

	domainCapabilitiesNothingUsable = `<domainCapabilities>
  <cpu>
    <mode name='host-passthrough' supported='yes'/>
    <mode name='host-model' supported='yes'>
      <model fallback='allow'>Skylake-Client-IBRS</model>
      <vendor>Intel</vendor>
      <feature policy='require' name='ds'/>
      <feature policy='require' name='acpi'/>
      <feature policy='require' name='ss'/>
    </mode>
    <mode name='custom' supported='yes'>
      <model usable='no'>EPYC-IBPB</model>
      <model>fake-model-without-usable</model>
      <model usable='no'>486</model>
      <model usable='no'>Conroe</model>
      <model usable='no'>coreduo</model>
      <model usable='no'>IvyBridge</model>
      <model usable='no'>Haswell</model>
    </mode>
  </cpu>
</domainCapabilities>`

	cpuModelPenrynFeatures = `<cpus>
<model name='Penryn'>
  <signature family='6' model='23'/>
  <vendor name='Intel'/>
  <feature name='apic'/>
  <feature name='clflush'/>
</model>
</cpus>`
)

func writeMockDataFile(path, data string) error {
	return ioutil.WriteFile(path, []byte(data), 0644)
}
