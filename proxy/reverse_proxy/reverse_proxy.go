package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/netsec-ethz/scion-apps/lib/shttp"
)

var addr = flag.String("local", "", "local SCION address to bind to")

func main() {

	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		remote := r.Header.Get("X-Forwarded-For")
		if remote != "" {
			log.Printf("Request from %s (proxy: %s) for resource %s", remote, r.RemoteAddr, r.URL.Path)
		} else {
			log.Printf("Request from %s for resource %s", r.RemoteAddr, r.URL.Path)
		}
		client := http.Client{
			// don't handle redirects
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://www.scion-architecture.net%s", r.URL.Path), nil)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		// add client hop to X-Forwarded-For header
		req.Header.Add("X-Forwarded-For", r.RemoteAddr)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprint(w, err)
		}
		// copy original response headers to reply
		for name, value := range resp.Header {
			for _, v := range value {
				w.Header().Add(name, v)
			}
		}

		// in case of redirect, re-write location header to point to this reverse proxy
		loc := resp.Header.Get("Location")
		if loc != "" {
			w.Header().Set("Location", strings.Replace(loc,
				"https://www.scion-architecture.net", "http://localhost:7070", -1))
		}

		// add header to show this site was served over SCION
		w.Header().Set("Served-Over", "SCION")

		// write original status code to reply
		w.WriteHeader(resp.StatusCode)

		// return response to client
		io.Copy(w, resp.Body)
	})

	log.Fatal(shttp.ListenAndServeSCION(*addr, "cert.pem", "key.pem", nil))
}
