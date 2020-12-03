# fd

Package fd provides a simple API to pass file descriptors
between different OS processes.

It can be useful if you want to inherit network connections
from another process without closing them.

Example scenario:

 * Running server receives a "let's upgrade" message
 * Server opens a Unix domain socket for the "upgrade"
 * Server starts a new copy of itself and passes Unix domain
   socket name
 * New copy starts reading data from the socket
 * Server sends its state over the socket, also sending the number
   of network connections to inherit, then it sends those connections
   using fd.Put()
 * New server copy reads the state and inherits connections using fd.Get(),
   checks that everything is OK and writes an "OK" message to the socket
 * Server receives "OK" message and kills itself

## Documentation

[fd on godoc.org](http://godoc.org/github.com/ftrvxmtrx/fd)
