package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/ahmetb/go-httpbin"
)

func main() {
	addr := flag.String("addr", ":8080", "address to use")
	flag.Parse()
	log.Println("starting httpbbin service at ", *addr)
	log.Fatal(http.ListenAndServe(*addr, httpbin.GetMux()))
}
