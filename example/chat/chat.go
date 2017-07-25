package main

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type User struct {
	io   sync.Mutex
	conn io.ReadWriteCloser

	id   uint
	name string
	chat *Chat
}

func (u *User) readRequest() (*Request, error) {
	u.io.Lock()
	defer u.io.Unlock()

	h, r, err := wsutil.NextReader(u.conn, ws.StateServerSide)
	if err != nil {
		return nil, err
	}
	if h.OpCode.IsControl() {
		return nil, wsutil.ControlHandler(u.conn, ws.StateServerSide)(h, r)
	}

	req := &Request{}
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(req); err != nil {
		return nil, err
	}

	return req, nil
}

func (u *User) writeErrorTo(req *Request, err Object) error {
	return u.write(Error{
		ID:    req.ID,
		Error: err,
	})
}

func (u *User) writeResultTo(req *Request, result Object) error {
	return u.write(Response{
		ID:     req.ID,
		Result: result,
	})
}

func (u *User) writeNotice(method string, params Object) error {
	return u.write(Request{
		Method: method,
		Params: params,
	})
}

func (u *User) write(x interface{}) error {
	w := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	u.io.Lock()
	defer u.io.Unlock()

	if err := encoder.Encode(x); err != nil {
		return err
	}

	return w.Flush()
}

func (u *User) writeRaw(p []byte) error {
	u.io.Lock()
	defer u.io.Unlock()

	_, err := u.conn.Write(p)

	return err
}

func (u *User) Receive() error {
	req, err := u.readRequest()
	if err != nil {
		u.conn.Close()
		return err
	}
	if req == nil {
		// Handled some control message.
		return nil
	}
	switch req.Method {
	case "rename":
		name := req.Params["name"]
		if ok := u.chat.Rename(u, name); !ok {
			u.writeErrorTo(req, Object{
				"error": "error",
			})
			return nil
		}
		return u.writeResultTo(req, nil)
	case "publish":
		req.Params["author"] = u.name
		u.chat.Broadcast("publish", req.Params)
	default:
		u.writeErrorTo(req, Object{
			"error": "not implemented",
		})
	}
	return nil
}

type Chat struct {
	mu  sync.RWMutex
	seq uint
	us  []*User
	ns  map[string]*User

	pool *Pool
	out  chan []byte
}

func NewChat(pool *Pool) *Chat {
	chat := &Chat{
		pool: pool,
		ns:   make(map[string]*User),
		out:  make(chan []byte, 1),
	}

	go chat.writer()

	return chat
}

func (c *Chat) Register(conn net.Conn) *User {
	user := &User{
		chat: c,
		conn: conn,
	}

	c.mu.Lock()
	{
		user.id = c.seq
		user.name = c.randName()

		c.us = append(c.us, user)
		c.ns[user.name] = user

		c.seq++
	}
	c.mu.Unlock()

	user.writeNotice("hello", Object{
		"name": user.name,
	})
	c.Broadcast("greet", Object{
		"name": user.name,
	})

	return user
}

func (c *Chat) Remove(user *User) {
	c.mu.Lock()
	removed := c.remove(user)
	c.mu.Unlock()

	if removed {
		c.Broadcast("goodbye", Object{
			"name": user.name,
		})
	}
}

func (c *Chat) Rename(user *User, name string) (ok bool) {
	var prev string
	c.mu.Lock()
	{
		if _, has := c.ns[name]; !has {
			ok = true
			prev, user.name = user.name, name
			delete(c.ns, prev)
			c.ns[name] = user
		}
	}
	c.mu.Unlock()

	if !ok {
		return false
	}

	c.Broadcast("rename", Object{
		"prev": prev,
		"name": name,
	})

	return true
}

func (c *Chat) Broadcast(method string, params Object) error {
	var buf bytes.Buffer

	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	r := Request{Method: method, Params: params}
	if err := encoder.Encode(r); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	c.out <- buf.Bytes()

	return nil
}

func (c *Chat) writer() {
	for bts := range c.out {
		c.mu.RLock()
		us := c.us
		c.mu.RUnlock()

		for _, u := range us {
			u := u // For closure.
			c.pool.Schedule(func() {
				u.writeRaw(bts)
			})
		}
	}
}

// mutex must be held.
func (c *Chat) remove(user *User) bool {
	if _, has := c.ns[user.name]; !has {
		return false
	}

	delete(c.ns, user.name)

	i := sort.Search(len(c.us), func(i int) bool {
		return c.us[i].id >= user.id
	})
	if i >= len(c.us) {
		panic("chat: inconsistent state")
	}
	without := make([]*User, len(c.us)-1)
	copy(without[:i], c.us[:i])
	copy(without[i:], c.us[i+1:])
	c.us = without

	return true
}

func (c *Chat) randName() string {
	var suffix string
	for {
		name := animals[rand.Intn(len(animals))] + suffix
		if _, has := c.ns[name]; !has {
			return name
		}
		suffix += strconv.Itoa(rand.Intn(10))
	}
	return ""
}

type deadliner struct {
	net.Conn
	t time.Duration
}

func (d deadliner) Write(p []byte) (int, error) {
	if err := d.Conn.SetWriteDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Write(p)
}

func (d deadliner) Read(p []byte) (int, error) {
	if err := d.Conn.SetReadDeadline(time.Now().Add(d.t)); err != nil {
		return 0, err
	}
	return d.Conn.Read(p)
}
