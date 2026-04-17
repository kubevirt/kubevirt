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

package proxy

// Proxy handler implements 2 types of proxy:
// - proxy for client interaction with Kubernetes API Server
// - proxy to deliver AdmissionReview requests from Kubernetes API Server to webhook server
//
// Proxy for webhooks acts as follows:
// ServerHTTP method reads request from Kubernetes API Server, restores apiVersion, kind and
// ownerRefs, sends it to real webhook, renames apiVersion, kind, and ownerRefs
// and sends it back to Kubernetes API Server.
//
//	+--------------------------------------------+
//	|           Kubernetes API Server            |
//	+--------------------------------------------+
//		  |                              ^
//		  |                              |
//	1. AdmissionReview request        4. AdmissionReview response
//	webhook.srv:443/webhook-endpoint     |
//	apiVersion: renamed-group.io         |
//	kind: PrefixedResource               |
//		 |                               |
//		 v                               |
//	+-----------------------------------------------------+
//	| Proxy                                               |
//	| 2. Restore               3. Rename                  |
//	| apiVersion, kind field     if Admission response    |
//	| in Admission request       has patchType: JSONPatch |
//	| in Admission request       rename kind in ownerRef  |
//	+-----------------------------------------------------+
//		|                                      ^
//		127.0.0.1:9443/webhook-endpoint        |
//		apiVersion: original-group.io          |
//		kind: Resource                         |
//		|                                      |
//		v                                      |
//	+-------------------------------------------------------+
//	| Webhook                                               |
//	|  handles request      --->        sends response      |
//	+-------------------------------------------------------+
