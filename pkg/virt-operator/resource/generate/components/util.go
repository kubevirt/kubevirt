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
 *
 */

package components

import "fmt"

func generateValueDifferenceInHourQuery(metricsToSum string) string {
	return generateValueDifferenceInIntervalQuery(metricsToSum, 60)
}
func generateValueDifferenceInFiveMinutesQuery(metricsToSum string) string {
	return generateValueDifferenceInIntervalQuery(metricsToSum, 5)
}
func generateValueDifferenceInIntervalQuery(metricsToSum string, timeIntervalInMinutes int) string {
	return fmt.Sprintf("clamp_min(sum (%s) - sum(%s offset %dm), 0)", metricsToSum, metricsToSum, timeIntervalInMinutes)
}
func generateAllPodRequestsQuery(podName string, ns string, codeRegex string) string {
	if codeRegex == "" {
		return fmt.Sprintf("rest_client_requests_total{pod=~'%s-.*', namespace='%s'}", podName, ns)
	}
	return fmt.Sprintf("rest_client_requests_total{pod=~'%s-.*', namespace='%s', code=~'%s'}", podName, ns, codeRegex)
}
