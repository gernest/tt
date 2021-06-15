package main

import (
	"crypto/rand"
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
	addr := flag.String("addr", ":8000", "address to use")
	flag.Parse()
	serve(*addr)
}

func serve(addr string) {
	ls, err := net.Listen("tcp", ":8082")
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
