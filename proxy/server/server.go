package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/netsec-ethz/scion-apps/lib/shttp"
)

var static = flag.String("dir", ".", "directory containing the static content to serve")

func main() {

	flag.Parse()

	log.Println("Started listening on :443")

	logHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// log request
			remote := r.Header.Get("X-Forwarded-For")
			if remote != "" {
				log.Printf("Request from %s (proxy: %s) for resource %s", remote, r.RemoteAddr, r.URL.Path)
			} else {
				log.Printf("Request from %s for resource %s", r.RemoteAddr, r.URL.Path)
			}
			// call next handler
			h.ServeHTTP(w, r)
		})
	}

	http.Handle("/", logHandler(http.StripPrefix("/", http.FileServer(http.Dir(*static)))))
	log.Fatal(shttp.ListenAndServeSCION("17-ffaa:1:c2,[127.0.8.1]:443", "cert.pem", "key.pem", nil))
}
