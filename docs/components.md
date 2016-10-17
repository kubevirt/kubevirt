# Components

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
