package main

import (
	"flag"
	"log"
	"net"
	"time"

	"stash.mail.ru/scm/ego/easygo.git/util/pool"

	"github.com/gobwas/ws"
	"github.com/mailru/easygo/netpoll"
)

var (
	addr    = flag.String("listen", ":8080", "address to bind to")
	workers = flag.Int("workers", 128, "max workers count")
)

func main() {
	flag.Parse()

	poller, err := netpoll.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				delay := 5 * time.Millisecond
				log.Printf("accept error: %v; retrying in %s", err, delay)
				time.Sleep(delay)
				continue
			}
			log.Fatal(err)
		}

		name := nameConn(conn)

		hs, err := ws.Upgrade(conn)
		if err != nil {
			log.Printf("%s: upgrade error: %v", name, err)
			conn.Close()
			continue
		}
		log.Printf("%s: established websocket connection", name)

		// Create read events descriptor for conn.
		desc := netpoll.Must(netpoll.HandleRead(conn))

		// Start receiving events from conn.
		poller.Start(desc, func(ev netpoll.Event) {
			if ev&netpoll.EventReadHup != 0 {
				log.Printf("%s: peer has closed the read end of connection", name)
				poller.Stop(desc)
				conn.Close()
				return
			}

			log.Printf("%s: scheduling next read", name)
			pool.Schedule(func() {
				// todo
			})
		})
	}
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
