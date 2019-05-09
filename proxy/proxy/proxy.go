package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/lucas-clemente/quic-go"
	"github.com/netsec-ethz/scion-apps/lib/scionutil"
	"github.com/netsec-ethz/scion-apps/lib/shttp"
	slog "github.com/scionproto/scion/go/lib/log"
)

var addr = flag.String("a", ":9090", "port the proxy is listening on")

func getClient() *http.Client {
	laddr, err := scionutil.GetLocalhost()
	if err != nil {
		log.Fatal(err)
	}

	c := &http.Client{
		Transport: &shttp.Transport{
			LAddr: laddr,
			QuicConfig: &quic.Config{
				IdleTimeout:      1 * time.Second,
				HandshakeTimeout: 1 * time.Second,
			},
		},
	}
	return c
}

func main() {

	flag.Parse()

	//disable SCION logging
	slog.Root().SetHandler(slog.DiscardHandler())

	scionHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// try to resolve the request to a SCION-enabled server
			_, _, err := scionutil.GetHostByName(r.Host)
			if err == nil {
				log.Printf("Serving %s request for %s%s over SCION\n", r.Method, r.Host, r.URL.Path)
				client := getClient()
				defer client.Transport.(*shttp.Transport).Close()
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s%s", r.Host, r.URL.Path), nil)
				if err != nil {
					fmt.Fprint(w, err)
					return
				}

				// add X-Forwarded-For header
				req.Header.Add("X-Forwarded-For", r.RemoteAddr)
				resp, respErr := client.Do(req)
				if respErr != nil {
					fmt.Fprint(w, respErr)
					return
				}
				defer resp.Body.Close()

				// copy original response headers to reply
				for name, value := range resp.Header {
					for _, v := range value {
						w.Header().Add(name, v)
					}
				}

				// return response to client
				io.Copy(w, resp.Body)
			} else {
				// else proxy the request to regular HTTP(S) server
				log.Printf("Serving %s request for %s%s over TCP\n", r.Method, r.Host, r.URL.Path)
				h.ServeHTTP(w, r)
			}
		})
	}

	// proxy for regular HTTP(S) traffic implemented as http.Handler
	proxy := goproxy.NewProxyHttpServer()

	log.Printf("Started Proxy on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, scionHandler(proxy)))
}
