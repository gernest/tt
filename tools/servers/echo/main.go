package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
)

func main() {
	addr := flag.String("addr", ":8081", "address to use")
	cert := flag.String("cert", "./bin/certs/echo.test/cert.pem", "tls cert")
	key := flag.String("key", "./bin/certs/echo.test/key.pem", "tls key")
	flag.Parse()
	serve(*addr, *cert, *key)
}

func serve(addr, cert, key string) {
	x, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		log.Fatal("Failed to load certs", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{x},
	}
	ls, err := tls.Listen("tcp", ":8081", &config)
	if err != nil {
		log.Fatal(err)
	}
	defer ls.Close()
	fmt.Println("started echo service at ", ls.Addr())
	for {
		a, err := ls.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func(conn net.Conn) {
			d := json.NewDecoder(conn)
			e := json.NewEncoder(conn)
			var n int64
			for {
				err := d.Decode(&n)
				if err != nil {
					log.Fatalf("failed to decode err=%v\n", err)
				}
				err = e.Encode(n + 1)
				if err != nil {
					log.Fatalf("failed to encode err=%v\n", err)
				}
			}
		}(a)
	}
}

type w struct{}

func (w) Write(b []byte) (int, error) {
	log.Printf("==> [%d] read %q\n", len(b), string(b))
	return len(b), nil
}
