# Upgrading KubeVirt

This document outlines the step by step process of upgrading kubevirt
from the perspective of the kubevirt-operator.  These steps can also
be done by hand.

**Major Upgrade** - Upgrade from 1.0 to 2.0
**Minor Upgrade** - Upgrade from 1.0 to 1.1

### Major Upgrade
*kubevirt-1.1 to kubevirt-2.0*

*When the kubevirt-operator sees the Virt CR change from version 1.1 to 2.0 it will...*
 - Rollout kubevirt-2.0 virt-handler daemonsets
 - Test virt-handler functionality with kubevirt-2.0 data
 - Scale down virt-controllers pods
 - Scale down virt-api pods
 - Register kubevirt-2.0 CRD webhooks
 - Update to kubevirt-2.0 CRDs
 - Rollout kubevirt-2.0 virt-controllers
 - Test virt-controller functionality with kubevirt-2.0 data
 - Rollout kubevirt-2.0 virt-api
 - Test virt-api functionality with kubevirt-2.0 data


### Minor Upgrade
*kubevirt-1.1 to kubevirt-1.2*

*When the kubevirt-operator sees the Virt CR change from version 1.1 to 1.2 it will…*
 - Rollout kubevirt-1.2 virt-handler daemonsets
 - Test virt-handler functionality with kubevirt-1.1 data
 - Rolling update of virt-controllers pods
 - Test virt-controller functionality with kubevirt-1.1 data
 - Rolling update of virt-api pods
 - Test virt-api functionality with kubevirt-1.1 data
 - Register kubevirt-1.2 CRD webhooks
 - Update to kubevirt-1.2 CRDs
 - Test virt-controller and virt-api functionality with kubevirt- 1.1 and kubevirt-1.2 data

### Virt-launcher Upgrade
Upgrading virt-launcher will require the VM to be shutdown and rescheduled. In
order to mitigate vm interuptions, virt-launcher will be forwards compatible
with:
(release x.y)
 - all minor kubevirt versions (>= y)
 - one major version (x + 1)

Once migrations are production-ready, additional compatibility restrictions may
be introduced.
