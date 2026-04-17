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
	ValidatingWebhookConfigurationKind     = "ValidatingWebhookConfiguration"
	ValidatingWebhookConfigurationListKind = "ValidatingWebhookConfigurationList"
	MutatingWebhookConfigurationKind       = "MutatingWebhookConfiguration"
	MutatingWebhookConfigurationListKind   = "MutatingWebhookConfigurationList"
)

func RewriteValidatingOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	if action == Rename {
		return RewriteResourceOrList(obj, ValidatingWebhookConfigurationListKind, func(singleObj []byte) ([]byte, error) {
			return RewriteArray(singleObj, "webhooks", func(webhook []byte) ([]byte, error) {
				return RewriteArray(webhook, "rules", func(item []byte) ([]byte, error) {
					return RenameResourceRule(rules, item)
				})
			})
		})
	}
	return RewriteResourceOrList(obj, ValidatingWebhookConfigurationListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteArray(singleObj, "webhooks", func(webhook []byte) ([]byte, error) {
			return RewriteArray(webhook, "rules", func(item []byte) ([]byte, error) {
				return RestoreResourceRule(rules, item)
			})
		})
	})
}

func RewriteMutatingOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	if action == Rename {
		return RewriteResourceOrList(obj, MutatingWebhookConfigurationListKind, func(singleObj []byte) ([]byte, error) {
			return RewriteArray(singleObj, "webhooks", func(webhook []byte) ([]byte, error) {
				return RewriteArray(webhook, "rules", func(item []byte) ([]byte, error) {
					return RenameResourceRule(rules, item)
				})
			})
		})
	}
	return RewriteResourceOrList(obj, MutatingWebhookConfigurationListKind, func(singleObj []byte) ([]byte, error) {
		return RewriteArray(singleObj, "webhooks", func(webhook []byte) ([]byte, error) {
			return RewriteArray(webhook, "rules", func(item []byte) ([]byte, error) {
				return RestoreResourceRule(rules, item)
			})
		})
	})
}

func RenameWebhookConfigurationPatch(rules *RewriteRules, obj []byte) ([]byte, error) {
	obj, err := RenameMetadataPatch(rules, obj)
	if err != nil {
		return nil, err
	}

	return TransformPatch(obj, func(mergePatch []byte) ([]byte, error) {
		return RewriteArray(mergePatch, "webhooks", func(webhook []byte) ([]byte, error) {
			return RewriteArray(webhook, "rules", func(item []byte) ([]byte, error) {
				return RestoreResourceRule(rules, item)
			})
		})
	}, func(jsonPatch []byte) ([]byte, error) {
		path := gjson.GetBytes(jsonPatch, "path").String()
		if path == "/webhooks" {
			return RewriteArray(jsonPatch, "value", func(webhook []byte) ([]byte, error) {
				return RewriteArray(webhook, "rules", func(item []byte) ([]byte, error) {
					return RenameResourceRule(rules, item)
				})
			})
		}
		return jsonPatch, nil
	})
}
