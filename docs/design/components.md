# Components

KubeVirt consists of a set of services:

                |
        Cluster | (virt-api-server)  (virt-controller)
                | (vm-pod [M])
                |
    ------------+---------------------------------------
                |
     Kubernetes | (VM [TPR])
                |
    ------------+---------------------------------------
                |
      DaemonSet | (libvirtd) (vms-handler)
                |

    M: Managed
    TPR: Third Party Resource


## Example flow: Create VM

    User       Virt API      VM TPR    Virt Controller    k8s      VM Pod
    createVM  --> |                           |            |
                  o create --> +              |            |
                  |            | <~ create ~> ?            |
                  |            |              |            |
                  |            |              o create --> o ------> +
                  |            |              |            |         |
                  |            | <---------------------- get VM Spec o
                  |            o ----------------------------------> |
                  |            |              |            |         o defineVM()
                  |            |              |            |         o watchVM()
                  |            |              |            |         |
    deleteVM -->  o delete --> *              |            |         |
                  |              <~ delete ~> ?            |
                  |                           |            |         |
                  |                           o delete --> o ------> *
                  |                           |            |
                  :                           :            :
    
    Legend: ?: Event notification


## Virt API Server

HTTP API server which serves as the endpoint for all virtualization related flows.

## Virt Controller

Takes care of the VM entities life-times.

## VM State

Repository of all VM definitions and, if running, their current states.

## VM Pod: VM Launcher, VM Handler

Every VM is getting a dedicated pod. Inside each pod, the vm launcher is responsible for bootstrapping the VM.

The vm handler is then responsible to perform operations on this VM during it’s life-cycle.

## WIP - Storage Controller

WIP - Interface to high-level storage entities/functionality

## WIP - Network Controller

WIP - Interface to high-level storage entities/functionality

## Libvirt

Libvirtd is used on every host to run VMs
