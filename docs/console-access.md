Console access for virtual machines
===================================

There are two main ways tenant users may interact with virtual machines they
are running.

 * Guest OS hosted network services. This covers SSH server, web apps like
   cockpit, Windows remote desktop. The set of services available varies
   according to what OS is installed, and can only be accessed if the guest
   is running "normally".

 * Host OS hosted network services. This covers VNC, SPICE, and serial port
   console services exposed over the host network by the QEMU emulator. These
   are available as long as the virtual machine is powered on, regardless of
   whether the guest OS is running "normally" or not. The set of services
   available is the same regardless of what kind of guest OS is installed.

Access to guest OS hosted network services does not require any special work
by kubevirt aside from ensuring the guest has a working network connection to
the tenant. This document thus considers how to expose the host OS hosted
network services to tenant users.


Proxied vs direct access
------------------------

The VNC, SPICE or serial console network service exposed by QEMU can be made
to listen on either a local-only UNIX domain socket, or a remotely accessible
TCP socket. These choices in QEMU configuration, lead to a number of different
options for how a tenant user can be allowed to connect.

If the QEMU network service listens on a UNIX domain socket, then there will
need to be a proxy service running on each compute node that exposes the
service over TCP for remote access. There are several options for proxying, but
the common approach is to use either a traditional HTTP CONNECT based proxy, or
a HTTP websockets encoded proxy. In such a scenario, the compute host will only
need to expose a single TCP port, no matter how many virtual machines are
running. If tenant users cannot access IP addresses in the cluster network, then
the POD running the proxy on each host will need an associated service IP
address, to which tenant users will connect.

If the QEMU network service listens on a TCP socket, then each virtual machine
running on a compute host will be listening on the same IP address, but using a
different port number. For example, if 100 virtual machines are running with VNC
enabled, TCP ports 5900->5999 will be used. The IP address(es) used will be the
those associated with one (or more) of the networks available to the libvirtd
daemon. If libvirtd is running in a POD, then the IP address will be associated
with the cluster network.

If tenant users can directly access IP addresses in the cluster network, then
they can connect directly to the TCP ports exposed by QEMU on the compute
nodes. More likely though, there will need to be a proxy in between the compute
host and the tenant user. The only difference from the UNIX domain socket
scenario is that the proxy is not running on each compute host. One or more
proxy services can be run in the cluster, multi-plexing to a much larger number
of compute hosts.


Network Security
----------------

With the prevalence in recent years of network attacks & breaches across a wide
variety of companies, the overriding assumption is that the network between the
compute host and tenant users is hostile. Even an "internal" network may have
unknown/undetected malicious parties present attacking services & traffic.
Thus while VNC, SPICE & serial console services exposed by QEMU can all run in
plain text mode, it is intended that all deployments of kubevirt will
unconditionally enable TLS with x509 certificates.

At the compute host level this means that x509 certificates need to be issued
to each host. Libvirt needs to be configured to pass these certificates to QEMU
when enabling SPICE, VNC or serial console services. For VNC and serial consoles
it is also possible to configure QEMU to require that clients provide their own
certificates, giving mutual authentication between the client & server.

From QEMU's POV, the client may not necessarily be the tenant user. If access
to the console is proxied, then the client is the internal proxy service. This
opens up different possibilities for certificate management within the cluster.

If a HTTP CONNECT proxy is used, the address that the client validates
against the certificate is still that of the compute node it is ultimately
connecting to. There is thus end-to-end encryption between the QEMU service
and the tenant user. A downside is that information about compute nodes is
leaked to the tenant user. It also means that every single compute node has
to be issued with a certificate that is signed by a CA that the tenant user
already trusts. This places an administrative burden on the cloud admin to
acquire certificates from an externally managed CA for each new compute node
they wish to deploy, with potentially significant financial cost incurred.

If a HTTP(s) websockets proxy is used, the address that the client validates
against the certificate is that of the proxy node. The websockets proxy is
acting as a man-in-the-middle between the tenant user and the compute node.
MITM proxies are generally undesirable when dealing with public internet
web access, because the administrator of the proxy is less trustworthy
than the administrator of the remote web site. In the case of kubevirt, the
administrator of the proxy is the same as the administrator of QEMU on the
compute nodes. Thus there is no trust mismatch for the tenant user. The cloud
administrator can secure their internal infrastructure using a private CA that
they directly manage with little overhead or cost, independent of the public CA
system. Communications between the proxy & compute host use certificates from
this private CA. The proxy will need to have a certificate issued by a public
CA for securing communications with the tenant user. When a single proxy can
service multiple compute hosts, this means that the number of certificates
required from the public CA is fairly small, and unlikely to grow over time as
compute nodes are added.


Network authentication
----------------------

As well as encrypting the data transport with TLS, there is a need to be able
to authenticate clients connecting to the network services.

