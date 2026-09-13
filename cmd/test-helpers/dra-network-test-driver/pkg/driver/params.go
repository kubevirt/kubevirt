package driver

import (
	"encoding/json"
	"fmt"

	resourcev1 "k8s.io/api/resource/v1"
)

func getOpaqueParams(claim *resourcev1.ResourceClaim, targetDriver string) (map[string]string, error) {
	for _, config := range claim.Spec.Devices.Config {
		if config.Opaque == nil || config.Opaque.Driver != targetDriver {
			continue
		}

		rawBytes := config.Opaque.Parameters.Raw
		if len(rawBytes) == 0 {
			continue
		}

		params := make(map[string]string)
		if err := json.Unmarshal(rawBytes, &params); err != nil {
			return nil, fmt.Errorf("failed to parse driver parameters: %w", err)
		}

		return params, nil
	}

	return nil, fmt.Errorf("no configuration found for driver %s", targetDriver)
}
