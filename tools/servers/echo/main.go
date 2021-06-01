package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func main() {
	addr := flag.String("addr", ":8081", "address to use")
	mode := flag.String("mode", "server", "client or server")
	flag.Parse()
	switch *mode {
	case "server":
		serve(*addr)
	case "client":
		client(*addr)
	}
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

func client(addr string) {
	ls, err := net.Dial("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}
	defer ls.Close()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	go func() {
		io.Copy(w{}, ls)
	}()
	go func() {
		tick := time.NewTicker(5 * time.Millisecond)
		e := json.NewEncoder(ls)
		var n int64
		for {
			<-tick.C
			n++
			e.Encode(n)
		}
	}()
	<-ctx.Done()
}

type w struct{}

func (w) Write(b []byte) (int, error) {
	log.Printf("==> [%d] read %q\n", len(b), string(b))
	return len(b), nil
}
