# Ticket Assessment: CNV-71612

## Ticket Identity

- **Key:** CNV-71612
- **Type:** Epic
- **Summary:** Restore: Predictable PVC name
- **Status:** In Progress
- **Priority:** Major
- **Assignee:** Danny Sanatar
- **Components:** Storage Platform
- **Labels:** ga

## Assessment Questions

### 1. Is the WHAT clear?

**Yes.** The ticket clearly defines the feature: users can choose PVC names
for each Volume using overrides when restoring a VM from a snapshot. Users
can also override Annotations and Labels that are normally copied from the
source PVC.

### 2. Is the WHAT TO TEST clear?

**Partially.** Two user stories are provided:

1. Restore VM from snapshot without changing PVC names
2. Choose specific PVC names when restoring from snapshot

However, edge cases are not specified:
- What happens with duplicate PVC names?
- What if the chosen name exceeds Kubernetes naming limits?
- Are there validation rules for the override values?
- How do Annotations/Labels overrides interact with existing values?

### 3. Can the pipeline trace code?

**Yes.** The Storage Platform component maps to known packages in
kubevirt/kubevirt. The restore/snapshot APIs are well-documented in
the KubeVirt API surface.

## Verdict: PARTIAL

The feature intent is clear and testable, but edge case behavior and
validation rules would benefit from additional specification. The
pipeline can proceed with the available information.
