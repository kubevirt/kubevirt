/*
Copyright 2024 Flant JSC

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

package rewriter

import (
	"github.com/tidwall/gjson"
)

const (
	PodKind                       = "Pod"
	PodListKind                   = "PodList"
	ServiceKind                   = "Service"
	ServiceListKind               = "ServiceList"
	JobKind                       = "Job"
	JobListKind                   = "JobList"
	PersistentVolumeClaimKind     = "PersistentVolumeClaim"
	PersistentVolumeClaimListKind = "PersistentVolumeClaimList"
)

func RewritePodOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, PodListKind, func(singleObj []byte) ([]byte, error) {
		singleObj, err := RewriteLabelsMap(rules, singleObj, "spec.nodeSelector", action)
		if err != nil {
			return nil, err
		}
		return RewriteAffinity(rules, singleObj, "spec.affinity", action)
	})
}

func RewriteServiceOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, ServiceListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteLabelsMap(rules, singleObj, "spec.selector", action)
	})
}

// RewriteJobOrList transforms known fields in the Job manifest.
func RewriteJobOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, JobListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteSpecTemplateLabelsAnno(rules, singleObj, "spec", action)
	})
}

// RewritePVCOrList transforms known fields in the PersistentVolumeClaim manifest.
func RewritePVCOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, PersistentVolumeClaimListKind, func(singleObj []byte) ([]byte, error) {
		singleObj, err := TransformObject(singleObj, "spec.dataSource", func(specDataSource []byte) ([]byte, error) {
			return RewriteAPIGroupAndKind(rules, specDataSource, action)
		})
		if err != nil {
			return nil, err
		}
		return TransformObject(singleObj, "spec.dataSourceRef", func(specDataSourceRef []byte) ([]byte, error) {
			return RewriteAPIGroupAndKind(rules, specDataSourceRef, action)
		})
	})
}

func RenameServicePatch(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RenameMetadataPatch(rules, obj)
	if err != nil {
		return nil, err
	}

	// Also rename patch on spec field.
	return TransformPatch(obj, nil, func(jsonPatch []byte) ([]byte, error) {
		path := gjson.GetBytes(jsonPatch, "path").String()
		switch path {
		case "/spec":
			return RewriteLabelsMap(rules, jsonPatch, "value.selector", Rename)
		}
		return jsonPatch, nil
	})
}
