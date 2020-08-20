package server

import (
	"fmt"
	"github.com/caeril/brelay/config"
	"github.com/valyala/fasthttp"
	"log"
	"strings"
	"time"
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
	Verb    string
	Uri     string
	Headers []Header
	Body    []byte
}

func getFromOrigin(backend config.Backend, or OriginRequest) (OriginResponse, bool) {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(fmt.Sprintf("http://%s:%d%s", backend.Host, backend.Port, or.Uri))

	if or.Verb == "POST" || or.Verb == "PUT" {
		req.Header.SetMethod(or.Verb)
		req.SetBody(or.Body)
	} else if or.Verb == "GET" || or.Verb == "DELETE" || or.Verb == "HEAD" {
		req.Header.SetMethod(or.Verb)
		// don't need to set the body
	} else {
		log.Printf("Unsupported verb %s\n", or.Verb)
		return OriginResponse{StatusCode: 500, Body: []byte("This method is unsupported by brelay. Sorry.")}, true
	}

	req.Header.Del("Host")
	req.Header.Del("User-Agent")
	for _, header := range or.Headers {
		req.Header.SetBytesKV(header.Key, header.Value)
	}

	err := fasthttp.Do(req, resp)
	if err != nil {
		log.Printf("ERROR from upstream: %s\n", err)
		return OriginResponse{}, false
	}

	out := OriginResponse{StatusCode: resp.StatusCode(), Body: resp.Body()}

	resp.Header.VisitAll(func(k, v []byte) {
		out.Headers = append(out.Headers, Header{k, v})
	})

	return out, true
}

func Run() {

	for _, frontend := range config.Get().Frontends {

		go func(lfrontend config.Frontend) {

			// default round-robin
			rrCx := 0

			primaryHandler := func(ctx *fasthttp.RequestCtx) {

				validResponse := false

				proxyRequest := OriginRequest{Uri: string(ctx.RequestURI())}
				if ctx.IsGet() {
					proxyRequest.Verb = "GET"
				}
				if ctx.IsPost() {
					proxyRequest.Verb = "POST"
					proxyRequest.Body = ctx.PostBody()
				}

				log.Printf("%s %s\n", proxyRequest.Verb, proxyRequest.Uri)

				ctx.Request.Header.VisitAllInOrder(func(k, v []byte) {
					proxyRequest.Headers = append(proxyRequest.Headers, Header{k, v})
					ctx.Response.Header.DelBytes(k)
				})

				var proxyResponse OriginResponse
				var lbackend config.Backend

				for !validResponse {

					// Select the target backend - for now just grab the first one
					lbackend = lfrontend.Backends[rrCx]
					rrCx++
					if rrCx >= len(lfrontend.Backends) {
						rrCx = 0
					}

					fmt.Printf("dispatching to %s\n", lbackend)

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

								fmt.Printf("handling redirect, sending to %s\n", sv)

								ctx.Response.Header.SetBytesK(header.Key, sv)
							} else {
								ctx.Response.Header.SetBytesKV(header.Key, header.Value)
							}
						} else {
							ctx.Response.Header.SetBytesKV(header.Key, header.Value)
						}
					}
					ctx.SetStatusCode(proxyResponse.StatusCode)
					ctx.Response.AppendBody(proxyResponse.Body)
					return
				}

				ctx.SetContentType("text/plain")
				ctx.SetStatusCode(fasthttp.StatusOK)

				//ctx.Error("not found", fasthttp.StatusNotFound)

			}

			feHost := fmt.Sprintf(":%d", lfrontend.BindPort)

			fmt.Printf("Now listening on %s\n", feHost)
			fasthttp.ListenAndServe(feHost, primaryHandler)

		}(frontend)
	}

	// wait forever until termination
	time.Sleep(time.Hour * 999999)

}
