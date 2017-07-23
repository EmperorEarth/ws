package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
)

var (
	addr    = flag.String("listen", "localhost:8080", "address to bind to")
	workers = flag.Int("workers", 128, "max workers count")
	queue   = flag.Int("queue", 1, "workers task queue size")
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

	// Create new pool with N workers.
	pool := NewPool(1, *workers, *queue)

	var clients []net.Conn
	var mu sync.RWMutex
	notice := make(chan Request, 1)
	go func() {
		for req := range notice {
			mu.RLock()
			cs := clients
			mu.RUnlock()

			req := req
			for _, conn := range cs {
				conn := conn
				err := pool.ScheduleTimeout(time.Millisecond, func() {
					//register
					dest := wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
					encoder := json.NewEncoder(dest)
					err = encoder.Encode(req)
					if err == nil {
						err = dest.Flush()
					}
					if err != nil {
						log.Printf("%s: send notice error: %v", nameConn(conn), err)
					}
				})
				if err != nil {
					log.Printf("%s: schedule notice error: %v; miss", nameConn(conn), err)
				}
			}
		}
	}()

	handle := func(conn net.Conn) {
		name := nameConn(conn)

		// Zero-copy upgrade to WebSocket connection.
		hs, err := ws.Upgrade(conn)
		if err != nil {
			log.Printf("%s: upgrade error: %v", name, err)
			conn.Close()
			return
		}

		log.Printf("%s: established websocket connection: %+v", name, hs)
		mu.Lock()
		clients = append(clients, conn)
		mu.Unlock()

		// Create read events descriptor for conn.
		desc := netpoll.Must(netpoll.HandleRead(conn))

		// Start receiving events from conn.
		poller.Start(desc, func(ev netpoll.Event) {
			log.Printf("%s: netpoll event: %s", name, ev)

			if ev&netpoll.EventReadHup != 0 {
				log.Printf("%s: peer has closed its read end, closing connection", name)
				poller.Stop(desc)
				conn.Close()
				desc.Close()
				return
			}

			pool.Schedule(func() {
				header, err := ws.ReadHeader(conn)
				if err != nil {
					log.Printf("%s: read frame header error: %v", name, err)
					return
				}
				src := wsutil.NewCipherReader(
					io.LimitReader(conn, header.Length),
					header.Mask,
				)
				//bts, err := ioutil.ReadAll(src)
				//if err != nil {
				//	log.Printf("%s: read all error: %v", name, err)
				//	return
				//}
				//log.Printf("%s: %#q", name, bts)
				//decoder := json.NewDecoder(bytes.NewReader(bts))
				decoder := json.NewDecoder(src)
				var r Request
				if err := decoder.Decode(&r); err != nil {
					log.Printf("%s: unmarshal request error: %v", name, err)
					poller.Stop(desc)
					conn.Close()
					desc.Close()
					return
				}
				log.Printf("%s: request: %+v", name, r)

				switch r.Method {
				case "hello":
					dest := wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
					encoder := json.NewEncoder(dest)
					err = encoder.Encode(Response{
						ID: r.ID,
						Result: map[string]string{
							"name": "ws",
						},
					})
					if err == nil {
						err = dest.Flush()
					}
					if err != nil {
						log.Printf("%s: send response error: %v", name, err)
						return
					}

				case "publish":
					r.ID = 0
					notice <- r
				}

				//register
			})
		})
	}

	accept := make(chan error, 1)
	for {
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
			if err == ErrScheduleTimeout {
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
	}
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
