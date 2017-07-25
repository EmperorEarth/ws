package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/mailru/easygo/netpoll"

	"net/http"
	_ "net/http/pprof"
)

var (
	addr      = flag.String("listen", "localhost:8080", "address to bind to")
	debug     = flag.String("pprof", "localhost:3333", "address for pprof http")
	workers   = flag.Int("workers", 128, "max workers count")
	queue     = flag.Int("queue", 1, "workers task queue size")
	ioTimeout = flag.Duration("io_timeout", time.Millisecond*100, "i/o operations timeout")
)

func main() {
	flag.Parse()

	// Initialize netpoll instance. We will use it to be noticed about peer
	// connection has some bytes to read.
	poller, err := netpoll.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", ln.Addr().String())

	if x := *debug; x != "" {
		go func() {
			log.Printf("starting pprof server on %s", x)
			log.Printf("pprof server error: %v", http.ListenAndServe(x, nil))
		}()
	}

	// Create new pool with N workers and 1 running worker.
	var (
		exit = make(chan struct{})
		pool = NewPool(1, *workers, *queue)
		chat = NewChat(pool)
	)

	handle := func(conn net.Conn) {
		safeConn := deadliner{conn, *ioTimeout}

		// Zero-copy upgrade to WebSocket connection.
		hs, err := ws.Upgrade(safeConn)
		if err != nil {
			log.Printf("%s: upgrade error: %v", nameConn(conn), err)
			conn.Close()
			return
		}

		log.Printf("%s: established websocket connection: %+v", nameConn(conn), hs)

		user := chat.Register(safeConn)

		// Create read events descriptor for conn.
		desc := netpoll.Must(netpoll.HandleRead(conn))

		// Start receiving events from conn.
		poller.Start(desc, func(ev netpoll.Event) {
			if ev&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
				poller.Stop(desc)
				chat.Remove(user)
				return
			}
			pool.Schedule(func() {
				if err := user.Receive(); err != nil {
					poller.Stop(desc)
					chat.Remove(user)
				}
			})
		})
	}

	// Create accept events descriptor for listener.
	acceptDesc := netpoll.Must(netpoll.HandleListener(
		ln, netpoll.EventRead|netpoll.EventOneShot,
	))
	var (
		accept = make(chan error, 1)
	)
	poller.Start(acceptDesc, func(e netpoll.Event) {
		err := pool.ScheduleTimeout(time.Millisecond, func() {
			conn, err := ln.Accept()
			if err != nil {
				accept <- err
				return
			}

			accept <- nil
			handle(conn)
		})
		if err == nil {
			err = <-accept
		}
		if err != nil {
			if err != ErrScheduleTimeout {
				goto cooldown
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				goto cooldown
			}

			log.Fatalf("accept error: %v", err)

		cooldown:
			delay := 5 * time.Millisecond
			log.Printf("accept error: %v; retrying in %s", err, delay)
			time.Sleep(delay)
		}

		poller.Resume(acceptDesc)
	})

	<-exit
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
