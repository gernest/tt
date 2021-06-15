package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
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
	addr := flag.String("addr", ":5555", "address to use")
	cert := flag.String("cert", "./bin/certs/rando.test/cert.pem", "tls cert")
	key := flag.String("key", "./bin/certs/rando.test/key.pem", "tls key")
	duration := flag.Duration("duration", 2*time.Second, "total time of running the client")

	flag.Parse()
	client(*addr, *cert, *key, *duration)
	upspeed := float64(upload) / duration.Seconds()
	downspeed := float64(download) / duration.Seconds()
	log.Printf("down => %v upload => %s", formatSpeed(downspeed), formatSpeed(upspeed))
}

func client(addr, cert, key string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	x, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		log.Fatal("Failed to load certs", err)
	}
	config := tls.Config{
		ServerName:   "rando.test",
		Certificates: []tls.Certificate{x},
	}
	go func() {
		ls, err := tls.Dial("tcp", ":5555", &config)
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
		ls, err := net.Dial("tcp", ":5555")
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
