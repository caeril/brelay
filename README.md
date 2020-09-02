# brelay

## BRelay - A Buffering Reverse Proxy in Go

NGINX and HAProxy are often far too heavy to install, configure, and maintain for simple use cases.

All I needed was a simple buffering reverse proxy with interval retry semantics, so I figured I'd write one.

Very early stages, so all this supports right now is:

 - Basic GET, POST, PUT, and DELETE forwarding
 - JSON configuration
 - Basic round-robin
 - Static retry interval (1 second)
 - Filesystem logging with stdout fallback
 - Basic TLS support

 Todo:

  - Allocate buffers from sync.Pool - right now we're creating way too much garbage
  - Expanded logging backends
  - Configurable retry intervals, timeouts, etc.






 
