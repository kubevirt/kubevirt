go-spice [![GoDoc](https://godoc.org/github.com/jsimonetti/go-spice?status.svg)](https://godoc.org/github.com/jsimonetti/go-spice) [![Go Report Card](https://goreportcard.com/badge/github.com/jsimonetti/go-spice)](https://goreportcard.com/report/github.com/jsimonetti/go-spice)
=======

Package `spice` attempts to implement a SPICE proxy.
It can be used to proxy virt-viewer/remote-viewer traffic to destination qemu instances.
Using this proxy over a HTML5 based web viewer has many advantages. One being, the native remote-viewer
client can be used through this proxy. This allows (for example) USB redirection, sound playback and recording
and clipboard sharing to function.

This package is mostly finished except for the below mentioned todo's. The API should be stable.
Vendoring this package is still advised in any case.

TODO:
- implement proper auth capability handling
- implement SASL authentication (Not planned, but nice to have)


See [example](https://godoc.org/github.com/jsimonetti/go-spice#Proxy) for an example including an Authenticator