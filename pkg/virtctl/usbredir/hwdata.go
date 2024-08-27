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
 * Copyright the KubeVirt Authors.
 *
 */

package usbredir

import (
	"bufio"
	"strings"
)

func MetadataLookup(hwdata, vendor, product string) (string, string, bool) {
	vendorName := ""
	scanner := bufio.NewScanner(strings.NewReader(hwdata))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, vendor) {
			vendorName = line[6:]
			break
		}
	}

	if vendorName == "" {
		return "", "", false
	}

	productName := "unidentified"
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			// Ignore possible comments
			continue
		}

		if !strings.HasPrefix(line, "\t") {
			// Another vendor
			break
		}

		if strings.HasPrefix(line[1:], product) {
			productName = line[7:]
			break
		}
	}

	return vendorName, productName, true
}
