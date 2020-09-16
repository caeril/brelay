package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/caeril/brelay/config"
	"github.com/caeril/brelay/logging"
	"github.com/valyala/fasthttp"
)

type Header struct {
	Key   []byte
	Value []byte
}

type OriginResponse struct {
	StatusCode int
	Body       []byte
	Headers    []Header
}

type OriginRequest struct {
	Verb     string
	Uri      string
	ClientIP string
	Headers  []Header
	Body     []byte
}

func getFromOrigin(backend config.Backend, or OriginRequest) (OriginResponse, bool) {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(fmt.Sprintf("http://%s:%d%s", backend.Hostname, backend.Port, or.Uri))

	if or.Verb == "POST" || or.Verb == "PUT" {
		req.Header.SetMethod(or.Verb)
		req.SetBody(or.Body)
	} else if or.Verb == "GET" || or.Verb == "DELETE" || or.Verb == "HEAD" {
		req.Header.SetMethod(or.Verb)
		// don't need to set the body
	} else {

		logging.Error(fmt.Sprintf("Unsupported verb %s", or.Verb))
		return OriginResponse{StatusCode: 500, Body: []byte("This method is unsupported by brelay. Sorry.")}, true
	}

	req.Header.Del("Host")
	req.Header.Del("User-Agent")
	for _, header := range or.Headers {
		req.Header.SetBytesKV(header.Key, header.Value)
	}

	req.Header.Set("X-Forwarded-For", or.ClientIP)
	req.Header.Set("Real-IP", or.ClientIP)

	err := fasthttp.Do(req, resp)
	if err != nil {
		logging.Error(fmt.Sprintf("Error from upstream: %s", err.Error()))
		return OriginResponse{}, false
	}

	out := OriginResponse{StatusCode: resp.StatusCode(), Body: resp.Body()}

	resp.Header.VisitAll(func(k, v []byte) {
		out.Headers = append(out.Headers, Header{k, v})
	})

	return out, true
}

func Run() {

	for _, tfrontend := range config.Get().Frontends {

		go func(lfrontend config.Frontend) {

			// default round-robin
			rrCx := uint64(0)

			// tls enabled?
			tls := len(lfrontend.TLSKeyPath) > 0 && len(lfrontend.TLSCertPath) > 0

			primaryHandler := func(ctx *fasthttp.RequestCtx) {

				// first, select the host based on incoming ( first pass, naive iteration )
				host := config.Host{}
				{
					requestHostname := string(ctx.Request.Host())
					foundHost := false
					for _, thost := range lfrontend.Hosts {
						if requestHostname == thost.Hostname || thost.Hostname == "" {
							host = thost
							foundHost = true
						}
					}
					if !foundHost {
						logging.Error(fmt.Sprintf("Unable to match [%s] with any hostnames", requestHostname))
						ctx.Error("Couldn't find it, Sorry!", fasthttp.StatusNotFound)
						return
					}
				}

				requestUri := string(ctx.RequestURI())
				path := config.Path{}
				{
					foundPath := false
					for _, tpath := range host.Paths {
						if strings.HasPrefix(requestUri, tpath.Path) {
							path = tpath
							foundPath = true
						}
					}
					if !foundPath {
						logging.Error(fmt.Sprintf("Unable to match [%s] with any paths in host [%s]", requestUri, host.Hostname))
						for _, tpath := range host.Paths {
							logging.Error(fmt.Sprintf(" |- Found [%s]", tpath.Path))
						}
						ctx.Error("Couldn't find it, Sorry!", fasthttp.StatusNotFound)
						return
					}

					// Strip path prefix from initial request URI
					// todo: inefficient - clean this up later
					requestUri = strings.Replace(requestUri, path.Path, "", 1)
					if !strings.HasPrefix(requestUri, "/") {
						requestUri = "/" + requestUri
					}
				}

				// grab initial client ip
				clientIP := ctx.RemoteIP().String()

				validResponse := false

				proxyRequest := OriginRequest{Uri: requestUri, ClientIP: clientIP}
				if ctx.IsGet() {
					proxyRequest.Verb = "GET"
				}
				if ctx.IsPost() {
					proxyRequest.Verb = "POST"
					proxyRequest.Body = ctx.PostBody()
				}

				ctx.Request.Header.VisitAllInOrder(func(k, v []byte) {
					proxyRequest.Headers = append(proxyRequest.Headers, Header{k, v})
					ctx.Response.Header.DelBytes(k)
				})

				var proxyResponse OriginResponse
				var lbackend config.Backend

				for !validResponse {

					localRr := int(rrCx % uint64(len(path.Backends)))
					rrCx++

					lbackend = path.Backends[localRr]

					proxyResponse, validResponse = getFromOrigin(lbackend, proxyRequest)

					if !validResponse {
						// let's wait a default value of one second
						time.Sleep(time.Second)
					}
				}

				if proxyResponse.StatusCode > 199 {

					for _, header := range proxyResponse.Headers {

						if proxyResponse.StatusCode == 301 || proxyResponse.StatusCode == 302 {
							// handle redirection bullshit
							sk := strings.ToLower(string(header.Key))
							sv := string(header.Value)
							if strings.HasPrefix(sk, "location") {

								// rewrite Location header if the idiot backend is sending its own port
								beHost := fmt.Sprintf(":%d", lbackend.Port)
								feHost := fmt.Sprintf(":%d", lfrontend.BindPort)
								sv = strings.ReplaceAll(sv, beHost, feHost)
								if len(lbackend.Hostname) > 0 && len(host.Hostname) > 0 {
									sv = strings.ReplaceAll(sv, lbackend.Hostname, host.Hostname)
								}
								if tls && strings.HasPrefix(sv, "http:") {
									sv = strings.Replace(sv, "http://", "https://", 1)
								}

								logging.Access(fmt.Sprintf("handling redirect, sending to %s\n", sv))

								ctx.Response.Header.SetBytesK(header.Key, sv)
							} else {
								ctx.Response.Header.SetBytesKV(header.Key, header.Value)
							}
						} else {
							ctx.Response.Header.SetBytesKV(header.Key, header.Value)
						}
					}
					ctx.Response.Header.Set("X-Forwarded-For", clientIP)
					ctx.Response.Header.Set("Real-IP", clientIP)
					ctx.SetStatusCode(proxyResponse.StatusCode)
					ctx.Response.SetBodyRaw(proxyResponse.Body)
					ctx.Response.SetConnectionClose()

					logging.Access(fmt.Sprintf("[%s]  %d  %s  %s", clientIP, proxyResponse.StatusCode, proxyRequest.Verb, proxyRequest.Uri))

					return
				}

				ctx.SetContentType("text/plain")
				ctx.SetStatusCode(fasthttp.StatusOK)

				//ctx.Error("not found", fasthttp.StatusNotFound)

			}

			feHost := fmt.Sprintf(":%d", lfrontend.BindPort)

			server := &fasthttp.Server{
				Handler:            primaryHandler,
				Name:               "BRelay v0.0.2",
				MaxRequestBodySize: (1024 * 1024 * 32),
			}

			if tls {
				fmt.Printf("Now listening (TLS) on %s\n", feHost)
				server.ListenAndServeTLS(feHost, lfrontend.TLSCertPath, lfrontend.TLSKeyPath)
			} else {
				fmt.Printf("Now listening (PLAIN) on %s\n", feHost)
				server.ListenAndServe(feHost)
			}
		}(tfrontend)
	}

	// wait forever until termination
	time.Sleep(time.Hour * 999999)

}
