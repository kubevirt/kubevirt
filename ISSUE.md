<!-- Make sure that you visit our User Guide at https://kubevirt.io/user-guide.
-->

**Is your feature request related to a problem? Please describe**:
There are 2 ways in which the virt-handler is unnecessarily aware of the specifics of the Guest Agent:

1. Currently, the VirtualMachineController of the virt-handler needs to be aware of how the Guest Agent is connected to the Libvirt daemon via the 
`org.qemu.guest_agent.0` channel. This detail should be confined to virt-launcher, because it deals with how the GA is connected to Libvirt, which is entirely within the scope of the virt-launcher.

2. Virt-Handler needs to be aware of all the required commands that need to be supported by the GA. These commands are in fact executed by components in the virt-launcher only, e.g., the Agent Poller. So there is no reason for this logic to remain within virt-handler.

**Describe the solution you'd like**:

Solution for problem 1: The return type of GetGuestInfo() in the Command gRPC API needs to be expanded to include the connectivity status of the GA.

Solution for problem 2: This is more complex. A few additional functions are expected from the GA if the VMI uses the QemuGuestAgent propagation method for user credentials. This information would need to be available in the Command gRPC server for it to respond with whether the GA is supported for the VMI. Then there is also the problem of checking whether the GA version is in `c.clusterConfig.GetSupportedAgentVersions()`, although this logic is deprecated. Otherwise, this functionality would have to remain in the virt-handler.

The following changes need to be made:

 - Add fields: gaConnected, gaSupported, gaNotSupportedReason to the GuestInfoResponse data structure.
 - Replace the input argument to function GetGuestInfo from EmptyRequest to VmiRequest, so that the VMI can be passed, along with VirtualMachineOptions, which will contain the supportedGuestAgentVersions.


**Describe alternatives you've considered**:
A clear and concise description of any alternative solutions or features you've considered.

**Additional context**:
Add any other context or screenshots about the feature request here.

