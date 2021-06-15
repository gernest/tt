package main

import (
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

var upload, download int64

func formatSpeed(v float64) string {
	return fmt.Sprintf("%.2fkb/s", v/1024)
}

func main() {
	addr := flag.String("addr", ":8081", "address to use")
	cert := flag.String("cert", "./bin/certs/rando.test/cert.pem", "tls cert")
	key := flag.String("key", "./bin/certs/rando.test/key.pem", "tls key")
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
	ls, err := tls.Listen("tcp", ":8082", &config)
	if err != nil {
		log.Fatal(err)
	}
	defer ls.Close()
	fmt.Println("started rando server service at ", ls.Addr())
	for {
		a, err := ls.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func(conn net.Conn) {
			halt := make(chan error, 1)
			defer conn.Close()
			go func(h chan error) {
				_, err := io.Copy(ioutil.Discard, conn)
				halt <- err
			}(halt)
			go func(h chan error) {
				_, err := io.Copy(conn, rand.Reader)
				halt <- err
			}(halt)
			<-halt
		}(a)
	}
}
