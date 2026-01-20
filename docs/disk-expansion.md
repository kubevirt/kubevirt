# Disk expansion

For some storage methods, Kubernetes may support expanding storage in-use (allowVolumeExpansion feature).
KubeVirt can respond to it by making the additional storage available for the virtual machines.
This feature is currently on by default.

This feature does two things:
- Notify the virtual machine about size changes
- If the disk is a Filesystem PVC, the matching file is expanded to the remaining size (while reserving some space for file system overhead).
