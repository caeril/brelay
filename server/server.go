package server

import (
	"fmt"
	"github.com/caeril/brelay/config"
	"github.com/valyala/fasthttp"
	"log"
	"strings"
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

func getFromOrigin(or OriginRequest) OriginResponse {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI("http://127.0.0.1:8001" + or.Uri)

	if or.Verb == "POST" {
		req.Header.SetMethod(or.Verb)
		req.SetBody(or.Body)
	}

	req.Header.Del("Host")
	req.Header.Del("User-Agent")
	for _, header := range or.Headers {
		req.Header.SetBytesKV(header.Key, header.Value)
	}

	err := fasthttp.Do(req, resp)
	if err != nil {
		log.Printf("ERROR from upstream: %s\n", err)
	}

	out := OriginResponse{StatusCode: resp.StatusCode(), Body: resp.Body()}

	resp.Header.VisitAll(func(k, v []byte) {
		out.Headers = append(out.Headers, Header{k, v})
	})

	return out
}

func Run() {

	primaryHandler := func(ctx *fasthttp.RequestCtx) {

		// Select the target backend - for now just grab the first one
		backend := config.Get().Backends[0]

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

		proxyResponse := getFromOrigin(proxyRequest)

		if proxyResponse.StatusCode > 199 {

			for _, header := range proxyResponse.Headers {

				if proxyResponse.StatusCode == 302 {
					// handle redirection bullshit
					sk := strings.ToLower(string(header.Key))
					sv := string(header.Value)
					if strings.HasPrefix(sk, "location") {

						// rewrite Location header if the idiot backend is sending its own port
						beHost := fmt.Sprintf(":%d", backend.Port)
						feHost := fmt.Sprintf(":%d", config.Get().BindPort)
						sv = strings.ReplaceAll(sv, beHost, feHost)

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

	feHost := fmt.Sprintf(":%d", config.Get().BindPort)

	fmt.Printf("Now listening on %s\n", feHost)
	fasthttp.ListenAndServe(feHost, primaryHandler)

}
