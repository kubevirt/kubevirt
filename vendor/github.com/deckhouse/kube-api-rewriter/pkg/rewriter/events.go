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

const (
	EventKind     = "Event"
	EventListKind = "EventList"
)

// RewriteEventOrList rewrites a single Event resource or a list of Events in EventList.
// The only field need to rewrite is involvedObject:
//
//	 {
//	  "metadata": { "name": "...", "namespace": "...", "managedFields": [...] },
//	  "involvedObject": {
//	    "kind": "SomeResource",
//	    "namespace": "name",
//	    "name": "ns",
//	    "uid": "a260fe4f-103a-41c6-996c-d29edb01fbbd",
//	    "apiVersion": "group.io/v1"
//	  },
//	  "type": "...",
//	  "reason": "...",
//	  "message": "...",
//	  "source": {
//	    "component": "...",
//	    "host": "..."
//	  },
//	  "reportingComponent": "...",
//	  "reportingInstance": "..."
//	},
func RewriteEventOrList(rules *RewriteRules, obj []byte, action Action) ([]byte, error) {
	return RewriteResourceOrList(obj, EventListKind, func(singleObj []byte) ([]byte, error) {
		return TransformObject(singleObj, "involvedObject", func(involvedObj []byte) ([]byte, error) {
			return RewriteAPIVersionAndKind(rules, involvedObj, action)
		})
	})
}
