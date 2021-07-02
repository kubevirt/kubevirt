# RDP into a VirtualMachineInstance

Every VM and VMI provides a `/portforward` subresource that can be used to create a websocket backed
network tunnel to a port inside the instance similar to Kubernetes pods.

One use-case for this subresource is to forward RDP traffic into the VMI either from the CLI
or a web-UI.

## Usage

To connect to a Windows Guest via RDP, first open a `port-forward` tunnel:

```sh
virtctl port-forward vm/win10 udp/3389 tcp/3389
```

Then you can use the tunnel with an RDP client of your preference:

```sh
freerdp /u:Administrator /p:YourPassword /v:127.0.0.1:3389
```
