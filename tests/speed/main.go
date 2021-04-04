package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync/atomic"
	"time"
)

var upload, download int64

func formatSpeed(v float64) string {
	return fmt.Sprintf("%.2fkb/s", v/1024)
}

func main() {
	addr := flag.String("addr", ":8000", "address to use")
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
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
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
			go io.Copy(ioutil.Discard, conn)
			go io.Copy(conn, rand.Reader)
			<-ctx.Done()
		}(a)
	}
}

func client(addr string) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go func() {
		log.Println("reporting starts")
		d := time.Second
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				up := atomic.SwapInt64(&upload, 0)
				down := atomic.SwapInt64(&download, 0)

				upspeed := float64(up) / d.Seconds()
				downspeed := float64(down) / d.Seconds()
				log.Printf("down => %v upload => %s", formatSpeed(downspeed), formatSpeed(upspeed))
			}
		}
	}()
	go func() {
		ls, err := net.Dial("tcp", ":8000")
		if err != nil {
			log.Fatal(err)
		}
		defer ls.Close()
		go func() {
			<-ctx.Done()
			ls.Close()
		}()
		_, err = io.Copy(w{func(i int) {
			atomic.AddInt64(&download, int64(i))
		}}, ls)
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		ls, err := net.Dial("tcp", ":8000")
		if err != nil {
			log.Fatal(err)
		}
		defer ls.Close()
		go func() {
			<-ctx.Done()
			ls.Close()
		}()
		_, err = io.Copy(u{
			fn: func(i int) {
				atomic.AddInt64(&upload, int64(i))
			},
			w: ls,
		}, rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
}

type w struct {
	fn func(int)
}

func (w w) Write(b []byte) (int, error) {
	w.fn(len(b))
	return len(b), nil
}

type u struct {
	fn func(int)
	w  io.Writer
}

func (u u) Write(b []byte) (int, error) {
	n, err := u.w.Write(b)
	u.fn(n)
	return n, err
}
