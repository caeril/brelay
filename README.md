# brelay

## BRelay - A Buffering Reverse Proxy in Go

NGINX and HAProxy are often far too heavy to install, configure, and maintain for simple use cases.

All I needed was a simple buffering reverse proxy with interval retry semantics, so I fugured I'd write one.

Very early stages, so all this supports right now is:

 - Basic GET and POST forwarding
 - Static configuration

 
