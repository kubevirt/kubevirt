# DHCP4 - A DHCP library written in Go.

## Original Author
http://richard.warburton.it/

## Quick Start
See example_test.go for how to use this library to create a basic server.

## Documentation
http://godoc.org/github.com/krolaw/dhcp4

## Thanks
Special thanks to:
* https://github.com/pietern for suggesting how to use go.net
to be able to listen on a single network interface.
* https://github.com/fdurand for proper interface binding on linux. 

## Wow
DHCP4 was one of the libraries used by Facebook's [DHCP load balancing relay](https://github.com/facebookincubator/dhcplb/tree/7f3b3859478a4f19a15984d97c96fceaa89e982b).  "Facebook currently uses it in production, and it's deployed at global scale across all of our data centers." FB has since moved to another lib.
