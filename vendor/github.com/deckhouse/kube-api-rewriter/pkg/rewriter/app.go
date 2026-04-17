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

import "github.com/tidwall/gjson"

const (
	DeploymentKind      = "Deployment"
	DeploymentListKind  = "DeploymentList"
	DaemonSetKind       = "DaemonSet"
	DaemonSetListKind   = "DaemonSetList"
	StatefulSetKind     = "StatefulSet"
	StatefulSetListKind = "StatefulSetList"
)

func RewriteDeploymentOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, DeploymentListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteSpecTemplateLabelsAnno(rules, singleObj, "spec", action)
	})
}

func RewriteDaemonSetOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, DaemonSetListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteSpecTemplateLabelsAnno(rules, singleObj, "spec", action)
	})
}

func RewriteStatefulSetOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, StatefulSetListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteSpecTemplateLabelsAnno(rules, singleObj, "spec", action)
	})
}

func RenameSpecTemplatePatch(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RenameMetadataPatch(rules, obj)
	if err != nil {
		return nil, err
	}

	return TransformPatch(obj, func(mergePatch []byte) ([]byte, error) {
		return RewriteSpecTemplateLabelsAnno(rules, mergePatch, "spec", Rename)
	}, func(jsonPatch []byte) ([]byte, error) {
		path := gjson.GetBytes(jsonPatch, "path").String()
		if path == "/spec" {
			return RewriteSpecTemplateLabelsAnno(rules, jsonPatch, "value", Rename)
		}
		return jsonPatch, nil
	})
}

// RewriteSpecTemplateLabelsAnno transforms labels and annotations in spec fields:
// - selector as LabelSelector
// - template.metadata.labels as labels map
// - template.metadata.annotations as annotations map
// - template.affinity as Affinity
// - template.nodeSelector as labels map.
func RewriteSpecTemplateLabelsAnno(rules *RewriteRules, obj []byte, path string, action Action) ([]byte, error) {
	return TransformObject(obj, path, func(obj []byte) ([]byte, error) {
		obj, err := RewriteLabelsMap(rules, obj, "template.metadata.labels", action)
		if err != nil {
			return nil, err
		}
		obj, err = RewriteLabelsMap(rules, obj, "selector.matchLabels", action)
		if err != nil {
			return nil, err
		}
		obj, err = RewriteLabelsMap(rules, obj, "template.spec.nodeSelector", action)
		if err != nil {
			return nil, err
		}
		obj, err = RewriteAffinity(rules, obj, "template.spec.affinity", action)
		if err != nil {
			return nil, err
		}
		return RewriteAnnotationsMap(rules, obj, "template.metadata.annotations", action)
	})
}
