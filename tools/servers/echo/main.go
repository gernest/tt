package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
)

func main() {
	addr := flag.String("addr", ":8081", "address to use")
	flag.Parse()
	serve(*addr)
}

func serve(addr string) {
	ls, err := net.Listen("tcp", ":8000")
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
