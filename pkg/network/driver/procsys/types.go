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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package procsys

type ArpReplyMode int

// arp_ignore - INTEGER
//
//	Define different modes for sending replies in response to
//	received ARP requests that resolve local target IP addresses:
//	0 - (default): reply for any local target IP address, configured
//	on any interface
//	1 - reply only if the target IP address is local address
//	configured on the incoming interface
//	2 - reply only if the target IP address is local address
//	configured on the incoming interface and both with the
//	sender's IP address are part from same subnet on this interface
//	3 - do not reply for local addresses configured with scope host,
//	only resolutions for global and link addresses are replied
//	4-7 - reserved
//	8 - do not reply for all local addresses
//
//	The max value from conf/{all,interface}/arp_ignore is used
//	when ARP request is received on the {interface}
//
// Ref: https://www.kernel.org/doc/Documentation/networking/ip-sysctl.txt
const (
	ARPReplyMode0 ArpReplyMode = iota
	ARPReplyMode1
	ARPReplyMode2
	ARPReplyMode3
	ARPReplyMode8 = 8
)