As the network transport level, VNC, SPICE and serial console services can all
be configured to require certificates from the client. This can be used as an
indirect authentication mechanism, if QEMU is configured to trust a particular
CA that is only used for issuing certificates for QEMU network services. This
likely precludes use of public CAs, and requires a private CA model, which in
turn forces the adoption of the HTTP websockets proxy which can act as an active
MITM. QEMU will eventually gain support for setting access control lists on the
network services, so individual VMs can white list individual client certificates
instead of having to trust every certificate from a CA.

The SPICE protocol provides a choice between two authentication mechanisms,
either a simple shared secret mechanism, or a pluggable SASL mechanism. The
latter can delegate to a diverse range of auth systems, including GSSAPI with
Kerberos, SCRAM-SHA-1 and PAM (which in turn opens up many more pluggable
options).

Traditionally VNC servers have supported a crude shared secret mechanism but
this is woefully insecure by modern standards. QEMU implements a VNC extension
based on SASL which delegates to the same set of options as SPICE SASL support.

The serial console service exposed by QEMU has no explicit authentication
options. It relies entirely on the network transport level to secure access.

When a proxy is used in between the tenant user and QEMU service, it is possible
to provide further authentication mechanisms at the network transport level, that
are protocol agnostic. The general concept is that each network service exposed
by a VM has an associated shared secret.

In a HTTP websockets proxy, the tenant user provides the shared secret in a
cookie when connecting to the proxy. The proxy uses the cookie to identify which
virtual machine to connect to, by doing a lookup against a table mapping secrets
to hostname + port number pairs. Thus the secret serves as both an identification
and authentication mechanism. It is still possible to pass through the protocol
specific authentication mechanism to the tenant user, but more likely is that
the protocol specific authentication mechanism would only be used between the
proxy and the QEMU service, if at all. It would be sufficient to rely on client
certificates for authentication between the proxy and QEMU.

In a HTTP CONNECT proxy, the tenant can provide the shared secret in a cookie as
with a websockets proxy, but it will only be used for authentication. The VM is
still identified by the address associated with the CONNECT verb. Alternatively
the shared secret can be used as a pseudo hostname with the CONNECT verb. In this
case the secret is used to lookup the real hostname + port of the QEMU service,
and thus acts as both an identification and authentication mechanism. The latter
has negative implications for x509 certificate validation though, because the
pseudo-hostname used for CONNECT must match the address encoded in the cert. Thus
a new cert would be required each time the shared secret changes, which is
impractical.

When using shared secrets with a HTTP CONNECT proxy, the connection between the
tenant user and the proxy must also be protected with TLS, otherwise the secret
can be intercepted. This would in turn mean that the console payload is double-
encrypted, once with TLS for the proxy connection, and again for the content
tunnelled over the proxy. This is bad for performance of both the proxy nodes
and tenant user clients. This makes it undesirable to use the shared secret auth
with HTTP CONNECT proxies. It is better to delegate all authentication to the
protocol level, which allows the connection to proxy to remain in clear text,
and rely on encryption of the tunnelled traffic instead. Until QEMU gains support
for fine grained access control based on client certs, this only works for the
SPICE and VNC services, since serial consoles have no native support for
direct authentication.

The shared secret used for proxy authentication can be treated as a long term
persistent token, a time limited token, or a use limited (one-time) token. If
the token is use limited, there are some complications with SPICE. A single SPICE
connection from the tenant user's POV is actually multiple TCP connections at the
network level. A 'single use' token should be scoped to the logical connection,
not the network connection(s). This implies that the proxy has to MITM the SPICE
network connections to identify the primary connection and associate with the
secondary connections, so it can permit reuse of the token for later secondary
connections, but not primary connections. Further SPICE connections can be
established on the fly at any time (eg to dynamically enable USB device
tunnelling), so even time limited tokens may need this MITM support, to enable
further secondary connections to be established even though the token may have
expired.


Virtual machine migration
-------------------------

When a virtual machine is migrated from one compute host to another, this would
ordinarily interrupt the console connection for any connected tenant. The SPICE
protocol has built-in support enable a so called "seamless" migration mode. When
the source QEMU starts the migration, it informs the connected client of the
IP+port of the SPICE service run by the target QEMU. At the end of the migration,
the tenant's client is able to automatically reconnect to the new QEMU. At worst
interaction is frozen for a few 100 milliseconds, but should continue thereafter.

With a HTTP CONNECT proxy, seamless migration should still work without changes,
since the client will open a new connection to the same proxy, and provide the
new IP+port to the CONNECT verb.

With a HTTP websockets proxy, the proxy server will need to intercept the message
containing the new migration target information, since that is a private IP+port
that the tenant user doesn't know about. It would have to issue a new message to
the tenant that just contains the IP+port of the proxy itself, to cause the client
to just drop + reconnect to the proxy, which should in turn make the proxy
connect to the new QEMU

