#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2024 Red Hat, Inc.

# Scale down the virt-operator deployment so it won't override the patch deployed later
cluster-up/kubectl.sh --namespace kubevirt scale deployment virt-operator --replicas=0
cluster-up/kubectl.sh --namespace kubevirt wait --for=delete pod -l kubevirt.io=virt-operator

# Patch the virt-controller deployment mainly to add a dlv container and use the debug image
cluster-up/kubectl.sh --namespace kubevirt patch deployment virt-controller --type='json' -p '[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/-",
    "value": {
      "name": "dlv-debugger",
      "image": "golang:1.23-alpine",
      "command": [
        "sh",
        "-c",
        "apk add --no-cache git bash && go install github.com/go-delve/delve/cmd/dlv@latest && \
        dlv attach $(pgrep virt-controller) --headless --accept-multiclient --api-version 2 --listen=:2345"
      ],
      "securityContext": {
        "seccompProfile": {
          "type": "Unconfined"
        },
        "runAsUser": 0
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/shareProcessNamespace",
    "value": true
  },
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/0/image",
    "value": "registry:5000/kubevirt/virt-controller:debug"
  },
  {
    "op": "replace",
    "path": "/spec/replicas",
    "value": 1
  },
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/0/imagePullPolicy",
    "value": "Always"
  },
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/0/securityContext",
    "value": {
      "runAsUser": 0
    }
  },
  {
    "op": "replace",
    "path": "/spec/template/spec/securityContext",
    "value": {}
  },
  {
    "op": "remove",
    "path": "/spec/template/spec/containers/0/readinessProbe"
  },
  {
    "op": "remove",
    "path": "/spec/template/spec/containers/0/livenessProbe"
  }
]'

