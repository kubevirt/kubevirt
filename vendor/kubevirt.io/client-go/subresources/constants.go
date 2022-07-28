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
 * Copyright 2018 The KubeVirt Authors
 *
 */

package subresources

// PlainStreamProtocolName is a subprotocol which indicates a plain websocket stream.
// Mostly useful for browser connections which need to use the websocket subprotocol
// field to pass credentials. As a consequence they need to get a subprotocol back.
const PlainStreamProtocolName = "plain.kubevirt.io"
