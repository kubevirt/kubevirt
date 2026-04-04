/*
Copyright The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vm

import (
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
)

const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.MustParse(magicUUID)

// CalculateLegacyUUID calculates the firmware UUID using the legacy name-based
// (UUID v5/SHA-1) algorithm. This is kept temporarily for backward compatibility
// during the transition period and will be removed once all legacy callers are cleaned up.
func CalculateLegacyUUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(name)).String())
}