Neither VNC or serial consoles have any protocol support for seamless migrations.
If those are used with a HTTP CONNECT proxy, the tenant user will need to acquire
the new IP+port of the target QEMU out of band from the same place they acquired
the info for the initial connection. With a HTTP websockets proxy, the tenant
user can simply try to reconnect when the initial connection is lost, which will
cause the proxy to connect to the new target QEMU instance. There could be some
complications, however, around authentication tokens if using time limited or
use limited shared secrets. This would require the tenant to acquire the updated
token. If the token is still valid, or token acquisition is automatable out of
band, then both VNC and serial consoles can be close to seamless with the
websockets proxy, without needing protocol support. It might be preferable to
just let SPICE work the same way instead of trying to fake migration reconnect
messages.


Supported deployment scenarios
------------------------------

Given the trade-offs described above the following would be considered
supportable deployment options:

### 1. No proxy

QEMU on the compute node running a TCP socket with TLS and x509 certificates
enabled.

Compute node certs must be issued by a CA that is trusted by all tenant
users and new certs acquired for each new compute node.

Tenant users make direct connections to the IP address and port of QEMU on
the cluster network.

Tenant users directly validate the received server certificates against
the requested IP + port of the QEMU service.

Choice of access control methods. Either QEMU can mandate client certificates
and perform access control checks on the certificate identity (planned, but not
yet supported in QEMU & libvirt), or protocol specific authentication must be
used.

Only supports VNC and SPICE, until QEMU is able to do fine grained ACL checks
against client certs.


### 2. HTTP CONNECT proxy

QEMU on the compute node running a TCP socket with TLS and x509 certificates
enabled.

Compute node certs must be issued by a CA that is trusted by all tenant
users and new certs acquired for each new compute node.

One or more proxy servers able to multiplex to an arbitrary number of backend
compute nodes

Tenant users make HTTP CONNECT request to proxy server, giving the IP address
and port on the cluster network that QEMU is listening on

Tenant users directly validate the received server certificates against
the requested IP + port of the QEMU service.

Choice of access control methods. Either QEMU can mandate client certificates
and perform access control checks on the certificate identity (planned, but not
yet supported in QEMU & libvirt), or protocol specific authentication must be
used.

Only supports VNC and SPICE, until QEMU is able to do fine grained ACL checks
against client certs.


### 3. HTTP websockets proxy

QEMU on the compute node running a TCP socket with TLS and x509 certificates
enabled.

Compute node certs can be issued by a private CA, which only needs to be
trusted by the proxy servers. Proxies are issued client certs by the CA
for mutual auth.

One or more proxy servers able to multiplex to an arbitrary number of backend
compute nodes

Shared secret between the proxy and tenant user identifies the QEMU service to
connect to and authenticates the user.

Tenant users validate the received server certificates against the IP + port
of the proxy server.

The proxy server performs TLS handshake with the QEMU servers and validates its
certificate.

Protocol specific authentication is not exposed to tenant users.

Supports VNC, SPICE and serial consoles with parity of functionality.


Proxy Implementation
--------------------

For both the HTTP CONNECT and HTTP websockets proxy there needs to be some be some
intelligence in the proxy. In the HTTP CONNECT proxy, there needs to be code to
validate that the requested IP+port pair actually corresponds to a valid guest
console connection. This protects against the proxy being used to attack other
arbitrary network servers running in the cluster network. In the HTTP websockets
proxy, there needs to be code to validate the shared secret obtained from the
client, to resolve it into a IP+port for the QEMU connection. It also needs to
be prepared to do some intelligent MITM of the connections to handle authentication
and migration.

The libvirt-console-proxy project aims to provide a general purpose server that
can support both HTTP CONNECT or HTTP websockets proxies, switchable based on
config options. It intends to provide all the necessary protocol specific glue
to do active MITM of VNC & SPICE to console authentication processes. To integrate
this with kubevirt, requires provision of a simple REST service to resolve valid
shared secret tokens to IP+port pairs, or to validate a tenant provided IP+port
pair.

For the HTTP CONNECT proxy, kubevirt would need to provide some mechanism to
configure the protocol specific authentication parameters. For example, to provide
the SPICE token, or provide a username+password for SASL with VNC/SPICE. For the
HTTP websockets proxy, kubevirt would need to provide some mechanism to configure
the shared secret for authenticating with the proxy, along with a way to update
it on the fly for time/use limited tokens. This would likely all be accomplished
via the VM spec document format.

When deploying libvirtd on the compute nodes, there further needs to be a way
to create new x509 certificates, or acquire them from a 3rd party. The libvirt
QEMU driver will need to be configured to enable use of these certificates too.
